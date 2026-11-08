package middleware

import (
	"context"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/pocket-id/fosite"

	"github.com/pocket-id/pocket-id/backend/internal/apikey"
	"github.com/pocket-id/pocket-id/backend/internal/appconfig"
	"github.com/pocket-id/pocket-id/backend/internal/apperror"
	"github.com/pocket-id/pocket-id/backend/internal/authz"
	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/emailverification"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	"github.com/pocket-id/pocket-id/backend/internal/oidc"
	"github.com/pocket-id/pocket-id/backend/internal/session"
	"github.com/pocket-id/pocket-id/backend/internal/utils/cookie"
)

// #nosec G101 -- this is the name of the header that carries the API key, not a credential
const apiKeyHeader = "X-API-Key"

// NewAuthorization creates the authorization middleware with every credential Pocket ID accepts
// A browser session is tried before an API key or OAuth token, so a signed-in browser is never mistaken for an API client
func NewAuthorization(apiKeyModule *apikey.Module, accessTokenVerifier AccessTokenVerifier, users UserProvider, sessions SessionValidator, appConfig appconfig.AppConfigResolver) *authz.Middleware {
	return authz.NewMiddleware(
		NewSessionAuthenticator(sessions, users, appConfig),
		NewAPIKeyAuthenticator(apiKeyModule, appConfig),
		NewOAuthAuthenticator(accessTokenVerifier, users, appConfig),
	)
}

// restrictUnverified withholds the scopes a user who still has to verify their email may not use yet
func restrictUnverified(ctx context.Context, appConfig appconfig.AppConfigResolver, user *model.User, principal *authz.Principal) error {
	if !emailverification.CanBePending(user) {
		return nil
	}

	cfg, err := appConfig.GetConfig(ctx)
	if err != nil {
		return err
	}
	if emailverification.IsPending(cfg, user) {
		principal.RestrictUnverified()
	}

	return nil
}

// SessionValidator resolves a session token to its user and session, which the session module implements
type SessionValidator interface {
	Validate(ctx context.Context, token string) (string, session.Session, error)
}

// UserProvider loads the user a credential belongs to, which the users module implements
type UserProvider interface {
	GetUser(ctx context.Context, userID string) (model.User, error)
}

// SessionAuthenticator authenticates the session cookie Pocket ID sets after sign-in
type SessionAuthenticator struct {
	sessions  SessionValidator
	users     UserProvider
	appConfig appconfig.AppConfigResolver
}

func NewSessionAuthenticator(sessions SessionValidator, users UserProvider, appConfig appconfig.AppConfigResolver) *SessionAuthenticator {
	return &SessionAuthenticator{sessions: sessions, users: users, appConfig: appConfig}
}

func (a *SessionAuthenticator) Kinds() authz.PrincipalKind {
	return authz.KindSession
}

func (a *SessionAuthenticator) Present(c *gin.Context) bool {
	_, err := c.Cookie(cookie.SessionTokenCookieName)
	return err == nil
}

func (a *SessionAuthenticator) Authenticate(c *gin.Context) (*authz.Principal, error) {
	token, _ := c.Cookie(cookie.SessionTokenCookieName)
	userID, userSession, err := a.sessions.Validate(c.Request.Context(), token)
	if err != nil {
		return nil, err
	}

	// Load the user so disabling an account, changing its admin flag or its email verification takes effect right away
	user, err := a.users.GetUser(c, userID)
	if err != nil {
		return nil, apperror.NotSignedIn()
	}
	if user.Disabled {
		return nil, apperror.UserDisabled()
	}

	principal := &authz.Principal{
		Kind:                 authz.KindSession,
		UserID:               user.ID,
		Scopes:               authz.UserScopes(user.IsAdmin, authz.KindSession),
		SessionID:            userSession.ID,
		AuthenticationMethod: userSession.AuthenticationMethod,
		AuthenticationTime:   userSession.CreatedAt,
	}
	err = restrictUnverified(c.Request.Context(), a.appConfig, &user, principal)
	if err != nil {
		return nil, err
	}

	return principal, nil
}

// APIKeyAuthenticator authenticates a personal API key sent in the X-API-Key header
type APIKeyAuthenticator struct {
	apiKeyModule *apikey.Module
	appConfig    appconfig.AppConfigResolver
}

func NewAPIKeyAuthenticator(apiKeyModule *apikey.Module, appConfig appconfig.AppConfigResolver) *APIKeyAuthenticator {
	return &APIKeyAuthenticator{apiKeyModule: apiKeyModule, appConfig: appConfig}
}

func (a *APIKeyAuthenticator) Kinds() authz.PrincipalKind {
	return authz.KindAPIKey
}

