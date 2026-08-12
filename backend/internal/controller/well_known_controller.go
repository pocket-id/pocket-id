package controller

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/pocket-id/pocket-id/backend/internal/common"
	_ "github.com/pocket-id/pocket-id/backend/internal/dto"
	"github.com/pocket-id/pocket-id/backend/internal/httpserver"
	"github.com/pocket-id/pocket-id/backend/internal/service"
)

// NewWellKnownController creates a new controller for OIDC discovery endpoints
// @Summary OIDC Discovery controller
// @Description Initializes OIDC discovery, OAuth 2.0 authorization server metadata and JWKS endpoints
// @Tags Well Known
func NewWellKnownController(group *gin.RouterGroup, jwtService *service.JwtService, getCIMDURLAllowlist func() []string) {
	wkc := &WellKnownController{
		jwtService:          jwtService,
		getCIMDURLAllowlist: getCIMDURLAllowlist,
	}

	group.GET("/.well-known/jwks.json", httpserver.Handle(wkc.jwksHandler))
	group.GET("/.well-known/openid-configuration", httpserver.Handle(wkc.openIDConfigurationHandler))
	group.GET("/.well-known/oauth-authorization-server", httpserver.Handle(wkc.oauthAuthorizationServerHandler))
}

type WellKnownController struct {
	jwtService          *service.JwtService
	getCIMDURLAllowlist func() []string
}

// jwksHandler godoc
// @Summary Get JSON Web Key Set (JWKS)
// @Description Returns the JSON Web Key Set used for token verification
// @Tags Well Known
// @Produce json
// @Success 200 {object} object "{ \"keys\": []interface{} }"
// @Failure default {object} dto.ErrorDto "Error"
// @Router /.well-known/jwks.json [get]
func (wkc *WellKnownController) jwksHandler(c *gin.Context) error {
	jwks, err := wkc.jwtService.GetPublicJWKSAsJSON()
	if err != nil {
		return err
	}

	c.Data(http.StatusOK, "application/json; charset=utf-8", jwks)
	return nil
}

// openIDConfigurationHandler godoc
// @Summary Get OpenID Connect discovery configuration
// @Description Returns the OpenID Connect discovery document with endpoints and capabilities
// @Tags Well Known
// @Success 200 {object} object "OpenID Connect configuration"
// @Failure default {object} dto.ErrorDto "Error"
// @Router /.well-known/openid-configuration [get]
func (wkc *WellKnownController) openIDConfigurationHandler(c *gin.Context) error {
	return wkc.writeServerMetadata(c)
}

// oauthAuthorizationServerHandler godoc
// @Summary Get OAuth 2.0 authorization server metadata
// @Description Returns the RFC 8414 OAuth 2.0 authorization server metadata document with endpoints and capabilities
// @Tags Well Known
// @Success 200 {object} object "OAuth 2.0 authorization server metadata"
// @Failure default {object} dto.ErrorDto "Error"
// @Router /.well-known/oauth-authorization-server [get]
func (wkc *WellKnownController) oauthAuthorizationServerHandler(c *gin.Context) error {
	return wkc.writeServerMetadata(c)
}

func (wkc *WellKnownController) writeServerMetadata(c *gin.Context) error {
	metadata, err := wkc.computeServerMetadata()
	if err != nil {
		return err
	}
	c.Data(http.StatusOK, "application/json; charset=utf-8", metadata)
	return nil
}

func (wkc *WellKnownController) computeServerMetadata() ([]byte, error) {
	appUrl := common.EnvConfig.AppURL

	internalAppUrl := common.EnvConfig.InternalAppURL

	alg, err := wkc.jwtService.GetKeyAlg()
	if err != nil {
		return nil, fmt.Errorf("failed to get key algorithm: %w", err)
	}
	cimdSupported := false
	if wkc.getCIMDURLAllowlist != nil {
		cimdSupported = len(wkc.getCIMDURLAllowlist()) > 0
	}

	config := map[string]any{
		"issuer":                                         appUrl,
		"authorization_endpoint":                         appUrl + "/authorize",
		"token_endpoint":                                 internalAppUrl + "/api/oidc/token",
		"userinfo_endpoint":                              internalAppUrl + "/api/oidc/userinfo",
		"end_session_endpoint":                           appUrl + "/api/oidc/end-session",
		"introspection_endpoint":                         internalAppUrl + "/api/oidc/introspect",
		"device_authorization_endpoint":                  appUrl + "/api/oidc/device/authorize",
		"jwks_uri":                                       internalAppUrl + "/.well-known/jwks.json",
		"grant_types_supported":                          []string{service.GrantTypeAuthorizationCode, service.GrantTypeRefreshToken, service.GrantTypeDeviceCode, service.GrantTypeClientCredentials},
		"scopes_supported":                               []string{"openid", "profile", "email", "groups", "offline_access"},
		"claims_supported":                               []string{"sub", "given_name", "family_name", "name", "display_name", "email", "email_verified", "preferred_username", "picture", "groups", "auth_time", "amr"},
		"response_types_supported":                       []string{"code", "id_token"},
		"response_modes_supported":                       []string{"query", "fragment", "form_post"},
		"subject_types_supported":                        []string{"public"},
		"id_token_signing_alg_values_supported":          []string{alg.String()},
		"authorization_response_iss_parameter_supported": true,
		"code_challenge_methods_supported":               []string{"plain", "S256"},
		"request_parameter_supported":                    true,
		"request_uri_parameter_supported":                false,
		"request_object_signing_alg_values_supported":    []string{"none"},
		"prompt_values_supported":                        []string{"none", "login", "consent", "select_account"},
		"token_endpoint_auth_methods_supported":          []string{"client_secret_basic", "client_secret_post", "none"},
		"pushed_authorization_request_endpoint":          internalAppUrl + "/api/oidc/par",
		"require_pushed_authorization_requests":          false,
		"client_id_metadata_document_supported":          cimdSupported,
		"service_documentation":                          "https://pocket-id.org/docs",
	}
	return json.Marshal(config)
}
