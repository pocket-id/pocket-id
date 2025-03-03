package bootstrap

import (
	"log"
	"net"
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

func initRouter(db *gorm.DB, appConfigService *service.AppConfigService) {
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
		log.Fatalf("Unable to create email service: %s", err)
	}

	geoLiteService := service.NewGeoLiteService()
	auditLogService := service.NewAuditLogService(db, appConfigService, emailService, geoLiteService)
	jwtService := service.NewJwtService(appConfigService)
	webauthnService := service.NewWebAuthnService(db, jwtService, auditLogService, appConfigService)
	userService := service.NewUserService(db, jwtService, auditLogService, emailService, appConfigService)
	customClaimService := service.NewCustomClaimService(db)
	oidcService := service.NewOidcService(db, jwtService, appConfigService, auditLogService, customClaimService)
	testService := service.NewTestService(db, appConfigService, jwtService)
	userGroupService := service.NewUserGroupService(db, appConfigService)
	ldapService := service.NewLdapService(db, appConfigService, userService, userGroupService)

	rateLimitMiddleware := middleware.NewRateLimitMiddleware()

	// Setup global middleware
	r.Use(middleware.NewCorsMiddleware().Add())
	r.Use(middleware.NewErrorHandlerMiddleware().Add())
	r.Use(rateLimitMiddleware.Add(rate.Every(time.Second), 60))
	r.Use(middleware.NewJwtAuthMiddleware(jwtService, true).Add(false))

	job.RegisterLdapJobs(ldapService, appConfigService)
	job.RegisterDbCleanupJobs(db)

	// Initialize middleware for specific routes
	jwtAuthMiddleware := middleware.NewJwtAuthMiddleware(jwtService, false)
	fileSizeLimitMiddleware := middleware.NewFileSizeLimitMiddleware()

	// Set up API routes
	apiGroup := r.Group("/api")
	controller.NewWebauthnController(apiGroup, jwtAuthMiddleware, middleware.NewRateLimitMiddleware(), webauthnService, appConfigService)
	controller.NewOidcController(apiGroup, jwtAuthMiddleware, fileSizeLimitMiddleware, oidcService, jwtService)
	controller.NewUserController(apiGroup, jwtAuthMiddleware, middleware.NewRateLimitMiddleware(), userService, appConfigService)
	controller.NewAppConfigController(apiGroup, jwtAuthMiddleware, appConfigService, emailService, ldapService)
	controller.NewAuditLogController(apiGroup, auditLogService, jwtAuthMiddleware)
	controller.NewUserGroupController(apiGroup, jwtAuthMiddleware, userGroupService)
	controller.NewCustomClaimController(apiGroup, jwtAuthMiddleware, customClaimService)

	// Add test controller in non-production environments
	if common.EnvConfig.AppEnv != "production" {
		controller.NewTestController(apiGroup, testService)
	}

	// Set up base routes
	baseGroup := r.Group("/")
	controller.NewWellKnownController(baseGroup, jwtService)

	// Get the listener
	l, err := net.Listen("tcp", common.EnvConfig.Host+":"+common.EnvConfig.Port)
	if err != nil {
		log.Fatal(err)
	}

	// Notify systemd that we are ready
	if err := systemd.SdNotify(systemd.SdNotifyReady); err != nil {
		log.Println("Unable to notify systemd that the service is ready: ", err)
		// continue to serve anyway since it's not that important
	}

	// Serve requests
	if err := r.RunListener(l); err != nil {
		log.Fatal(err)
	}
}
