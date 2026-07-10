package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/danielgtaylor/huma/v2"

	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/service"
	httpapi "github.com/pocket-id/pocket-id/backend/internal/utils/huma"
)

type wellKnownOutput struct {
	ContentType string `header:"Content-Type"`
	Body        []byte
}

// NewWellKnownController registers OIDC discovery endpoints
func NewWellKnownController(api huma.API, jwtService *service.JwtService) {
	controller := &WellKnownController{jwtService: jwtService}

	var err error
	controller.oidcConfig, err = controller.computeOIDCConfiguration()
	if err != nil {
		slog.Error("Failed to pre-compute OpenID Connect configuration document", slog.Any("error", err))
		os.Exit(1)
		return
	}

	httpapi.Register(api, huma.Operation{OperationID: "get-jwks", Method: http.MethodGet, Path: "/.well-known/jwks.json", Summary: "Get JSON Web Key Set", Tags: []string{"Well Known"}}, controller.jwksHandler)
	httpapi.Register(api, huma.Operation{OperationID: "get-openid-configuration", Method: http.MethodGet, Path: "/.well-known/openid-configuration", Summary: "Get OpenID Connect discovery configuration", Tags: []string{"Well Known"}}, controller.openIDConfigurationHandler)
}

type WellKnownController struct {
	jwtService *service.JwtService
	oidcConfig []byte
}

func (wkc *WellKnownController) jwksHandler(_ context.Context, _ *httpapi.EmptyInput) (*wellKnownOutput, error) {
	jwks, err := wkc.jwtService.GetPublicJWKSAsJSON()
	if err != nil {
		return nil, err
	}
	return &wellKnownOutput{ContentType: "application/json; charset=utf-8", Body: jwks}, nil
}

func (wkc *WellKnownController) openIDConfigurationHandler(_ context.Context, _ *httpapi.EmptyInput) (*wellKnownOutput, error) {
	return &wellKnownOutput{ContentType: "application/json; charset=utf-8", Body: wkc.oidcConfig}, nil
}

func (wkc *WellKnownController) computeOIDCConfiguration() ([]byte, error) {
	appURL := common.EnvConfig.AppURL
	internalAppURL := common.EnvConfig.InternalAppURL

	alg, err := wkc.jwtService.GetKeyAlg()
	if err != nil {
		return nil, fmt.Errorf("failed to get key algorithm: %w", err)
	}
	config := map[string]any{
		"issuer":                                         appURL,
		"authorization_endpoint":                         appURL + "/authorize",
		"token_endpoint":                                 internalAppURL + "/api/oidc/token",
		"userinfo_endpoint":                              internalAppURL + "/api/oidc/userinfo",
		"end_session_endpoint":                           appURL + "/api/oidc/end-session",
		"introspection_endpoint":                         internalAppURL + "/api/oidc/introspect",
		"device_authorization_endpoint":                  appURL + "/api/oidc/device/authorize",
		"jwks_uri":                                       internalAppURL + "/.well-known/jwks.json",
		"grant_types_supported":                          []string{service.GrantTypeAuthorizationCode, service.GrantTypeRefreshToken, service.GrantTypeDeviceCode, service.GrantTypeClientCredentials},
		"scopes_supported":                               []string{"openid", "profile", "email", "groups", "offline_access"},
		"claims_supported":                               []string{"sub", "given_name", "family_name", "name", "display_name", "email", "email_verified", "preferred_username", "picture", "groups", "auth_time", "amr"},
		"response_types_supported":                       []string{"code", "id_token"},
		"subject_types_supported":                        []string{"public"},
		"id_token_signing_alg_values_supported":          []string{alg.String()},
		"authorization_response_iss_parameter_supported": true,
		"code_challenge_methods_supported":               []string{"plain", "S256"},
		"request_parameter_supported":                    true,
		"request_uri_parameter_supported":                false,
		"request_object_signing_alg_values_supported":    []string{"none"},
		"prompt_values_supported":                        []string{"none", "login", "consent", "select_account"},
		"token_endpoint_auth_methods_supported":          []string{"client_secret_basic", "client_secret_post", "none"},
		"pushed_authorization_request_endpoint":          internalAppURL + "/api/oidc/par",
		"require_pushed_authorization_requests":          false,
	}
	return json.Marshal(config)
}
