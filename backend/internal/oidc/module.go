package oidc

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-fuego/fuego"
	"github.com/go-fuego/fuego/option"
	"github.com/go-fuego/fuego/param"
	francishost "github.com/italypaleale/francis/host"
	"github.com/lestrrat-go/jwx/v4/jwa"
	"github.com/pocket-id/fosite"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/appconfig"
	"github.com/pocket-id/pocket-id/backend/internal/auditlogs"
	"github.com/pocket-id/pocket-id/backend/internal/authz"
	"github.com/pocket-id/pocket-id/backend/internal/dto"
	"github.com/pocket-id/pocket-id/backend/internal/httpserver"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	"github.com/pocket-id/pocket-id/backend/internal/openapi"
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
	apiAccess    APIAccessProvider

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
		apiAccess:    deps.APIAccess,

		authorizationHandler: newAuthorizationHandler(provider, authorizationService),
		tokenHandler:         newTokenHandler(provider, claimsService, deps.APIAccess, deps.Sessions),
		userInfoHandler:      newUserInfoHandler(provider, claimsService, deps.Config.BaseURL),
		parHandler:           newPARHandler(provider),
		introspectionHandler: newIntrospectionHandler(provider, authenticator, claimsService, deps.APIAccess, deps.Config.BaseURL),
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

// VerifyAccessToken validates an access token Pocket ID issued, including that it hasn't been revoked and that its client is still granted what it carries, and returns the request it was granted for
func (m *Module) VerifyAccessToken(ctx context.Context, token string) (fosite.AccessRequester, error) {
	tokenType, accessRequest, err := m.provider.IntrospectToken(ctx, token, fosite.AccessToken, NewEmptySession())
	if err != nil {
		return nil, err
	}
	if tokenType != fosite.AccessToken {
		return nil, fosite.ErrRequestUnauthorized.WithDescription("Only access tokens are accepted.")
	}

	_, err = validateGrantAPIAccess(ctx, m.apiAccess, m.store.issuer, accessRequest)
	if err != nil {
		return nil, err
	}
	return accessRequest, nil
}

