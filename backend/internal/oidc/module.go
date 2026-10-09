package oidc

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	francishost "github.com/italypaleale/francis/host"
	"github.com/lestrrat-go/jwx/v4/jwa"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/auditlogs"
	"github.com/pocket-id/pocket-id/backend/internal/authz"
	"github.com/pocket-id/pocket-id/backend/internal/model"
)

type Config struct {
	BaseURL                   string
	TokenBaseURL              string
	Secret                    []byte
	AllowInsecureCallbackURLs bool
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
	Create(ctx context.Context, event auditlogs.Event, ipAddress, userAgent, userID string, data auditlogs.Data, tx *gorm.DB) (auditlogs.AuditLog, bool)
}

type Dependencies struct {
	DB         *gorm.DB
	Actors     francishost.Host
	Config     Config
	HTTPClient *http.Client

	GetCIMDURLAllowlist func() []string

	Signer       TokenSigner
	CustomClaims CustomClaimSource
	Reauth       ReauthenticationTokenConsumer
	AuditLog     AuditLogger
	APIAccess    APIAccessProvider

	// CleanupDisabled skips registering the cron jobs that delete expired rows from the database, for example in tests
	CleanupDisabled bool
}

type Module struct {
	Preview *ClientPreviewBuilder

	config       Config
	store        *Store
	cimdResolver *cimdClientResolver

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
	cimdResolver := newCIMDClientResolver(store, cimdResolverConfig{
		getURLAllowlist: deps.GetCIMDURLAllowlist,
		transportDecorator: func(transport http.RoundTripper) http.RoundTripper {
			return otelhttp.NewTransport(transport)
		},
	})
	store.clientResolver = cimdResolver

	authenticator, err := newFederatedClientAuthenticator(ctx, store, deps.HTTPClient, deps.Config.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create federated client authenticator: %w", err)
	}
	provider, err := newProvider(store, authenticator, deps.Signer, deps.Config, cimdResolver)
	if err != nil {
		return nil, fmt.Errorf("failed to create OAuth2 provider: %w", err)
	}

	claimsService := newClaimsService(deps.DB, deps.CustomClaims, deps.Config.BaseURL, deps.Signer)
	previewBuilder := newClientPreviewBuilder(claimsService, provider.tokenStrategies)
	interactionSessionService := newInteractionSessionService(deps.DB)
	authorizationService := newAuthorizationService(deps.DB, interactionSessionService, claimsService, deps.Reauth, deps.AuditLog, deps.APIAccess)
	deviceService := newDeviceService(provider, store, provider.deviceStrategy, authorizationService, claimsService, deps.AuditLog, deps.DB)
	endSessionService := newEndSessionService(deps.DB, store, deps.Signer, deps.Config.BaseURL)

	// Register the cleanup jobs for expired OIDC rows
	if !deps.CleanupDisabled {
		if deps.Actors == nil {
			return nil, errors.New("actor host is required for the OIDC cleanup cron jobs")
		}

		jobs, err := newCleanupJobs(deps.DB)
		if err != nil {
			return nil, err
		}

		for _, cj := range jobs {
			err = deps.Actors.RegisterBuiltInActor(cj)
			if err != nil {
				return nil, fmt.Errorf("error registering OIDC cleanup cron actor %q: %w", cj.ActorType(), err)
			}
		}
	}

	return &Module{
		Preview: previewBuilder,

		config:       deps.Config,
		store:        store,
		cimdResolver: cimdResolver,

		authorizationHandler: newAuthorizationHandler(provider, authorizationService),
		tokenHandler:         newTokenHandler(provider, claimsService, deps.APIAccess),
		userInfoHandler:      newUserInfoHandler(provider, claimsService, deps.Config.BaseURL),
		parHandler:           newPARHandler(provider),
		introspectionHandler: newIntrospectionHandler(provider, authenticator, deps.Config.BaseURL),
		endSessionHandler:    newEndSessionHandler(endSessionService, deps.Config.BaseURL),
		deviceHandler:        newDeviceHandler(provider, deviceService),
	}, nil
}

// RefreshClientMetadata forces a re-fetch of the OAuth Client ID Metadata Document.
func (m *Module) RefreshClientMetadata(ctx context.Context, clientID string) (model.OidcClient, error) {
	return m.cimdResolver.RefreshMetadataClient(ctx, clientID)
}

func (m *Module) RegisterRoutes(root, api *authz.Router) {
	root.Optional().GET("/authorize", authz.AccountSession, m.authorizationHandler.authorize)
	root.Optional().POST("/authorize", authz.AccountSession, m.authorizationHandler.authorize)

	api.Public().GET("/oidc/interactions/:id", m.authorizationHandler.getInteractionSession)
	api.POST("/oidc/interactions/:id/complete", authz.AccountSession, m.authorizationHandler.completeInteraction)

	api.Public().POST("/oidc/par", m.parHandler.pushedAuthorizationRequest)

	api.Public().POST("/oidc/token", m.tokenHandler.token)

	api.Public().GET("/oidc/userinfo", m.userInfoHandler.userInfo)
	api.Public().POST("/oidc/userinfo", m.userInfoHandler.userInfo)

	api.Public().POST("/oidc/introspect", m.introspectionHandler.introspectToken)

	api.Optional().GET("/oidc/end-session", authz.AccountSession, m.endSessionHandler.endSession)
	api.Optional().POST("/oidc/end-session", authz.AccountSession, m.endSessionHandler.endSession)

	api.Public().POST("/oidc/device/authorize", m.deviceHandler.authorizeDevice)
	api.POST("/oidc/device/verify", authz.AccountSession, m.deviceHandler.verifyDeviceCode)
	api.GET("/oidc/device/info", authz.AccountSession, m.deviceHandler.deviceCodeInfo)
}
