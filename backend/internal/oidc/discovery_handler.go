package oidc

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/lestrrat-go/jwx/v4/jwa"
	"github.com/pocket-id/fosite"
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/claimmappingpolicy"
	"github.com/pocket-id/pocket-id/backend/internal/common"
	_ "github.com/pocket-id/pocket-id/backend/internal/dto"
)

// KeyPublisher publishes the public half of the signing key, so clients can verify the tokens Pocket ID signs
type KeyPublisher interface {
	GetKeyAlg() (jwa.KeyAlgorithm, error)
	PublicJWKS() []byte
}

type discoveryHandler struct {
	db                  *gorm.DB
	keys                KeyPublisher
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
func (h *discoveryHandler) jwksHandler(c *gin.Context) error {
	c.Data(http.StatusOK, "application/json; charset=utf-8", h.keys.PublicJWKS())
	return nil
}

// openIDConfigurationHandler godoc
// @Summary Get OpenID Connect discovery configuration
// @Description Returns the OpenID Connect discovery document with endpoints and capabilities
// @Tags Well Known
// @Success 200 {object} object "OpenID Connect configuration"
// @Failure default {object} dto.ErrorDto "Error"
// @Router /.well-known/openid-configuration [get]
func (h *discoveryHandler) openIDConfigurationHandler(c *gin.Context) error {
	return h.writeServerMetadata(c)
}

// oauthAuthorizationServerHandler godoc
// @Summary Get OAuth 2.0 authorization server metadata
// @Description Returns the RFC 8414 OAuth 2.0 authorization server metadata document with endpoints and capabilities
// @Tags Well Known
// @Success 200 {object} object "OAuth 2.0 authorization server metadata"
// @Failure default {object} dto.ErrorDto "Error"
// @Router /.well-known/oauth-authorization-server [get]
func (h *discoveryHandler) oauthAuthorizationServerHandler(c *gin.Context) error {
	return h.writeServerMetadata(c)
}

func (h *discoveryHandler) writeServerMetadata(c *gin.Context) error {
	metadata, err := h.computeServerMetadata(c.Request.Context())
	if err != nil {
		return err
	}
	c.Data(http.StatusOK, "application/json; charset=utf-8", metadata)
	return nil
}

func (h *discoveryHandler) computeServerMetadata(ctx context.Context) ([]byte, error) {
	appUrl := common.EnvConfig.AppURL

	internalAppUrl := common.EnvConfig.InternalAppURL

	alg, err := h.keys.GetKeyAlg()
	if err != nil {
		return nil, fmt.Errorf("failed to get key algorithm: %w", err)
	}
	customScopes, err := claimmappingpolicy.CustomScopesByKey(ctx, h.db, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to load custom scopes: %w", err)
	}
	scopesSupported := []string{"openid", "profile", "email", "groups", "offline_access"}
	for _, scope := range customScopes {
		scopesSupported = append(scopesSupported, scope.Key)
	}

	cimdSupported := false
	if h.getCIMDURLAllowlist != nil {
		cimdSupported = len(h.getCIMDURLAllowlist()) > 0
	}

	config := map[string]any{
		"issuer":                 appUrl,
		"authorization_endpoint": appUrl + "/authorize",
		"token_endpoint":         internalAppUrl + "/api/oidc/token",
		"userinfo_endpoint":      internalAppUrl + "/api/oidc/userinfo",
		"end_session_endpoint":   appUrl + "/api/oidc/end-session",
		"introspection_endpoint": internalAppUrl + "/api/oidc/introspect",
		"introspection_endpoint_auth_methods_supported":  []string{"client_secret_basic", "Bearer"},
		"device_authorization_endpoint":                  appUrl + "/api/oidc/device/authorize",
		"jwks_uri":                                       internalAppUrl + "/.well-known/jwks.json",
		"grant_types_supported":                          []fosite.GrantType{fosite.GrantTypeAuthorizationCode, fosite.GrantTypeRefreshToken, fosite.GrantTypeDeviceCode, fosite.GrantTypeClientCredentials},
		"scopes_supported":                               scopesSupported,
		"claims_supported":                               []string{"sub", "given_name", "family_name", "name", "display_name", "email", "email_verified", "preferred_username", "picture", "groups", "auth_time", "amr"},
		"response_types_supported":                       []string{"code"},
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
		"backchannel_logout_supported":                   true,
		"backchannel_logout_session_supported":           true,
		"client_id_metadata_document_supported":          cimdSupported,
		"service_documentation":                          "https://pocket-id.org/docs",
	}
	return json.Marshal(config)
}
