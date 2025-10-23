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
	group.POST("/test/refreshtoken", testController.signRefreshToken)

	group.GET("/externalidp/jwks.json", testController.externalIdPJWKS)
	group.POST("/externalidp/sign", testController.externalIdPSignToken)
}

type TestController struct {
	TestService *service.TestService
}

func (tc *TestController) resetAndSeedHandler(c *gin.Context) {
	var baseURL string
	if c.Request.TLS != nil {
		baseURL = "https://" + c.Request.Host
	} else {
		baseURL = "http://" + c.Request.Host
	}

	skipLdap := c.Query("skip-ldap") == "true"
	skipSeed := c.Query("skip-seed") == "true"

	if err := tc.TestService.ResetDatabase(); err != nil {
		_ = c.Error(err)
		return
	}

	if err := tc.TestService.ResetApplicationImages(); err != nil {
		_ = c.Error(err)
		return
	}

	if !skipSeed {
		if err := tc.TestService.SeedDatabase(baseURL); err != nil {
			_ = c.Error(err)
			return
		}
	}

	if err := tc.TestService.ResetAppConfig(c.Request.Context()); err != nil {
		_ = c.Error(err)
		return
	}

	if !skipLdap {
		if err := tc.TestService.SetLdapTestConfig(c.Request.Context()); err != nil {
			_ = c.Error(err)
			return
		}

		if err := tc.TestService.SyncLdap(c.Request.Context()); err != nil {
			_ = c.Error(err)
			return
		}
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

func (tc *TestController) externalIdPSignToken(c *gin.Context) {
	var input struct {
		Aud string `json:"aud"`
		Iss string `json:"iss"`
		Sub string `json:"sub"`
	}
	err := c.ShouldBindJSON(&input)
	if err != nil {
		_ = c.Error(err)
		return
	}

	token, err := tc.TestService.SignExternalIdPToken(input.Iss, input.Sub, input.Aud)
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.Writer.WriteString(token)
}

func (tc *TestController) signRefreshToken(c *gin.Context) {
	var input struct {
		UserID       string `json:"user"`
		ClientID     string `json:"client"`
		RefreshToken string `json:"rt"`
	}
	err := c.ShouldBindJSON(&input)
	if err != nil {
		_ = c.Error(err)
		return
	}

	token, err := tc.TestService.SignRefreshToken(input.UserID, input.ClientID, input.RefreshToken)
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.Writer.WriteString(token)
}
