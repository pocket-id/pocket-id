package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	sloggin "github.com/gin-contrib/slog"
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"golang.org/x/time/rate"
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/frontend"
	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/controller"
	"github.com/pocket-id/pocket-id/backend/internal/middleware"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
	"github.com/pocket-id/pocket-id/backend/internal/utils/systemd"
)

// This is used to register additional controllers for tests
var registerTestControllers []func(apiGroup *gin.RouterGroup, db *gorm.DB, svc *services)

func initRouter(db *gorm.DB, svc *services) (utils.Service, error) {
	// Set the appropriate Gin mode based on the environment
	switch common.EnvConfig.AppEnv {
	case common.AppEnvProduction:
		gin.SetMode(gin.ReleaseMode)
	case common.AppEnvDevelopment:
		gin.SetMode(gin.DebugMode)
	case common.AppEnvTest:
		gin.SetMode(gin.TestMode)
	}

	r := gin.New()
	initLogger(r)

	if !common.EnvConfig.TrustProxy {
		_ = r.SetTrustedProxies(nil)
	}

	if common.EnvConfig.TracingEnabled {
		r.Use(otelgin.Middleware(common.Name))
	}

	rateLimitMiddleware := middleware.NewRateLimitMiddleware().Add(rate.Every(time.Second), 60)

	// Setup global middleware
	r.Use(middleware.HeadMiddleware())
	r.Use(middleware.NewCacheControlMiddleware().Add())
	r.Use(middleware.NewCorsMiddleware().Add())
	r.Use(middleware.NewCspMiddleware().Add())
	r.Use(middleware.NewErrorHandlerMiddleware().Add())

	err := frontend.RegisterFrontend(r)
	if errors.Is(err, frontend.ErrFrontendNotIncluded) {
		slog.Warn("Frontend is not included in the build. Skipping frontend registration.")
	} else if err != nil {
		return nil, fmt.Errorf("failed to register frontend: %w", err)
	}

	// Initialize middleware for specific routes
	authMiddleware := middleware.NewAuthMiddleware(svc.apiKeyService, svc.userService, svc.jwtService)
	fileSizeLimitMiddleware := middleware.NewFileSizeLimitMiddleware()

	// Set up API routes
	apiGroup := r.Group("/api", rateLimitMiddleware)
	controller.NewApiKeyController(apiGroup, authMiddleware, svc.apiKeyService)
	controller.NewWebauthnController(apiGroup, authMiddleware, middleware.NewRateLimitMiddleware(), svc.webauthnService, svc.appConfigService)
	controller.NewOidcController(apiGroup, authMiddleware, fileSizeLimitMiddleware, svc.oidcService, svc.jwtService)
	controller.NewUserController(apiGroup, authMiddleware, middleware.NewRateLimitMiddleware(), svc.userService, svc.oneTimeAccessService, svc.appConfigService)
	controller.NewAppConfigController(apiGroup, authMiddleware, svc.appConfigService, svc.emailService, svc.ldapService)
	controller.NewAppImagesController(apiGroup, authMiddleware, svc.appImagesService)
	controller.NewAuditLogController(apiGroup, svc.auditLogService, authMiddleware)
	controller.NewUserGroupController(apiGroup, authMiddleware, svc.userGroupService)
	controller.NewCustomClaimController(apiGroup, authMiddleware, svc.customClaimService)
	controller.NewVersionController(apiGroup, svc.versionService)
	controller.NewScimController(apiGroup, authMiddleware, svc.scimService)
	controller.NewUserSignupController(apiGroup, authMiddleware, middleware.NewRateLimitMiddleware(), svc.userSignUpService, svc.appConfigService)

	// Add test controller in non-production environments
	if !common.EnvConfig.AppEnv.IsProduction() {
		for _, f := range registerTestControllers {
			f(apiGroup, db, svc)
		}
	}

	// Set up base routes
	baseGroup := r.Group("/", rateLimitMiddleware)
	controller.NewWellKnownController(baseGroup, svc.jwtService)

	// Set up healthcheck routes
	// These are not rate-limited
	controller.NewHealthzController(r)

	// Set up the server
	srv := &http.Server{
		MaxHeaderBytes:    1 << 20,
		ReadHeaderTimeout: 10 * time.Second,
		Handler: http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			// HEAD requests don't get matched by Gin routes, so we convert them to GET
			// middleware.HeadMiddleware will convert them back to HEAD later
			if req.Method == http.MethodHead {
				req.Method = http.MethodGet
				ctx := context.WithValue(req.Context(), middleware.IsHeadRequestCtxKey{}, true)
				req = req.WithContext(ctx)
			}

			r.ServeHTTP(w, req)
		}),
	}

	// Set up the listener
	network := "tcp"
	addr := net.JoinHostPort(common.EnvConfig.Host, common.EnvConfig.Port)
	if common.EnvConfig.UnixSocket != "" {
		network = "unix"
		addr = common.EnvConfig.UnixSocket
		os.Remove(addr) // remove dangling the socket file to avoid file-exist error
	}

	listener, err := net.Listen(network, addr) //nolint:noctx
	if err != nil {
		return nil, fmt.Errorf("failed to create %s listener: %w", network, err)
	}

	// Set the socket mode if using a Unix socket
	if network == "unix" && common.EnvConfig.UnixSocketMode != "" {
		mode, err := strconv.ParseUint(common.EnvConfig.UnixSocketMode, 8, 32)
		if err != nil {
			return nil, fmt.Errorf("failed to parse UNIX socket mode '%s': %w", common.EnvConfig.UnixSocketMode, err)
		}

		if err := os.Chmod(addr, os.FileMode(mode)); err != nil {
			return nil, fmt.Errorf("failed to set UNIX socket mode '%s': %w", common.EnvConfig.UnixSocketMode, err)
		}
	}

	// Service runner function
	runFn := func(ctx context.Context) error {
		slog.Info("Server listening", slog.String("addr", addr))

		// Start the server in a background goroutine
		go func() {
			defer listener.Close()

			// Next call blocks until the server is shut down
			srvErr := srv.Serve(listener)
			if srvErr != http.ErrServerClosed {
				slog.Error("Error starting app server", "error", srvErr)
				os.Exit(1)
			}
		}()

		// Notify systemd that we are ready
		err = systemd.SdNotifyReady()
		if err != nil {
			// Log the error only
			slog.Warn("Unable to notify systemd that the service is ready", "error", err)
		}

		// Block until the context is canceled
		<-ctx.Done()

		// Handle graceful shutdown
		// Note we use the background context here as ctx has been canceled already
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		shutdownErr := srv.Shutdown(shutdownCtx) //nolint:contextcheck
		shutdownCancel()
		if shutdownErr != nil {
			// Log the error only (could be context canceled)
			slog.Warn("App server shutdown error", "error", shutdownErr)
		}

		return nil
	}

	return runFn, nil
}

func initLogger(r *gin.Engine) {
	loggerSkipPathsPrefix := []string{
		"GET /api/application-images/logo",
		"GET /api/application-images/background",
		"GET /api/application-images/favicon",
		"GET /api/application-images/email",
		"GET /_app",
		"GET /fonts",
		"GET /healthz",
		"HEAD /healthz",
	}

	r.Use(sloggin.SetLogger(
		sloggin.WithLogger(func(_ *gin.Context, _ *slog.Logger) *slog.Logger {
			return slog.Default()
		}),
		sloggin.WithSkipper(func(c *gin.Context) bool {
			for _, prefix := range loggerSkipPathsPrefix {
				if strings.HasPrefix(c.Request.Method+" "+c.Request.URL.String(), prefix) {
					return true
				}
			}
			return false
		}),
	))
}
