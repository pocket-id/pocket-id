package oidc

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	francishost "github.com/italypaleale/francis/host"
	"github.com/lestrrat-go/jwx/v4/jwa"
	"github.com/pocket-id/fosite"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/appconfig"
	"github.com/pocket-id/pocket-id/backend/internal/auditlogs"
	"github.com/pocket-id/pocket-id/backend/internal/authz"
	"github.com/pocket-id/pocket-id/backend/internal/httpserver"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	"github.com/pocket-id/pocket-id/backend/internal/storage"
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

// Signer is the instance key, which signs tokens and is published for the discovery endpoints
type Signer interface {
	TokenSigner
	KeyPublisher
}

type CustomClaimSource interface {
	GetCustomClaimsForUserWithUserGroups(ctx context.Context, userID string, tx *gorm.DB) ([]model.CustomClaim, error)
}

type ReauthenticationTokenConsumer interface {
	ConsumeReauthenticationToken(ctx context.Context, tx *gorm.DB, token string, userID string) (time.Time, error)
}

// Sessions gives access to the Pocket ID sessions grants are issued in
type Sessions interface {
	IsActive(ctx context.Context, userID, sessionID string) (bool, error)
	Revoke(ctx context.Context, userID, sessionID string, origin auditlogs.Origin) error
}

type AuditLogger interface {
	Create(ctx context.Context, event auditlogs.Event, ipAddress, userAgent, userID string, data auditlogs.Data, tx *gorm.DB) (auditlogs.AuditLog, bool)
	Record(ctx context.Context, event auditlogs.Event, origin auditlogs.Origin, userID string, data auditlogs.Data, tx *gorm.DB)
}

// ScimSyncScheduler schedules SCIM after a client change has committed
type ScimSyncScheduler interface {
	ScheduleSync(ctx context.Context)
}

// BackchannelLogout tells clients to end the sessions of users who lost access to them
type BackchannelLogout interface {
	PrepareAuthorizationNotification(ctx context.Context, tx *gorm.DB, userID string, clientID string) (func(), error)
	PrepareClientNotifications(ctx context.Context, tx *gorm.DB, clientID string) (func(), error)
	NotifyLostGroupAccess(ctx context.Context, userIDs []string, clientID string)
}

type Dependencies struct {
	DB                  *gorm.DB
	Actors              francishost.Host
	Config              Config
	FederatedJWKSClient *http.Client
	CIMDTransport       http.RoundTripper

	GetCIMDURLAllowlist func() []string

	// LogoHTTPClient downloads the logos admins set by URL
	LogoHTTPClient *http.Client
	FileStorage    storage.FileStorage

	Signer            Signer
	CustomClaims      CustomClaimSource
	Reauth            ReauthenticationTokenConsumer
	AuditLog          AuditLogger
	Sessions          Sessions
	APIAccess         APIAccessProvider
	AppConfig         appconfig.AppConfigResolver
	ScimSync          ScimSyncScheduler
	BackchannelLogout BackchannelLogout

	// CleanupDisabled skips registering the cron jobs that delete expired rows from the database, for example in tests
	CleanupDisabled bool
}

type Module struct {
	config       Config
	provider     fosite.OAuth2Provider
	store        *Store
	cimdResolver *cimdClientResolver

	authorizationHandler *authorizationHandler
	tokenHandler         *tokenHandler
	userInfoHandler      *userInfoHandler
	parHandler           *parHandler
	introspectionHandler *introspectionHandler
	endSessionHandler    *endSessionHandler
	deviceHandler        *deviceHandler

	claimMappingsPreviewHandler *claimMappingsPreviewHandler
	discoveryHandler            *discoveryHandler
	clientHandler               *clientHandler
}

