//go:build e2etest

package bootstrap

import (
	"log"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/controller"
	"github.com/pocket-id/pocket-id/backend/internal/service"
)

// When building for E2E tests, add the e2etest controller
func init() {
	registerTestControllers = []func(apiGroup *gin.RouterGroup, db *gorm.DB, svc *services){
		func(apiGroup *gin.RouterGroup, db *gorm.DB, svc *services) {
			testService, err := service.NewTestService(db, svc.appConfigService, svc.jwtService, svc.ldapService)
			if err != nil {
				log.Fatalf("failed to initialize test service: %v", err)
				return
			}

			controller.NewTestController(apiGroup, testService)
		},
	}
}
