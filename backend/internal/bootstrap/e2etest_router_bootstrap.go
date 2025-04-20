//go:build e2etest

package bootstrap

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/controller"
	"github.com/pocket-id/pocket-id/backend/internal/service"
)

// When building for E2E tests, add the e2etest controller
func init() {
	registerTestControllers = []func(apiGroup *gin.RouterGroup, db *gorm.DB, appConfigService *service.AppConfigService, jwtService *service.JwtService, ldapService *service.LdapService){
		func(apiGroup *gin.RouterGroup, db *gorm.DB, appConfigService *service.AppConfigService, jwtService *service.JwtService, ldapService *service.LdapService) {
			testService := service.NewTestService(db, appConfigService, jwtService, ldapService)
			controller.NewTestController(apiGroup, testService)
		},
	}
}
