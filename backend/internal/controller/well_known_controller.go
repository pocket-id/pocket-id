package controller

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/httpserver"
	"github.com/pocket-id/pocket-id/backend/internal/service"
)

// NewWellKnownController creates a new controller for OIDC discovery endpoints
// @Summary OIDC Discovery controller
// @Description Initializes OIDC discovery and JWKS endpoints
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
// @Router /.well-known/openid-configuration [get]
func (wkc *WellKnownController) openIDConfigurationHandler(c *gin.Context) error {
	oidcConfig, err := wkc.computeOIDCConfiguration()
	if err != nil {
		return err
	}
	c.Data(http.StatusOK, "application/json; charset=utf-8", oidcConfig)
	return nil
}

// oauthAuthorizationServerHandler godoc
// @Summary Get OAuth 2.0 Authorization Server metadata
// @Description Returns the RFC 8414 OAuth 2.0 Authorization Server metadata document with endpoints and capabilities
// @Tags Well Known
// @Success 200 {object} object "OAuth 2.0 Authorization Server metadata"
// @Router /.well-known/oauth-authorization-server [get]
func (wkc *WellKnownController) oauthAuthorizationServerHandler(c *gin.Context) error {
	oauthASConfig, err := wkc.computeOAuthASMetadata()
	if err != nil {
		return err
	}
	c.Data(http.StatusOK, "application/json; charset=utf-8", oauthASConfig)
	return nil
}

// computeBaseMetadata returns the set of metadata fields shared between the OpenID Connect
// discovery document and the RFC 8414 OAuth 2.0 Authorization Server metadata document.
func (wkc *WellKnownController) computeBaseMetadata() (map[string]any, error) {
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

	return map[string]any{
		"issuer":                                         appUrl,
		"authorization_endpoint":                         appUrl + "/authorize",
		"token_endpoint":                                 internalAppUrl + "/api/oidc/token",
		"introspection_endpoint":                         internalAppUrl + "/api/oidc/introspect",
		"device_authorization_endpoint":                  appUrl + "/api/oidc/device/authorize",
		"jwks_uri":                                       internalAppUrl + "/.well-known/jwks.json",
		"registration_endpoint":                          appUrl + "/api/oidc/register",
		"grant_types_supported":                          []string{service.GrantTypeAuthorizationCode, service.GrantTypeRefreshToken, service.GrantTypeDeviceCode, service.GrantTypeClientCredentials},
		"scopes_supported":                               []string{"openid", "profile", "email", "groups", "offline_access"},
		"response_types_supported":                       []string{"code", "id_token"},
		"subject_types_supported":                        []string{"public"},
		"id_token_signing_alg_values_supported":          []string{alg.String()},
		"authorization_response_iss_parameter_supported": true,
		"code_challenge_methods_supported":               []string{"plain", "S256"},
		"token_endpoint_auth_methods_supported":          []string{"client_secret_basic", "client_secret_post", "none"},
		"pushed_authorization_request_endpoint":          internalAppUrl + "/api/oidc/par",
		"require_pushed_authorization_requests":          false,
		// Custom advertisement: not defined by RFC 8414 or RFC 8707, but included so
		// clients can detect that the `resource` parameter is supported.
		"resource_indicators_supported":         true,
		"client_id_metadata_document_supported": cimdSupported,
	}, nil
}

func (wkc *WellKnownController) computeOIDCConfiguration() ([]byte, error) {
	config, err := wkc.computeBaseMetadata()
	if err != nil {
		return nil, err
	}

	appUrl := common.EnvConfig.AppURL
	internalAppUrl := common.EnvConfig.InternalAppURL

	config["userinfo_endpoint"] = internalAppUrl + "/api/oidc/userinfo"
	config["end_session_endpoint"] = appUrl + "/api/oidc/end-session"
	config["claims_supported"] = []string{"sub", "given_name", "family_name", "name", "display_name", "email", "email_verified", "preferred_username", "picture", "groups", "auth_time", "amr"}
	config["request_parameter_supported"] = true
	config["request_uri_parameter_supported"] = false
	config["request_object_signing_alg_values_supported"] = []string{"none"}
	config["prompt_values_supported"] = []string{"none", "login", "consent", "select_account"}

	return json.Marshal(config)
}

// computeOAuthASMetadata returns the RFC 8414 OAuth 2.0 Authorization Server metadata document.
func (wkc *WellKnownController) computeOAuthASMetadata() ([]byte, error) {
	config, err := wkc.computeBaseMetadata()
	if err != nil {
		return nil, err
	}

	return json.Marshal(config)
}
