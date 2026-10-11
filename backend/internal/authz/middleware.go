package authz

import (
	"github.com/gin-gonic/gin"

	"github.com/pocket-id/pocket-id/backend/internal/apperror"
)

// Authenticator resolves a principal from one kind of credential
type Authenticator interface {
	// Kind reports the kind of principal this authenticator produces
	Kind() PrincipalKind

	// Present reports whether the request carries this authenticator's credential at all, without validating it
	Present(c *gin.Context) bool

	// Authenticate validates the credential and resolves the principal
	// It returns an error with code not_signed_in when the credential is invalid, so the next authenticator gets a chance
	// Any other error, such as a disabled user, rejects the request
	Authenticate(c *gin.Context) (*Principal, error)
}

// Middleware authenticates requests and enforces the scope each route declares
type Middleware struct {
	authenticators []Authenticator
	declared       map[string]struct{}
}

// NewMiddleware creates the authorization middleware
// Authenticators are tried in order and the first one that resolves a principal wins
func NewMiddleware(authenticators ...Authenticator) *Middleware {
	return &Middleware{
		authenticators: authenticators,
		declared:       make(map[string]struct{}),
	}
}

// Router wraps a gin router group so every route registered through it declares its access
func (m *Middleware) Router(group *gin.RouterGroup) *Router {
	return &Router{group: group, auth: m}
}

// IsDeclared reports whether the route was registered through a Router or PublicRouter, so a test can compare gin's route table against the declarations
func (m *Middleware) IsDeclared(method, path string) bool {
	_, ok := m.declared[routeKey(method, path)]
	return ok
}

func (m *Middleware) declare(method, path string) {
	m.declared[routeKey(method, path)] = struct{}{}
}

func routeKey(method, path string) string {
	return method + " " + path
}

// require returns the handler that enforces the scope on a route
// An optional route lets requests without a usable credential through as anonymous instead of rejecting them
func (m *Middleware) require(scope Scope, optional bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		principal, kindRejected, err := m.authenticate(c, scope)
		if err != nil {
			c.Abort()
			_ = c.Error(err)
			return
		}

		// Requests without a usable credential are anonymous
		if principal == nil {
			if optional {
				c.Next()
				return
			}

			c.Abort()
			if kindRejected {
				// Only API keys can be rejected by kind today, so the error tells the caller to use a browser session instead
				_ = c.Error(apperror.APIKeyAuthNotAllowed())
				return
			}
			_ = c.Error(apperror.NotSignedIn())
			return
		}

		// A valid credential without the scope is forbidden even on optional routes, so a caller is never silently downgraded to anonymous
		if !principal.Scopes.Has(scope) {
			c.Abort()
			_ = c.Error(apperror.MissingScope(string(scope)))
			return
		}

		SetPrincipal(c, principal)
		c.Next()
	}
}

// authenticate resolves the principal from the first credential that can hold the scope and validates
// Credentials whose kind can never hold the scope are not validated at all, and kindRejected reports that one was present
func (m *Middleware) authenticate(c *gin.Context, scope Scope) (principal *Principal, kindRejected bool, err error) {
	for _, authenticator := range m.authenticators {
		if !authenticator.Present(c) {
			continue
		}

		// Skip credentials that could never satisfy the route so they are not validated or marked as used
		if !scope.GrantableTo(authenticator.Kind()) {
			kindRejected = true
			continue
		}

		principal, err = authenticator.Authenticate(c)
		if err == nil {
			return principal, false, nil
		}

		// An invalid credential falls through to the next authenticator, while a valid but rejected one ends the request
		if !apperror.IsCode(err, apperror.CodeNotSignedIn) {
			return nil, false, err
		}
	}

	return nil, kindRejected, nil
}
