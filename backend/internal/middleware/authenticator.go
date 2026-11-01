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
	"github.com/pocket-id/pocket-id/backend/internal/service"
	"github.com/pocket-id/pocket-id/backend/internal/utils/cookie"
)

// #nosec G101 -- this is the name of the header that carries the API key, not a credential
const apiKeyHeader = "X-API-Key"

// NewAuthorization creates the authorization middleware with every credential Pocket ID accepts
// A browser session is tried before an API key or OAuth token, so a signed-in browser is never mistaken for an API client
// The session authenticator also reads the Authorization header, but an OAuth access token fails its signature check and falls through to the OAuth authenticator
func NewAuthorization(apiKeyModule *apikey.Module, accessTokenVerifier AccessTokenVerifier, userService *service.UserService, jwtService *service.JwtService, appConfig appconfig.AppConfigResolver) *authz.Middleware {
	return authz.NewMiddleware(
		NewSessionAuthenticator(jwtService, userService, appConfig),
		NewAPIKeyAuthenticator(apiKeyModule, appConfig),
		NewOAuthAuthenticator(accessTokenVerifier, userService, appConfig),
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

// SessionAuthenticator authenticates the session access token Pocket ID issues after sign-in
type SessionAuthenticator struct {
	jwtService  *service.JwtService
	userService *service.UserService
	appConfig   appconfig.AppConfigResolver
}

func NewSessionAuthenticator(jwtService *service.JwtService, userService *service.UserService, appConfig appconfig.AppConfigResolver) *SessionAuthenticator {
	return &SessionAuthenticator{jwtService: jwtService, userService: userService, appConfig: appConfig}
}

func (a *SessionAuthenticator) Kind() authz.PrincipalKind {
	return authz.KindSession
}

func (a *SessionAuthenticator) Present(c *gin.Context) bool {
	return sessionToken(c) != ""
}

func (a *SessionAuthenticator) Authenticate(c *gin.Context) (*authz.Principal, error) {
	// Verify the token signature, audience and type
	token, err := a.jwtService.VerifyAccessToken(sessionToken(c))
	if err != nil {
		return nil, apperror.NotSignedIn()
	}
	authenticationMethod, err := a.jwtService.GetAuthenticationMethod(token)
	if err != nil {
		return nil, apperror.NotSignedIn()
	}
	authenticationTime, _ := token.IssuedAt()

	subject, ok := token.Subject()
	if !ok {
		return nil, apperror.TokenInvalid()
	}

	// Load the user so disabling an account, changing its admin flag or its email verification takes effect before the token expires
	user, err := a.userService.GetUser(c, subject)
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
		AuthenticationMethod: authenticationMethod,
		AuthenticationTime:   authenticationTime,
	}
	err = restrictUnverified(c.Request.Context(), a.appConfig, &user, principal)
	if err != nil {
		return nil, err
	}

	return principal, nil
}

// sessionToken reads the session access token from its cookie, or from the Authorization header when the cookie is absent
// An invalid cookie deliberately does not fall back to the header
func sessionToken(c *gin.Context) string {
	if accessToken, err := c.Cookie(cookie.AccessTokenCookieName); err == nil {
		return accessToken
	}

	_, accessToken, _ := strings.Cut(c.GetHeader("Authorization"), " ")
	return accessToken
}

// APIKeyAuthenticator authenticates a personal API key sent in the X-API-Key header
type APIKeyAuthenticator struct {
	apiKeyModule *apikey.Module
	appConfig    appconfig.AppConfigResolver
}

func NewAPIKeyAuthenticator(apiKeyModule *apikey.Module, appConfig appconfig.AppConfigResolver) *APIKeyAuthenticator {
	return &APIKeyAuthenticator{apiKeyModule: apiKeyModule, appConfig: appConfig}
}

func (a *APIKeyAuthenticator) Kind() authz.PrincipalKind {
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

// OAuthAuthenticator authenticates an OAuth access token that a client obtained for Pocket ID's own API on behalf of a user
type OAuthAuthenticator struct {
	verifier    AccessTokenVerifier
	userService *service.UserService
	appConfig   appconfig.AppConfigResolver
	audience    string
}

func NewOAuthAuthenticator(verifier AccessTokenVerifier, userService *service.UserService, appConfig appconfig.AppConfigResolver) *OAuthAuthenticator {
	return &OAuthAuthenticator{verifier: verifier, userService: userService, appConfig: appConfig, audience: authz.ResourceURI(common.EnvConfig.AppURL)}
}

func (a *OAuthAuthenticator) Kind() authz.PrincipalKind {
	return authz.KindOAuthUser
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

	// Load the user so disabling the account or changing its role takes effect before the token expires
	user, err := a.userService.GetUser(ctx, accessRequest.GetSession().GetSubject())
	if err != nil {
		return nil, apperror.NotSignedIn()
	}
	if user.Disabled {
		return nil, apperror.UserDisabled()
	}

	// A token never holds more than its user currently does, so demoting the user narrows their tokens right away
	tokenScopes := make(authz.ScopeSet, len(accessRequest.GetGrantedScopes()))
	for _, scope := range accessRequest.GetGrantedScopes() {
		tokenScopes[authz.Scope(scope)] = struct{}{}
	}
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