func New(ctx context.Context, deps Dependencies) (*Module, error) {
	store := NewStore(deps.DB, deps.APIAccess).WithIssuer(deps.Config.BaseURL)
	cimdResolver := newCIMDClientResolver(store, cimdResolverConfig{
		getURLAllowlist: deps.GetCIMDURLAllowlist,
		transport:       deps.CIMDTransport,
		transportDecorator: func(transport http.RoundTripper) http.RoundTripper {
			return otelhttp.NewTransport(transport)
		},
	})
	store.clientResolver = cimdResolver

	authenticator, err := newFederatedClientAuthenticator(ctx, store, deps.FederatedJWKSClient, deps.Config.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create federated client authenticator: %w", err)
	}
	provider, err := newProvider(store, authenticator, deps.Signer, deps.Config, cimdResolver)
	if err != nil {
		return nil, fmt.Errorf("failed to create OAuth2 provider: %w", err)
	}

	claimsService := newClaimsService(deps.DB, deps.CustomClaims, deps.Config.BaseURL, deps.Signer, deps.AppConfig)
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
		config:       deps.Config,
		provider:     provider,
		store:        store,
		cimdResolver: cimdResolver,

		authorizationHandler: newAuthorizationHandler(provider, authorizationService),
		tokenHandler:         newTokenHandler(provider, claimsService, deps.APIAccess, deps.Sessions),
		userInfoHandler:      newUserInfoHandler(provider, claimsService, deps.Config.BaseURL),
		parHandler:           newPARHandler(provider),
		introspectionHandler: newIntrospectionHandler(provider, authenticator, claimsService, deps.Config.BaseURL),
		endSessionHandler:    newEndSessionHandler(endSessionService, deps.Sessions, deps.Config.BaseURL),
		deviceHandler:        newDeviceHandler(provider, deviceService),

		claimMappingsPreviewHandler: &claimMappingsPreviewHandler{claimsService: claimsService},
		discoveryHandler:            &discoveryHandler{db: deps.DB, keys: deps.Signer, getCIMDURLAllowlist: deps.GetCIMDURLAllowlist},
		clientHandler: &clientHandler{
			service:   newClientService(deps.DB, previewBuilder, cimdResolver, deps.ScimSync, deps.BackchannelLogout, deps.AuditLog, deps.LogoHTTPClient, deps.FileStorage),
			appConfig: deps.AppConfig,
		},
	}, nil
}

// VerifyAccessToken validates an access token Pocket ID issued, including that it hasn't been revoked, and returns the request it was granted for
func (m *Module) VerifyAccessToken(ctx context.Context, token string) (fosite.AccessRequester, error) {
	tokenType, accessRequest, err := m.provider.IntrospectToken(ctx, token, fosite.AccessToken, NewEmptySession())
	if err != nil {
		return nil, err
	}
	if tokenType != fosite.AccessToken {
		return nil, fosite.ErrRequestUnauthorized.WithDescription("Only access tokens are accepted.")
	}
	return accessRequest, nil
}

