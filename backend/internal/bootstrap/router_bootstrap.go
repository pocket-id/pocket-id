package bootstrap

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/controller"
	"github.com/pocket-id/pocket-id/backend/internal/job"
	"github.com/pocket-id/pocket-id/backend/internal/middleware"
	"github.com/pocket-id/pocket-id/backend/internal/service"
	"github.com/pocket-id/pocket-id/backend/internal/utils/systemd"
	"golang.org/x/time/rate"
	"gorm.io/gorm"
)

// This is used to register additional controllers for tests
var registerTestControllers []func(apiGroup *gin.RouterGroup, db *gorm.DB, appConfigService *service.AppConfigService, jwtService *service.JwtService)

func initRouter(ctx context.Context, db *gorm.DB, appConfigService *service.AppConfigService) {
	err := initRouterInternal(ctx, db, appConfigService)
	if err != nil {
		log.Fatalf("failed to init router: %v", err)
	}
}

func initRouterInternal(ctx context.Context, db *gorm.DB, appConfigService *service.AppConfigService) error {
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

	// Initialize services
	emailService, err := service.NewEmailService(appConfigService, db)
	if err != nil {
		return fmt.Errorf("unable to create email service: %w", err)
	}

	geoLiteService := service.NewGeoLiteService(ctx)
	auditLogService := service.NewAuditLogService(db, appConfigService, emailService, geoLiteService)
	jwtService := service.NewJwtService(appConfigService)
	webauthnService := service.NewWebAuthnService(db, jwtService, auditLogService, appConfigService)
	userService := service.NewUserService(db, jwtService, auditLogService, emailService, appConfigService)
	customClaimService := service.NewCustomClaimService(db)
	oidcService := service.NewOidcService(db, jwtService, appConfigService, auditLogService, customClaimService)
	userGroupService := service.NewUserGroupService(db, appConfigService)
	ldapService := service.NewLdapService(db, appConfigService, userService, userGroupService)
	apiKeyService := service.NewApiKeyService(db, emailService)

	rateLimitMiddleware := middleware.NewRateLimitMiddleware().Add(rate.Every(time.Second), 60)

	// Setup global middleware
	r.Use(middleware.NewCorsMiddleware().Add())
	r.Use(middleware.NewErrorHandlerMiddleware().Add())

	scheduler, err := job.NewScheduler()
	if err != nil {
		return fmt.Errorf("failed to create job scheduler: %w", err)
	}

	err = scheduler.RegisterLdapJobs(ctx, ldapService, appConfigService)
	if err != nil {
		return fmt.Errorf("failed to register LDAP jobs in scheduler: %w", err)
	}
	err = scheduler.RegisterDbCleanupJobs(ctx, db)
	if err != nil {
		return fmt.Errorf("failed to register DB cleanup jobs in scheduler: %w", err)
	}
	err = scheduler.RegisterFileCleanupJobs(ctx, db)
	if err != nil {
		return fmt.Errorf("failed to register file cleanup jobs in scheduler: %w", err)
	}
	err = scheduler.RegisterApiKeyExpiryJob(ctx, apiKeyService, appConfigService)
	if err != nil {
		return fmt.Errorf("failed to register API key expiration jobs in scheduler: %w", err)
	}

	// Run the scheduler in a background goroutine, until the context is canceled
	go scheduler.Run(ctx)

	// Initialize middleware for specific routes
	authMiddleware := middleware.NewAuthMiddleware(apiKeyService, userService, jwtService)
	fileSizeLimitMiddleware := middleware.NewFileSizeLimitMiddleware()

	// Set up API routes
	apiGroup := r.Group("/api", rateLimitMiddleware)
	controller.NewApiKeyController(apiGroup, authMiddleware, apiKeyService)
	controller.NewWebauthnController(apiGroup, authMiddleware, middleware.NewRateLimitMiddleware(), webauthnService, appConfigService)
	controller.NewOidcController(apiGroup, authMiddleware, fileSizeLimitMiddleware, oidcService, jwtService)
	controller.NewUserController(apiGroup, authMiddleware, middleware.NewRateLimitMiddleware(), userService, appConfigService)
	controller.NewAppConfigController(apiGroup, authMiddleware, appConfigService, emailService, ldapService)
	controller.NewAuditLogController(apiGroup, auditLogService, authMiddleware)
	controller.NewUserGroupController(apiGroup, authMiddleware, userGroupService)
	controller.NewCustomClaimController(apiGroup, authMiddleware, customClaimService)

	// Add test controller in non-production environments
	if common.EnvConfig.AppEnv != "production" {
		for _, f := range registerTestControllers {
			f(apiGroup, db, appConfigService, jwtService)
		}
	}

	// Set up base routes
	baseGroup := r.Group("/", rateLimitMiddleware)
	controller.NewWellKnownController(baseGroup, jwtService)

	// Set up healthcheck routes
	// These are not rate-limited
	controller.NewHealthzController(r)

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
		return fmt.Errorf("failed to create TCP listener: %w", err)
	}

	log.Printf("Server listening on %s", srv.Addr)

	// Notify systemd that we are ready
	err = systemd.SdNotifyReady()
	if err != nil {
		log.Printf("[WARN] Unable to notify systemd that the service is ready: %v", err)
		// continue to serve anyway since it's not that important
	}

	// Start the server in a background goroutine
	go func() {
		defer listener.Close()

		// Next call blocks until the server is shut down
		srvErr := srv.Serve(listener)
		if srvErr != http.ErrServerClosed {
			log.Fatalf("Error starting app server: %v", srvErr)
		}
	}()

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
