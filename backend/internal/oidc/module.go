package oidc

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/gin-gonic/gin"
	"github.com/lestrrat-go/jwx/v3/jwa"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	httpapi "github.com/pocket-id/pocket-id/backend/internal/utils/huma"
	"gorm.io/gorm"
)

type Config struct {
	BaseURL      string
	TokenBaseURL string
	Secret       []byte
}

type TokenSigner interface {
	GetPrivateKey() any
	GetKeyAlg() (jwa.KeyAlgorithm, error)
	GetKeyID() (string, bool)
}

type CustomClaimSource interface {
	GetCustomClaimsForUserWithUserGroups(ctx context.Context, userID string, tx *gorm.DB) ([]model.CustomClaim, error)
}

type ReauthenticationTokenConsumer interface {
	ConsumeReauthenticationToken(ctx context.Context, tx *gorm.DB, token string, userID string) (time.Time, error)
}

type AuditLogger interface {
	Create(ctx context.Context, event model.AuditLogEvent, ipAddress, userAgent, userID string, data model.AuditLogData, tx *gorm.DB) (model.AuditLog, bool)
}

type Dependencies struct {
	DB         *gorm.DB
	Config     Config
	HTTPClient *http.Client

	Signer       TokenSigner
	CustomClaims CustomClaimSource
	Reauth       ReauthenticationTokenConsumer
	AuditLog     AuditLogger
	APIAccess    APIAccessProvider
}

type Module struct {
	Preview *ClientPreviewBuilder

	config Config
	store  *Store

	authorizationHandler *authorizationHandler
	tokenHandler         *tokenHandler
	userInfoHandler      *userInfoHandler
	parHandler           *parHandler
	introspectionHandler *introspectionHandler
	endSessionHandler    *endSessionHandler
	deviceHandler        *deviceHandler
}

func New(ctx context.Context, deps Dependencies) (*Module, error) {
	store := NewStore(deps.DB, deps.APIAccess).WithIssuer(deps.Config.BaseURL)
	authenticator, err := newFederatedClientAuthenticator(ctx, store, deps.HTTPClient, deps.Config.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create federated client authenticator: %w", err)
	}
	provider, err := newProvider(store, authenticator, deps.Signer, deps.Config)
	if err != nil {
		return nil, fmt.Errorf("failed to create OAuth2 provider: %w", err)
	}

	claimsService := newClaimsService(deps.DB, deps.CustomClaims, deps.Config.BaseURL, deps.Signer)
	previewBuilder := newClientPreviewBuilder(claimsService, provider.tokenStrategies)
	interactionSessionService := newInteractionSessionService(deps.DB)
	authorizationService := newAuthorizationService(deps.DB, interactionSessionService, claimsService, deps.Reauth, deps.AuditLog, deps.APIAccess)
	deviceService := newDeviceService(provider, store, provider.deviceStrategy, authorizationService, claimsService, deps.AuditLog, deps.DB)
	endSessionService := newEndSessionService(deps.DB, store, deps.Signer, deps.Config.BaseURL)

	return &Module{
		Preview: previewBuilder,

		config: deps.Config,
		store:  store,

		authorizationHandler: newAuthorizationHandler(provider, authorizationService, deps.Config.BaseURL),
		tokenHandler:         newTokenHandler(provider, claimsService, deps.APIAccess),
		userInfoHandler:      newUserInfoHandler(provider, claimsService, deps.Config.BaseURL),
		parHandler:           newPARHandler(provider),
		introspectionHandler: newIntrospectionHandler(provider, authenticator, deps.Config.BaseURL),
		endSessionHandler:    newEndSessionHandler(endSessionService, deps.Config.BaseURL),
		deviceHandler:        newDeviceHandler(provider, deviceService),
	}, nil
}

