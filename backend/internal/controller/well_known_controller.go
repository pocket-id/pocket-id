package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/service"
)

// NewWellKnownController creates a new controller for OIDC discovery endpoints
// @Summary OIDC Discovery controller
// @Description Initializes OIDC discovery and JWKS endpoints
// @Tags Well Known
func NewWellKnownController(group *gin.RouterGroup, jwtService *service.JwtService) {
	wkc := &WellKnownController{jwtService: jwtService}
	group.GET("/.well-known/jwks.json", wkc.jwksHandler)
	group.GET("/.well-known/openid-configuration", wkc.openIDConfigurationHandler)
}

type WellKnownController struct {
	jwtService *service.JwtService
}

// jwksHandler godoc
// @Summary Get JSON Web Key Set (JWKS)
// @Description Returns the JSON Web Key Set used for token verification
// @Tags Well Known
// @Produce json
// @Success 200 {object} object "{ \"keys\": []interface{} }"
// @Router /.well-known/jwks.json [get]
func (wkc *WellKnownController) jwksHandler(c *gin.Context) {
	jwks, err := wkc.jwtService.GetPublicJWKSAsJSON()
	if err != nil {
		c.Error(err)
		return
	}

	c.Data(http.StatusOK, "application/json; charset=utf-8", jwks)
}

// openIDConfigurationHandler godoc
// @Summary Get OpenID Connect discovery configuration
// @Description Returns the OpenID Connect discovery document with endpoints and capabilities
// @Tags Well Known
// @Success 200 {object} object "OpenID Connect configuration"
// @Router /.well-known/openid-configuration [get]
func (wkc *WellKnownController) openIDConfigurationHandler(c *gin.Context) {
	appUrl := common.EnvConfig.AppURL
	config := map[string]interface{}{
		"issuer":                                appUrl,
		"authorization_endpoint":                appUrl + "/authorize",
		"token_endpoint":                        appUrl + "/api/oidc/token",
		"userinfo_endpoint":                     appUrl + "/api/oidc/userinfo",
		"end_session_endpoint":                  appUrl + "/api/oidc/end-session",
		"jwks_uri":                              appUrl + "/.well-known/jwks.json",
		"scopes_supported":                      []string{"openid", "profile", "email", "groups"},
		"claims_supported":                      []string{"sub", "given_name", "family_name", "name", "email", "email_verified", "preferred_username", "picture", "groups"},
		"response_types_supported":              []string{"code", "id_token"},
		"subject_types_supported":               []string{"public"},
		"id_token_signing_alg_values_supported": []string{"RS256"},
	}
	c.JSON(http.StatusOK, config)
}
