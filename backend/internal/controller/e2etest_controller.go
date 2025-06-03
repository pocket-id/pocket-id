//go:build e2etest

package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/pocket-id/pocket-id/backend/internal/service"
)

func NewTestController(group *gin.RouterGroup, testService *service.TestService) {
	testController := &TestController{TestService: testService}

	group.POST("/test/reset", testController.resetAndSeedHandler)
	group.GET("/test/externalidp/jwks.json", testController.externalIdPJWKS)
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

	if err := tc.TestService.ResetAppConfig(c.Request.Context()); err != nil {
		_ = c.Error(err)
		return
	}

	if err := tc.TestService.SetLdapTestConfig(c.Request.Context()); err != nil {
		_ = c.Error(err)
		return
	}

	if err := tc.TestService.SyncLdap(c.Request.Context()); err != nil {
		_ = c.Error(err)
		return
	}

	tc.TestService.SetJWTKeys()

	c.Status(http.StatusNoContent)
}

func (tc *TestController) externalIdPJWKS(c *gin.Context) {
	jwks, err := tc.TestService.GetExternalIdPJWKS()
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, jwks)
}
