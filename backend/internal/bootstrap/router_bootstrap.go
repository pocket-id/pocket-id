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

	"github.com/gin-gonic/gin"
	sloggin "github.com/samber/slog-gin"
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

func initRouter(db *gorm.DB, svc *services) utils.Service {
	runner, err := initRouterInternal(db, svc)
	if err != nil {
		slog.Error("Failed to init router", "error", err)
		os.Exit(1)
	}
	return runner
}

func initRouterInternal(db *gorm.DB, svc *services) (utils.Service, error) {
	// Set the appropriate Gin mode based on the environment
	switch common.EnvConfig.AppEnv {
	case "production":
		gin.SetMode(gin.ReleaseMode)
	case "development":
		gin.SetMode(gin.DebugMode)
	case "test":
		gin.SetMode(gin.TestMode)
	}

	// do not log these URLs
	loggerSkipPathsPrefix := []string{
		"GET /application-configuration/logo",
		"GET /application-configuration/background-image",
		"GET /application-configuration/favicon",
		"GET /_app",
		"GET /fonts",
		"GET /healthz",
		"HEAD /healthz",
	}

	r := gin.New()
	r.Use(sloggin.NewWithConfig(slog.Default(), sloggin.Config{
		Filters: []sloggin.Filter{
			func(c *gin.Context) bool {
				for _, prefix := range loggerSkipPathsPrefix {
					if strings.HasPrefix(c.Request.Method+" "+c.Request.URL.String(), prefix) {
						return false
					}
				}
				return true
			},
		},
	}))

	if !common.EnvConfig.TrustProxy {
		_ = r.SetTrustedProxies(nil)
	}

	if common.EnvConfig.TracingEnabled {
		r.Use(otelgin.Middleware(common.Name))
	}

	rateLimitMiddleware := middleware.NewRateLimitMiddleware().Add(rate.Every(time.Second), 60)

	// Setup global middleware
	r.Use(middleware.NewCorsMiddleware().Add())
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
	controller.NewUserController(apiGroup, authMiddleware, middleware.NewRateLimitMiddleware(), svc.userService, svc.appConfigService)
	controller.NewAppConfigController(apiGroup, authMiddleware, svc.appConfigService, svc.emailService, svc.ldapService)
	controller.NewAuditLogController(apiGroup, svc.auditLogService, authMiddleware)
	controller.NewUserGroupController(apiGroup, authMiddleware, svc.userGroupService)
	controller.NewCustomClaimController(apiGroup, authMiddleware, svc.customClaimService)

	// Add test controller in non-production environments
	if common.EnvConfig.AppEnv != "production" {
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
		Handler:           r,
	}

	// Set up the listener
	network := "tcp"
	addr := net.JoinHostPort(common.EnvConfig.Host, common.EnvConfig.Port)
	if common.EnvConfig.UnixSocket != "" {
		network = "unix"
		addr = common.EnvConfig.UnixSocket
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