func (m *Module) RegisterRoutes(root, api *authz.Router, logoSizeLimit gin.HandlerFunc) {
	// Browsers reach the authorize endpoint through a redirect, so authorization errors are shown on the interaction error page instead of as JSON
	authorize := root.Use(m.authorizationHandler.redirectAuthorizationErrors).Optional()
	authorize.GET("/authorize", authz.AccountAuthorize, httpserver.Endpoint{Handler: m.authorizationHandler.authorize})
	authorize.POST("/authorize", authz.AccountAuthorize, httpserver.Endpoint{Handler: m.authorizationHandler.authorize})

	root.Public().GET("/.well-known/jwks.json", httpserver.Handle(m.discoveryHandler.jwksHandler),
		option.Summary("Get JSON Web Key Set (JWKS)"),
		option.Description("Returns the JSON Web Key Set used for token verification"),
		option.Tags("Well Known"),
		openapi.Response[map[string]any](http.StatusOK, "{ \"keys\": []interface{} }"),
	)
	root.Public().GET("/.well-known/openid-configuration", httpserver.Handle(m.discoveryHandler.openIDConfigurationHandler),
		option.Summary("Get OpenID Connect discovery configuration"),
		option.Description("Returns the OpenID Connect discovery document with endpoints and capabilities"),
		option.Tags("Well Known"),
		openapi.Response[map[string]any](http.StatusOK, "OpenID Connect configuration"),
	)
	root.Public().GET("/.well-known/oauth-authorization-server", httpserver.Handle(m.discoveryHandler.oauthAuthorizationServerHandler),
		option.Summary("Get OAuth 2.0 authorization server metadata"),
		option.Description("Returns the RFC 8414 OAuth 2.0 authorization server metadata document with endpoints and capabilities"),
		option.Tags("Well Known"),
		openapi.Response[map[string]any](http.StatusOK, "OAuth 2.0 authorization server metadata"),
	)

	api.Public().GET("/oidc/interactions/:id", httpserver.Endpoint{Handler: m.authorizationHandler.getInteractionSession})
	api.POST("/oidc/interactions/:id/complete", authz.AccountAuthorize, httpserver.Endpoint{Handler: m.authorizationHandler.completeInteraction})

	api.Public().POST("/oidc/par", httpserver.Endpoint{Handler: m.parHandler.pushedAuthorizationRequest})

	api.Public().POST("/oidc/token", httpserver.Endpoint{Handler: m.tokenHandler.token})

	api.Public().GET("/oidc/userinfo", httpserver.Endpoint{Handler: m.userInfoHandler.userInfo},
		option.Summary("Get user information"),
		option.Description("Get user information based on the access token"),
		option.Tags("OIDC"),
		openapi.Response[map[string]any](http.StatusOK, "User claims based on requested scopes"),
	)
	api.Public().POST("/oidc/userinfo", httpserver.Endpoint{Handler: m.userInfoHandler.userInfo})

	api.Public().POST("/oidc/introspect", httpserver.Endpoint{Handler: m.introspectionHandler.introspectToken},
		option.Summary("Introspect OIDC tokens"),
		option.Description("Pass a token to verify if it is considered valid."),
		option.Tags("OIDC"),
		openapi.Body[introspectionForm]("application/x-www-form-urlencoded"),
		openapi.Response[map[string]any](http.StatusOK, "Response with the introspection result."),
	)

	api.Optional().GET("/oidc/end-session", authz.AccountSession, httpserver.Endpoint{Handler: m.endSessionHandler.endSession})
	api.Optional().POST("/oidc/end-session", authz.AccountSession, httpserver.Endpoint{Handler: m.endSessionHandler.endSession})
	api.Optional().GET("/oidc/end-session/request", authz.AccountSession, httpserver.Endpoint{Handler: m.endSessionHandler.getEndSessionRequest})
	api.Optional().POST("/oidc/end-session/confirm", authz.AccountSession, httpserver.Endpoint{Handler: m.endSessionHandler.confirmEndSession})

	api.Public().POST("/oidc/device/authorize", httpserver.Endpoint{Handler: m.deviceHandler.authorizeDevice})
	api.POST("/oidc/device/verify", authz.AccountAuthorize, httpserver.Endpoint{Handler: m.deviceHandler.verifyDeviceCode})
	api.GET("/oidc/device/info", authz.AccountAuthorize, httpserver.Endpoint{Handler: m.deviceHandler.deviceCodeInfo})

	admin := api.Group("/admin")

	// The preview renders a user's claims, so it is guarded by the user scope like the client preview
	admin.POST("/oidc/claim-mapping-policies/preview", authz.UsersRead, httpserver.BodyJSON(http.StatusOK, m.claimMappingsPreviewHandler.previewClaimMappings),
		option.Summary("Preview claim mappings"),
		option.Description("Get the claims that claim mappings, saved or not, release for a user and set of scopes"),
		option.Tags("OIDC Claim Mapping Policies"),
	)

	admin.GET("/oidc/clients", authz.OidcClientsRead, httpserver.JSON(http.StatusOK, m.clientHandler.listClientsHandler),
		option.Summary("List OIDC clients"),
		option.Description("Get a paginated list of OIDC clients with optional search and sorting"),
		option.Tags("OIDC"),
		option.Query("search", "Search term to filter clients by name"),
		openapi.List[model.OidcClient]("allowedUserGroups"),
	)
	admin.POST("/oidc/clients", authz.OidcClientsWrite, httpserver.BodyJSON(http.StatusCreated, m.clientHandler.createClientHandler),
		option.Summary("Create OIDC client"),
		option.Description("Create a new OIDC client"),
		option.Tags("OIDC"),
	)
	admin.GET("/oidc/clients/:id", authz.OidcClientsRead, httpserver.JSON(http.StatusOK, m.clientHandler.getClientHandler),
		option.Summary("Get OIDC client"),
		option.Description("Get detailed information about an OIDC client"),
		option.Tags("OIDC"),
		option.Path("id", "Client ID"),
	)
	updateClientDocs := option.Group(
		option.Summary("Update OIDC client"),
		option.Description("Update an existing OIDC client, where PATCH changes only the fields in the body"),
		option.Tags("OIDC"),
		option.Path("id", "Client ID"),
	)
	admin.PUT("/oidc/clients/:id", authz.OidcClientsWrite, httpserver.UpdateJSON(m.clientHandler.updateClientHandler), updateClientDocs)
	admin.PATCH("/oidc/clients/:id", authz.OidcClientsWrite, httpserver.UpdateJSON(m.clientHandler.updateClientHandler), updateClientDocs)
	admin.POST("/oidc/clients/:id/refresh", authz.OidcClientsWrite, httpserver.JSON(http.StatusOK, m.clientHandler.refreshClientMetadataHandler),
		option.Summary("Refresh client metadata document"),
		option.Description("Force a re-fetch of the OAuth Client ID Metadata Document for a CIMD client"),
		option.Tags("OIDC"),
		option.Path("id", "Client ID"),
	)
	admin.DELETE("/oidc/clients/:id", authz.OidcClientsWrite, httpserver.NoContent(m.clientHandler.deleteClientHandler),
		option.Summary("Delete OIDC client"),
		option.Description("Delete an OIDC client by ID"),
		option.Tags("OIDC"),
		option.Path("id", "Client ID"),
	)

	admin.PUT("/oidc/clients/:id/allowed-user-groups", authz.OidcClientsWrite, httpserver.BodyJSON(http.StatusOK, m.clientHandler.updateAllowedUserGroupsHandler),
		option.Summary("Update allowed user groups"),
		option.Description("Update the user groups allowed to access an OIDC client"),
		option.Tags("OIDC"),
		option.Path("id", "Client ID"),
	)
	admin.GET("/oidc/clients/:id/secrets", authz.OidcClientsRead, httpserver.JSON(http.StatusOK, m.clientHandler.listClientSecretsHandler),
		option.Summary("List client secrets"),
		option.Description("List the secrets of an OIDC client, without disclosing their values"),
		option.Tags("OIDC"),
		option.Path("id", "Client ID"),
	)
	admin.POST("/oidc/clients/:id/secrets", authz.OidcClientsWrite, httpserver.JSON(http.StatusCreated, m.clientHandler.createClientSecretHandler),
		option.Summary("Create client secret"),
		option.Description("Add a new secret to an OIDC client, leaving the existing ones usable. The value is only returned by this endpoint and cannot be retrieved later."),
		option.Tags("OIDC"),
		option.Path("id", "Client ID"),
		openapi.Body[dto.OidcClientSecretCreateDto](),
		// Without a body the secret gets the default expiration
		func(r *fuego.BaseRoute) { r.Operation.RequestBody.Value.Required = false },
	)
	admin.DELETE("/oidc/clients/:id/secrets/:secretId", authz.OidcClientsWrite, httpserver.NoContent(m.clientHandler.deleteClientSecretHandler),
		option.Summary("Delete client secret"),
		option.Description("Delete a single secret of an OIDC client, making it immediately unusable"),
		option.Tags("OIDC"),
		option.Path("id", "Client ID"),
		option.Path("secretId", "Client secret ID"),
	)

	api.Public().GET("/oidc/clients/:id/logo", httpserver.Handle(m.clientHandler.getClientLogoHandler),
		option.Summary("Get client logo"),
		option.Description("Get the logo image for an OIDC client"),
		option.Tags("OIDC"),
		option.Path("id", "Client ID"),
		option.QueryBool("light", "Light mode logo (true) or dark mode logo (false)", param.Default(true)),
		openapi.File("Logo image", "image/png", "image/jpeg", "image/svg+xml"),
	)
	admin.DELETE("/oidc/clients/:id/logo", authz.OidcClientsWrite, httpserver.NoContent(m.clientHandler.deleteClientLogoHandler),
		option.Summary("Delete client logo"),
		option.Description("Delete the logo for an OIDC client"),
		option.Tags("OIDC"),
		option.Path("id", "Client ID"),
		option.QueryBool("light", "Light mode logo (true) or dark mode logo (false)", param.Default(true)),
	)
	admin.With(logoSizeLimit).POST("/oidc/clients/:id/logo", authz.OidcClientsWrite, httpserver.NoContent(m.clientHandler.updateClientLogoHandler),
		option.Summary("Update client logo"),
		option.Description("Upload or update the logo for an OIDC client"),
		option.Tags("OIDC"),
		option.Path("id", "Client ID"),
		openapi.Upload("Logo image file (PNG, JPG, or SVG)"),
		option.QueryBool("light", "Light mode logo (true) or dark mode logo (false)", param.Default(true)),
	)

	// The preview renders a user's claims, so it is guarded by the user scope rather than the client scope
	admin.GET("/oidc/clients/:id/preview/:userId", authz.UsersRead, httpserver.JSON(http.StatusOK, m.clientHandler.getClientPreviewHandler),
		option.Summary("Preview OIDC client data for user"),
		option.Description("Get a preview of the OIDC data (ID token, access token, userinfo) that would be sent to the client for a specific user"),
		option.Tags("OIDC"),
		option.Path("id", "Client ID"),
		option.Path("userId", "User ID to preview data for"),
		option.Query("scopes", "Scopes to include in the preview (space-separated)"),
	)

	api.GET("/oidc/users/me/authorized-clients", authz.AccountApps, httpserver.JSON(http.StatusOK, m.clientHandler.listOwnAuthorizedClientsHandler),
		option.Summary("List authorized clients for current user"),
		option.Description("Get a paginated list of OIDC clients that the current user has authorized"),
		option.Tags("OIDC"),
		option.Query("search", "Search term to filter clients by name"),
		openapi.List[model.UserAuthorizedOidcClient]("name"),
		option.QueryBool("filters[hasLaunchURL]", "Filter clients by whether a launch URL is configured"),
	)
	admin.GET("/oidc/users/:id/authorized-clients", authz.UsersRead, httpserver.JSON(http.StatusOK, m.clientHandler.listAuthorizedClientsHandler),
		option.Summary("List authorized clients for a user"),
		option.Description("Get a paginated list of OIDC clients that a specific user has authorized"),
		option.Tags("OIDC"),
		option.Path("id", "User ID"),
		option.Query("search", "Search term to filter clients by name"),
		openapi.List[model.UserAuthorizedOidcClient]("name"),
		option.QueryBool("filters[hasLaunchURL]", "Filter clients by whether a launch URL is configured"),
	)

	api.DELETE("/oidc/users/me/authorized-clients/:clientId", authz.AccountApps, httpserver.NoContent(m.clientHandler.revokeOwnClientAuthorizationHandler),
		option.Summary("Revoke authorization for an OIDC client"),
		option.Description("Revoke the authorization for a specific OIDC client for the current user"),
		option.Tags("OIDC"),
		option.Path("clientId", "Client ID to revoke authorization for"),
	)

	api.GET("/oidc/users/me/clients", authz.AccountApps, httpserver.JSON(http.StatusOK, m.clientHandler.listOwnAccessibleClientsHandler),
		option.Summary("List accessible OIDC clients for current user"),
		option.Description("Get a list of OIDC clients that the current user can access"),
		option.Tags("OIDC"),
		option.Query("search", "Search term to filter clients by name"),
		openapi.List[model.OidcClient]("lastUsedAt"),
		option.QueryBool("filters[hasLaunchURL]", "Filter clients by whether a launch URL is configured"),
	)
}