func (m *Module) RegisterRoutes(root, api *authz.Router, logoSizeLimit gin.HandlerFunc) {
	// Browsers reach the authorize endpoint through a redirect, so authorization errors are shown on the interaction error page instead of as JSON
	authorize := root.Use(m.authorizationHandler.redirectAuthorizationErrors).Optional()
	authorize.GET("/authorize", authz.AccountAuthorize, m.authorizationHandler.authorize)
	authorize.POST("/authorize", authz.AccountAuthorize, m.authorizationHandler.authorize)

	root.Public().GET("/.well-known/jwks.json", httpserver.Handle(m.discoveryHandler.jwksHandler))
	root.Public().GET("/.well-known/openid-configuration", httpserver.Handle(m.discoveryHandler.openIDConfigurationHandler))
	root.Public().GET("/.well-known/oauth-authorization-server", httpserver.Handle(m.discoveryHandler.oauthAuthorizationServerHandler))

	api.Public().GET("/oidc/interactions/:id", m.authorizationHandler.getInteractionSession)
	api.POST("/oidc/interactions/:id/complete", authz.AccountAuthorize, m.authorizationHandler.completeInteraction)

	api.Public().POST("/oidc/par", m.parHandler.pushedAuthorizationRequest)

	api.Public().POST("/oidc/token", m.tokenHandler.token)

	api.Public().GET("/oidc/userinfo", m.userInfoHandler.userInfo)
	api.Public().POST("/oidc/userinfo", m.userInfoHandler.userInfo)

	api.Public().POST("/oidc/introspect", m.introspectionHandler.introspectToken)

	api.Optional().GET("/oidc/end-session", authz.AccountSession, m.endSessionHandler.endSession)
	api.Optional().POST("/oidc/end-session", authz.AccountSession, m.endSessionHandler.endSession)
	api.Optional().GET("/oidc/end-session/request", authz.AccountSession, m.endSessionHandler.getEndSessionRequest)
	api.Optional().POST("/oidc/end-session/confirm", authz.AccountSession, m.endSessionHandler.confirmEndSession)

	api.Public().POST("/oidc/device/authorize", m.deviceHandler.authorizeDevice)
	api.POST("/oidc/device/verify", authz.AccountAuthorize, m.deviceHandler.verifyDeviceCode)
	api.GET("/oidc/device/info", authz.AccountAuthorize, m.deviceHandler.deviceCodeInfo)

	// The preview renders a user's claims, so it is guarded by the user scope like the client preview
	api.POST("/oidc/claim-mapping-policies/preview", authz.UsersRead, httpserver.Handle(m.claimMappingsPreviewHandler.previewClaimMappings))

	api.GET("/oidc/clients", authz.OidcClientsRead, httpserver.Handle(m.clientHandler.listClientsHandler))
	api.POST("/oidc/clients", authz.OidcClientsWrite, httpserver.Handle(m.clientHandler.createClientHandler))
	api.GET("/oidc/clients/:id", authz.OidcClientsRead, httpserver.Handle(m.clientHandler.getClientHandler))
	api.PUT("/oidc/clients/:id", authz.OidcClientsWrite, httpserver.Handle(m.clientHandler.updateClientHandler))
	api.POST("/oidc/clients/:id/refresh", authz.OidcClientsWrite, httpserver.Handle(m.clientHandler.refreshClientMetadataHandler))
	api.DELETE("/oidc/clients/:id", authz.OidcClientsWrite, httpserver.Handle(m.clientHandler.deleteClientHandler))

	api.PUT("/oidc/clients/:id/allowed-user-groups", authz.OidcClientsWrite, httpserver.Handle(m.clientHandler.updateAllowedUserGroupsHandler))
	api.GET("/oidc/clients/:id/secrets", authz.OidcClientsRead, httpserver.Handle(m.clientHandler.listClientSecretsHandler))
	api.POST("/oidc/clients/:id/secrets", authz.OidcClientsWrite, httpserver.Handle(m.clientHandler.createClientSecretHandler))
	api.DELETE("/oidc/clients/:id/secrets/:secretId", authz.OidcClientsWrite, httpserver.Handle(m.clientHandler.deleteClientSecretHandler))

	api.Public().GET("/oidc/clients/:id/logo", httpserver.Handle(m.clientHandler.getClientLogoHandler))
	api.DELETE("/oidc/clients/:id/logo", authz.OidcClientsWrite, httpserver.Handle(m.clientHandler.deleteClientLogoHandler))
	api.POST("/oidc/clients/:id/logo", authz.OidcClientsWrite, logoSizeLimit, httpserver.Handle(m.clientHandler.updateClientLogoHandler))

	// The preview renders a user's claims, so it is guarded by the user scope rather than the client scope
	api.GET("/oidc/clients/:id/preview/:userId", authz.UsersRead, httpserver.Handle(m.clientHandler.getClientPreviewHandler))

	api.GET("/oidc/users/me/authorized-clients", authz.AccountApps, httpserver.Handle(m.clientHandler.listOwnAuthorizedClientsHandler))
	api.GET("/oidc/users/:id/authorized-clients", authz.UsersRead, httpserver.Handle(m.clientHandler.listAuthorizedClientsHandler))

	api.DELETE("/oidc/users/me/authorized-clients/:clientId", authz.AccountApps, httpserver.Handle(m.clientHandler.revokeOwnClientAuthorizationHandler))

	api.GET("/oidc/users/me/clients", authz.AccountApps, httpserver.Handle(m.clientHandler.listOwnAccessibleClientsHandler))
}
