package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/service"
)

func NewWellKnownController(group *gin.RouterGroup, jwtService *service.JwtService) {
	wkc := &WellKnownController{jwtService: jwtService}
	group.GET("/.well-known/jwks.json", wkc.jwksHandler)
	group.GET("/.well-known/openid-configuration", wkc.openIDConfigurationHandler)
}

type WellKnownController struct {
	jwtService *service.JwtService
}

func (wkc *WellKnownController) jwksHandler(c *gin.Context) {
	jwk, err := wkc.jwtService.GetJWK()
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"keys": []interface{}{jwk}})
}

func (wkc *WellKnownController) openIDConfigurationHandler(c *gin.Context) {
	appUrl := common.EnvConfig.AppURL
	config := map[string]interface{}{
		"issuer":                                appUrl,
		"authorization_endpoint":                appUrl + "/authorize",
		"token_endpoint":                        appUrl + "/api/oidc/token",
		"userinfo_endpoint":                     appUrl + "/api/oidc/userinfo",
		"end_session_endpoint":                  appUrl + "/api/oidc/end-session",
		"device_authorization_endpoint":         appUrl + "/api/oidc/device/authorize",
		"jwks_uri":                              appUrl + "/.well-known/jwks.json",
		"scopes_supported":                      []string{"openid", "profile", "email", "groups"},
		"claims_supported":                      []string{"sub", "given_name", "family_name", "name", "email", "email_verified", "preferred_username", "picture", "groups"},
		"response_types_supported":              []string{"code", "id_token"},
		"subject_types_supported":               []string{"public"},
		"id_token_signing_alg_values_supported": []string{"RS256"},
		"grant_types_supported":                 []string{"authorization_code", "urn:ietf:params:oauth:grant-type:device_code"},
	}
	c.JSON(http.StatusOK, config)
}
