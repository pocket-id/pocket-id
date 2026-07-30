//go:build e2etest

package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/pocket-id/pocket-id/backend/internal/httpserver"
	"github.com/pocket-id/pocket-id/backend/internal/service"
)

func NewTestController(group *gin.RouterGroup, testService *service.TestService) {
	testController := &TestController{TestService: testService}

	group.POST("/test/reset", httpserver.Handle(testController.resetAndSeedHandler))
	group.POST("/test/accesstoken", httpserver.Handle(testController.signAccessToken))
	group.POST("/test/refreshtoken", httpserver.Handle(testController.signRefreshToken))

	group.GET("/externalidp/jwks.json", httpserver.Handle(testController.externalIdPJWKS))
	group.POST("/externalidp/sign", httpserver.Handle(testController.externalIdPSignToken))
}

type TestController struct {
	TestService *service.TestService
}

func (tc *TestController) resetAndSeedHandler(c *gin.Context) error {
	var baseURL string
	if c.Request.TLS != nil {
		baseURL = "https://" + c.Request.Host
	} else {
		baseURL = "http://" + c.Request.Host
	}

	skipLdap := c.Query("skip-ldap") == "true"
	skipSeed := c.Query("skip-seed") == "true"

	if err := tc.TestService.ResetDatabase(); err != nil {
		return err
	}

	if err := tc.TestService.ResetApplicationImages(c.Request.Context()); err != nil {
		return err
	}

	if !skipSeed {
		if err := tc.TestService.SeedDatabase(baseURL); err != nil {
			return err
		}
	}

	if err := tc.TestService.ResetAppConfig(c.Request.Context()); err != nil {
		return err
	}

	if !skipLdap {
		if err := tc.TestService.SetLdapTestConfig(c.Request.Context()); err != nil {
			return err
		}

		if err := tc.TestService.SyncLdap(c.Request.Context()); err != nil {
			return err
		}
	}

	c.Status(http.StatusNoContent)
	return nil
}

func (tc *TestController) externalIdPJWKS(c *gin.Context) error {
	jwks, err := tc.TestService.GetExternalIdPJWKS()
	if err != nil {
		return err
	}

	c.JSON(http.StatusOK, jwks)
	return nil
}

func (tc *TestController) externalIdPSignToken(c *gin.Context) error {
	var input struct {
		Aud string `json:"aud"`
		Iss string `json:"iss"`
		Sub string `json:"sub"`
	}
	err := httpserver.BindJSON(c, &input)
	if err != nil {
		return err
	}

	token, err := tc.TestService.SignExternalIdPToken(input.Iss, input.Sub, input.Aud)
	if err != nil {
		return err
	}

	c.Writer.WriteString(token)
	return nil
}

func (tc *TestController) signAccessToken(c *gin.Context) error {
	var input struct {
		UserID   string `json:"user"`
		ClientID string `json:"client"`
		Expired  bool   `json:"expired"`
	}
	err := httpserver.BindJSON(c, &input)
	if err != nil {
		return err
	}

	token, err := tc.TestService.SignAccessToken(c.Request.Context(), input.UserID, input.ClientID, input.Expired)
	if err != nil {
		return err
	}

	c.Writer.WriteString(token)
	return nil
}

func (tc *TestController) signRefreshToken(c *gin.Context) error {
	var input struct {
		UserID       string `json:"user"`
		ClientID     string `json:"client"`
		RefreshToken string `json:"rt"`
	}
	err := httpserver.BindJSON(c, &input)
	if err != nil {
		return err
	}

	token, err := tc.TestService.SignRefreshToken(c.Request.Context(), input.UserID, input.ClientID, input.RefreshToken)
	if err != nil {
		return err
	}

	c.Writer.WriteString(token)
	return nil
}
