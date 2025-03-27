package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/pocket-id/pocket-id/backend/internal/service"
)

func NewTestController(group *gin.RouterGroup, testService *service.TestService) {
	testController := &TestController{TestService: testService}

	group.POST("/test/reset", testController.resetAndSeedHandler)
}

type TestController struct {
	TestService *service.TestService
}

func (tc *TestController) resetAndSeedHandler(c *gin.Context) {
	if err := tc.TestService.ResetDatabase(); err != nil {
		_ = c.Error(err)
		return
	}

	if err := tc.TestService.ResetApplicationImages(); err != nil {
		_ = c.Error(err)
		return
	}

	if err := tc.TestService.SeedDatabase(); err != nil {
		_ = c.Error(err)
		return
	}

	if err := tc.TestService.ResetAppConfig(); err != nil {
		_ = c.Error(err)
		return
	}

	tc.TestService.SetJWTKeys()

	c.Status(http.StatusNoContent)
}
