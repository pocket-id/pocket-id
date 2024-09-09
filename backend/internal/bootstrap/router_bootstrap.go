package bootstrap

import (
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stonith404/pocket-id/backend/internal/common"
	"github.com/stonith404/pocket-id/backend/internal/controller"
	"github.com/stonith404/pocket-id/backend/internal/middleware"
	"github.com/stonith404/pocket-id/backend/internal/service"
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
	webauthnService := service.NewWebAuthnService(db, appConfigService)
	jwtService := service.NewJwtService(appConfigService)
	emailService := service.NewEmailService(appConfigService)
	userService := service.NewUserService(db, jwtService)
	oidcService := service.NewOidcService(db, jwtService, appConfigService, emailService)
	testService := service.NewTestService(db, appConfigService)

	// Add global middleware
	r.Use(middleware.NewCorsMiddleware().Add())
	r.Use(middleware.NewRateLimitMiddleware().Add(rate.Every(time.Second), 60))
	r.Use(middleware.NewJwtAuthMiddleware(jwtService, true).Add(false))

	// Initialize middleware
	jwtAuthMiddleware := middleware.NewJwtAuthMiddleware(jwtService, false)
	fileSizeLimitMiddleware := middleware.NewFileSizeLimitMiddleware()

	// Set up API routes
	apiGroup := r.Group("/api")
	controller.NewWebauthnController(apiGroup, jwtAuthMiddleware, middleware.NewRateLimitMiddleware(), webauthnService, jwtService)
	controller.NewOidcController(apiGroup, jwtAuthMiddleware, fileSizeLimitMiddleware, oidcService, jwtService)
	controller.NewUserController(apiGroup, jwtAuthMiddleware, middleware.NewRateLimitMiddleware(), userService)
	controller.NewAppConfigController(apiGroup, jwtAuthMiddleware, appConfigService)

	// Add test controller in non-production environments
	if common.EnvConfig.AppEnv != "production" {
		controller.NewTestController(apiGroup, testService)
	}

	// Set up base routes
	baseGroup := r.Group("/")
	controller.NewWellKnownController(baseGroup, jwtService)

	// Run the server
	if err := r.Run(common.EnvConfig.Host + ":" + common.EnvConfig.Port); err != nil {
		log.Fatal(err)
	}
}