// RegisterRawRoutes mounts protocol endpoints that must retain direct Gin and Fosite response control
func (m *Module) RegisterRawRoutes(rootGroup *gin.RouterGroup, apiGroup *gin.RouterGroup, optionalBrowserAuth gin.HandlerFunc, api huma.API) {
	rootGroup.GET("/authorize", optionalBrowserAuth, m.authorizationHandler.authorize)
	rootGroup.POST("/authorize", optionalBrowserAuth, m.authorizationHandler.authorize)

	apiGroup.POST("/oidc/par", m.parHandler.pushedAuthorizationRequest)
	apiGroup.POST("/oidc/token", m.tokenHandler.token)
	apiGroup.GET("/oidc/userinfo", m.userInfoHandler.userInfo)
	apiGroup.POST("/oidc/userinfo", m.userInfoHandler.userInfo)
	apiGroup.POST("/oidc/introspect", m.introspectionHandler.introspectToken)
	apiGroup.GET("/oidc/end-session", optionalBrowserAuth, m.endSessionHandler.endSession)
	apiGroup.POST("/oidc/end-session", optionalBrowserAuth, m.endSessionHandler.endSession)
	apiGroup.POST("/oidc/device/authorize", m.deviceHandler.authorizeDevice)

	httpapi.AddRawOperation(api, huma.Operation{
		OperationID: "authorize-get",
		Method:      http.MethodGet,
		Path:        "/authorize",
		Summary:     "Authorize",
		Tags:        []string{"OIDC Protocol"},
	}, http.StatusOK, http.StatusFound)

	httpapi.AddRawOperation(api, huma.Operation{
		OperationID: "authorize-post",
		Method:      http.MethodPost,
		Path:        "/authorize",
		Summary:     "Authorize",
		Tags:        []string{"OIDC Protocol"},
	}, http.StatusOK, http.StatusFound)

	httpapi.AddRawOperation(api, huma.Operation{
		OperationID: "pushed-authorization-request",
		Method:      http.MethodPost,
		Path:        "/api/oidc/par",
		Summary:     "Create pushed authorization request",
		Tags:        []string{"OIDC Protocol"},
		Security:    []map[string][]string{{"OIDCClientBasic": {}}},
	})

	httpapi.AddRawOperation(api, huma.Operation{
		OperationID: "oidc-token",
		Method:      http.MethodPost,
		Path:        "/api/oidc/token",
		Summary:     "Exchange an OIDC token",
		Tags:        []string{"OIDC Protocol"},
		Security:    []map[string][]string{{"OIDCClientBasic": {}}},
	})

	httpapi.AddRawOperation(api, huma.Operation{
		OperationID: "oidc-userinfo-get",
		Method:      http.MethodGet,
		Path:        "/api/oidc/userinfo",
		Summary:     "Get OIDC user info",
		Tags:        []string{"OIDC Protocol"},
		Security:    []map[string][]string{{"OIDCAccessToken": {}}},
	})

	httpapi.AddRawOperation(api, huma.Operation{
		OperationID: "oidc-userinfo-post",
		Method:      http.MethodPost,
		Path:        "/api/oidc/userinfo",
		Summary:     "Get OIDC user info",
		Tags:        []string{"OIDC Protocol"},
		Security:    []map[string][]string{{"OIDCAccessToken": {}}},
	})

	httpapi.AddRawOperation(api, huma.Operation{
		OperationID: "oidc-introspection",
		Method:      http.MethodPost,
		Path:        "/api/oidc/introspect",
		Summary:     "Introspect an OIDC token",
		Tags:        []string{"OIDC Protocol"},
		Security:    []map[string][]string{{"OIDCClientBasic": {}}},
	})

	httpapi.AddRawOperation(api, huma.Operation{
		OperationID: "oidc-end-session-get",
		Method:      http.MethodGet,
		Path:        "/api/oidc/end-session",
		Summary:     "End an OIDC session",
		Tags:        []string{"OIDC Protocol"},
	}, http.StatusFound)

	httpapi.AddRawOperation(api, huma.Operation{
		OperationID: "oidc-end-session-post",
		Method:      http.MethodPost,
		Path:        "/api/oidc/end-session",
		Summary:     "End an OIDC session",
		Tags:        []string{"OIDC Protocol"},
	}, http.StatusFound)

	httpapi.AddRawOperation(api, huma.Operation{
		OperationID: "oidc-device-authorization",
		Method:      http.MethodPost,
		Path:        "/api/oidc/device/authorize",
		Summary:     "Create device authorization",
		Tags:        []string{"OIDC Protocol"},
		Security:    []map[string][]string{{"OIDCClientBasic": {}}},
	})
}

// RegisterTypedRoutes mounts JSON interaction and device verification endpoints
func (m *Module) RegisterTypedRoutes(api huma.API, browserAuth func(*huma.Operation)) {
	httpapi.Register(api, huma.Operation{
		OperationID: "get-oidc-interaction",
		Method:      http.MethodGet,
		Path:        "/api/oidc/interactions/{id}",
		Summary:     "Get OIDC interaction",
		Tags:        []string{"OIDC Interactions"},
	}, m.authorizationHandler.getInteractionSession)

	httpapi.Register(api, huma.Operation{
		OperationID: "complete-oidc-interaction",
		Method:      http.MethodPost,
		Path:        "/api/oidc/interactions/{id}/complete",
		Summary:     "Complete OIDC interaction",
		Tags:        []string{"OIDC Interactions"},
	}, m.authorizationHandler.completeInteraction, browserAuth)

	httpapi.Register(api, huma.Operation{
		OperationID:   "verify-oidc-device-code",
		Method:        http.MethodPost,
		Path:          "/api/oidc/device/verify",
		Summary:       "Verify OIDC device code",
		Tags:          []string{"OIDC Protocol"},
		DefaultStatus: http.StatusNoContent,
	}, m.deviceHandler.verifyDeviceCode, browserAuth)

	httpapi.Register(api, huma.Operation{
		OperationID: "get-oidc-device-info",
		Method:      http.MethodGet,
		Path:        "/api/oidc/device/info",
		Summary:     "Get OIDC device code info",
		Tags:        []string{"OIDC Protocol"},
	}, m.deviceHandler.deviceCodeInfo, browserAuth)
}
