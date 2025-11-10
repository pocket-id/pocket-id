//go:build e2etest

package bootstrap

import (
	"log/slog"
	"os"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/controller"
	"github.com/pocket-id/pocket-id/backend/internal/service"
)

// When building for E2E tests, add the e2etest controller
func init() {
	registerTestControllers = []func(apiGroup *gin.RouterGroup, db *gorm.DB, svc *services){
		func(apiGroup *gin.RouterGroup, db *gorm.DB, svc *services) {
			testService, err := service.NewTestService(db, svc.appConfigService, svc.jwtService, svc.ldapService, svc.appLockService, svc.fileStorage)
			if err != nil {
				slog.Error("Failed to initialize test service", slog.Any("error", err))
				os.Exit(1)
				return
			}

			controller.NewTestController(apiGroup, testService)
		},
	}
}
