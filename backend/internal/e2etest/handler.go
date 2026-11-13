//go:build e2etest

package e2etest

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/pocket-id/pocket-id/backend/internal/httpserver"
)

type handler struct {
	service *Service
}

func (tc *handler) resetAndSeedHandler(c *gin.Context) error {
	var baseURL string
	if c.Request.TLS != nil {
		baseURL = "https://" + c.Request.Host
	} else {
		baseURL = "http://" + c.Request.Host
	}

	skipLdap := c.Query("skip-ldap") == "true"
	skipSeed := c.Query("skip-seed") == "true"

	if err := tc.service.ResetDatabase(); err != nil {
		return err
	}

	if err := tc.service.ResetApplicationAssets(c.Request.Context()); err != nil {
		return err
	}

	if !skipSeed {
		if err := tc.service.SeedDatabase(baseURL); err != nil {
			return err
		}
	}

	if err := tc.service.ResetAppConfig(c.Request.Context()); err != nil {
		return err
	}

	if !skipLdap {
		if err := tc.service.SetLdapTestConfig(c.Request.Context()); err != nil {
			return err
		}

		if err := tc.service.SyncLdap(c.Request.Context()); err != nil {
			return err
		}
	}

	c.Status(http.StatusNoContent)
	return nil
}

func (tc *handler) externalIdPJWKS(c *gin.Context) error {
	jwks, err := tc.service.GetExternalIdPJWKS()
	if err != nil {
		return err
	}

	c.JSON(http.StatusOK, jwks)
	return nil
}

func (tc *handler) externalIdPSignToken(c *gin.Context) error {
	var input struct {
		Aud string `json:"aud"`
		Iss string `json:"iss"`
		Sub string `json:"sub"`
	}
	err := httpserver.BindJSON(c, &input)
	if err != nil {
		return err
	}

	token, err := tc.service.SignExternalIdPToken(input.Iss, input.Sub, input.Aud)
	if err != nil {
		return err
	}

	c.Writer.WriteString(token)
	return nil
}

func (tc *handler) signAccessToken(c *gin.Context) error {
	var input struct {
		UserID   string `json:"user"`
		ClientID string `json:"client"`
		Expired  bool   `json:"expired"`
	}
	err := httpserver.BindJSON(c, &input)
	if err != nil {
		return err
	}

	token, err := tc.service.SignAccessToken(c.Request.Context(), input.UserID, input.ClientID, input.Expired)
	if err != nil {
		return err
	}

	c.Writer.WriteString(token)
	return nil
}

func (tc *handler) signRefreshToken(c *gin.Context) error {
	var input struct {
		UserID       string `json:"user"`
		ClientID     string `json:"client"`
		RefreshToken string `json:"rt"`
	}
	err := httpserver.BindJSON(c, &input)
	if err != nil {
		return err
	}

	token, err := tc.service.SignRefreshToken(c.Request.Context(), input.UserID, input.ClientID, input.RefreshToken)
	if err != nil {
		return err
	}

	c.Writer.WriteString(token)
	return nil
}