func (a *APIKeyAuthenticator) Present(c *gin.Context) bool {
	return c.GetHeader(apiKeyHeader) != ""
}

func (a *APIKeyAuthenticator) Authenticate(c *gin.Context) (*authz.Principal, error) {
	key, err := a.apiKeyModule.ValidateApiKey(c.Request.Context(), c.GetHeader(apiKeyHeader))
	if err != nil {
		return nil, apperror.NotSignedIn()
	}
	user := key.User
	if user.Disabled {
		return nil, apperror.UserDisabled()
	}

	// A key never holds more than its owner currently does, so demoting the owner narrows their keys right away
	keyScopes := make(authz.ScopeSet, len(key.Scopes))
	for _, scope := range key.Scopes {
		keyScopes[authz.Scope(scope)] = struct{}{}
	}
	principal := &authz.Principal{
		Kind:   authz.KindAPIKey,
		UserID: user.ID,
		Scopes: authz.UserScopes(user.IsAdmin, authz.KindAPIKey).Intersect(keyScopes),
	}
	err = restrictUnverified(c.Request.Context(), a.appConfig, &user, principal)
	if err != nil {
		return nil, err
	}

	return principal, nil
}

// AccessTokenVerifier validates an OAuth access token Pocket ID issued, which the oidc module implements
type AccessTokenVerifier interface {
	VerifyAccessToken(ctx context.Context, token string) (fosite.AccessRequester, error)
}

// OAuthAuthenticator authenticates an OAuth access token that a client obtained for Pocket ID's own API, either on behalf of a user or for itself
type OAuthAuthenticator struct {
	verifier  AccessTokenVerifier
	users     UserProvider
	appConfig appconfig.AppConfigResolver
	audience  string
}

func NewOAuthAuthenticator(verifier AccessTokenVerifier, users UserProvider, appConfig appconfig.AppConfigResolver) *OAuthAuthenticator {
	return &OAuthAuthenticator{verifier: verifier, users: users, appConfig: appConfig, audience: authz.ResourceURI(common.EnvConfig.AppURL)}
}

func (a *OAuthAuthenticator) Kinds() authz.PrincipalKind {
	return authz.KindOAuthUser | authz.KindOAuthClient
}

func (a *OAuthAuthenticator) Present(c *gin.Context) bool {
	return bearerToken(c) != ""
}

func (a *OAuthAuthenticator) Authenticate(c *gin.Context) (*authz.Principal, error) {
	ctx := c.Request.Context()
	accessRequest, err := a.verifier.VerifyAccessToken(ctx, bearerToken(c))
	if err != nil {
		return nil, apperror.NotSignedIn()
	}

	// Only a token issued for Pocket ID's own API may call it, which keeps a custom API's token with a colliding permission key out
	if !accessRequest.GetGrantedAudience().Has(a.audience) {
		return nil, apperror.NotSignedIn()
	}

	tokenScopes := make(authz.ScopeSet, len(accessRequest.GetGrantedScopes()))
	for _, scope := range accessRequest.GetGrantedScopes() {
		tokenScopes[authz.Scope(scope)] = struct{}{}
	}

	// A client acting as itself has no user or role, so its token is only limited to the scopes a client can ever hold
	clientID := accessRequest.GetClient().GetID()
	subject := accessRequest.GetSession().GetSubject()
	if subject == oidc.ClientSubject(clientID) {
		return &authz.Principal{
			Kind:     authz.KindOAuthClient,
			ClientID: clientID,
			Scopes:   authz.UserScopes(true, authz.KindOAuthClient).Intersect(tokenScopes),
		}, nil
	}

	// Load the user so disabling the account or changing its role takes effect before the token expires
	user, err := a.users.GetUser(ctx, subject)
	if err != nil {
		return nil, apperror.NotSignedIn()
	}
	if user.Disabled {
		return nil, apperror.UserDisabled()
	}

	// A token never holds more than its user currently does, so demoting the user narrows their tokens right away
	principal := &authz.Principal{
		Kind:   authz.KindOAuthUser,
		UserID: user.ID,
		Scopes: authz.UserScopes(user.IsAdmin, authz.KindOAuthUser).Intersect(tokenScopes),
	}
	err = restrictUnverified(ctx, a.appConfig, &user, principal)
	if err != nil {
		return nil, err
	}

	return principal, nil
}

// bearerToken reads the token of a Bearer Authorization header, whose scheme is case-insensitive
func bearerToken(c *gin.Context) string {
	scheme, token, ok := strings.Cut(c.GetHeader("Authorization"), " ")
	if !ok || !strings.EqualFold(scheme, "Bearer") {
		return ""
	}
	return token
}
