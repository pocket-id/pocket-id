package middleware

import (
	"context"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/pocket-id/pocket-id/backend/internal/apikey"
	"github.com/pocket-id/pocket-id/backend/internal/appconfig"
	"github.com/pocket-id/pocket-id/backend/internal/apperror"
	"github.com/pocket-id/pocket-id/backend/internal/authz"
	"github.com/pocket-id/pocket-id/backend/internal/emailverification"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	"github.com/pocket-id/pocket-id/backend/internal/service"
	"github.com/pocket-id/pocket-id/backend/internal/utils/cookie"
)

// #nosec G101 -- this is the name of the header that carries the API key, not a credential
const apiKeyHeader = "X-API-Key"

// NewAuthorization creates the authorization middleware with every credential Pocket ID accepts
// A browser session is tried before an API key, so a signed-in browser is never mistaken for an API client
func NewAuthorization(apiKeyModule *apikey.Module, userService *service.UserService, jwtService *service.JwtService, appConfig appconfig.AppConfigResolver) *authz.Middleware {
	return authz.NewMiddleware(
		NewSessionAuthenticator(jwtService, userService, appConfig),
		NewAPIKeyAuthenticator(apiKeyModule, appConfig),
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
	user, err := a.apiKeyModule.ValidateApiKey(c.Request.Context(), c.GetHeader(apiKeyHeader))
	if err != nil {
		return nil, apperror.NotSignedIn()
	}
	if user.Disabled {
		return nil, apperror.UserDisabled()
	}

	principal := &authz.Principal{
		Kind:   authz.KindAPIKey,
		UserID: user.ID,
		Scopes: authz.UserScopes(user.IsAdmin, authz.KindAPIKey),
	}
	err = restrictUnverified(c.Request.Context(), a.appConfig, &user, principal)
	if err != nil {
		return nil, err
	}

	return principal, nil
}
