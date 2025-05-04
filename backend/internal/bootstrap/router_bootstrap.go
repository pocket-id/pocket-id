package bootstrap

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
	"gorm.io/gorm"

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
		log.Fatalf("failed to init router: %v", err)
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

	r := gin.Default()
	r.Use(gin.Logger())

	rateLimitMiddleware := middleware.NewRateLimitMiddleware()

	// Setup global middleware
	r.Use(middleware.NewCorsMiddleware().Add())
	r.Use(middleware.NewErrorHandlerMiddleware().Add())
	r.Use(rateLimitMiddleware.Add(rate.Every(time.Second), 60))

	// Initialize middleware for specific routes
	authMiddleware := middleware.NewAuthMiddleware(svc.apiKeyService, svc.userService, svc.jwtService)
	fileSizeLimitMiddleware := middleware.NewFileSizeLimitMiddleware()

	// Set up API routes
	apiGroup := r.Group("/api")
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
	baseGroup := r.Group("/")
	controller.NewWellKnownController(baseGroup, svc.jwtService)

	// Set up the server
	srv := &http.Server{
		Addr:              net.JoinHostPort(common.EnvConfig.Host, common.EnvConfig.Port),
		MaxHeaderBytes:    1 << 20,
		ReadHeaderTimeout: 10 * time.Second,
		Handler:           r,
	}

	// Set up the listener
	listener, err := net.Listen("tcp", srv.Addr)
	if err != nil {
		return nil, fmt.Errorf("failed to create TCP listener: %w", err)
	}

	// Service runner function
	runFn := func(ctx context.Context) error {
		log.Printf("Server listening on %s", srv.Addr)

		// Start the server in a background goroutine
		go func() {
			defer listener.Close()

			// Next call blocks until the server is shut down
			srvErr := srv.Serve(listener)
			if srvErr != http.ErrServerClosed {
				log.Fatalf("Error starting app server: %v", srvErr)
			}
		}()

		// Notify systemd that we are ready
		err = systemd.SdNotifyReady()
		if err != nil {
			// Log the error only
			log.Printf("[WARN] Unable to notify systemd that the service is ready: %v", err)
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
			log.Printf("[WARN] App server shutdown error: %v", shutdownErr)
		}

		return nil
	}

	return runFn, nil
}
