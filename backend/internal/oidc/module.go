package oidc

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lestrrat-go/jwx/v3/jwa"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	"gorm.io/gorm"
)

type Config struct {
	BaseURL      string
	TokenBaseURL string
	Secret       string
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
	store := NewStore(deps.DB)
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
	authorizationService := newAuthorizationService(deps.DB, interactionSessionService, claimsService, deps.Reauth, deps.AuditLog)
	deviceService := newDeviceService(provider, store, provider.deviceStrategy, authorizationService, claimsService, deps.AuditLog, deps.DB)
	endSessionService := newEndSessionService(deps.DB, store, deps.Signer, deps.Config.BaseURL)

	return &Module{
		Preview: previewBuilder,

		config: deps.Config,
		store:  store,

		authorizationHandler: newAuthorizationHandler(provider, authorizationService, deps.Config.BaseURL),
		tokenHandler:         newTokenHandler(provider, claimsService),
		userInfoHandler:      newUserInfoHandler(provider, claimsService),
		parHandler:           newPARHandler(provider),
		introspectionHandler: newIntrospectionHandler(provider, authenticator, deps.Config.BaseURL),
		endSessionHandler:    newEndSessionHandler(endSessionService, deps.Config.BaseURL),
		deviceHandler:        newDeviceHandler(provider, deviceService),
	}, nil
}

func (m *Module) RegisterRoutes(rootGroup *gin.RouterGroup, apiGroup *gin.RouterGroup, optionalBrowserAuth gin.HandlerFunc, browserAuth gin.HandlerFunc) {
	rootGroup.GET("/authorize", optionalBrowserAuth, m.authorizationHandler.authorize)
	rootGroup.POST("/authorize", optionalBrowserAuth, m.authorizationHandler.authorize)

	apiGroup.GET("/oidc/interactions/:id", m.authorizationHandler.getInteractionSession)
	apiGroup.POST("/oidc/interactions/:id/complete", browserAuth, m.authorizationHandler.completeInteraction)

	apiGroup.POST("/oidc/par", m.parHandler.pushedAuthorizationRequest)

	apiGroup.POST("/oidc/token", m.tokenHandler.token)

	apiGroup.GET("/oidc/userinfo", m.userInfoHandler.userInfo)
	apiGroup.POST("/oidc/userinfo", m.userInfoHandler.userInfo)

	apiGroup.POST("/oidc/introspect", m.introspectionHandler.introspectToken)

	apiGroup.GET("/oidc/end-session", optionalBrowserAuth, m.endSessionHandler.endSession)
	apiGroup.POST("/oidc/end-session", optionalBrowserAuth, m.endSessionHandler.endSession)

	apiGroup.POST("/oidc/device/authorize", m.deviceHandler.authorizeDevice)
	apiGroup.POST("/oidc/device/verify", browserAuth, m.deviceHandler.verifyDeviceCode)
	apiGroup.GET("/oidc/device/info", browserAuth, m.deviceHandler.deviceCodeInfo)
}
