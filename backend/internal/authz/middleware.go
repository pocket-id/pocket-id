package authz

import (
	"maps"
	"slices"

	"github.com/gin-gonic/gin"
	"github.com/go-fuego/fuego"

	"github.com/pocket-id/pocket-id/backend/internal/apperror"
	"github.com/pocket-id/pocket-id/backend/internal/openapi"
)

// Authenticator resolves a principal from one kind of credential
type Authenticator interface {
	// Kinds reports every kind of principal this authenticator can produce, as a set of flags
	Kinds() PrincipalKind

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
	declared       map[string]openapi.Route
}

// NewMiddleware creates the authorization middleware
// Authenticators are tried in order and the first one that resolves a principal wins
func NewMiddleware(authenticators ...Authenticator) *Middleware {
	return &Middleware{
		authenticators: authenticators,
		declared:       make(map[string]openapi.Route),
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

// Routes returns every declared route
func (m *Middleware) Routes() []openapi.Route {
	return slices.Collect(maps.Values(m.declared))
}

// register adds the route to gin and declares it with its documentation
func (m *Middleware) register(group *gin.RouterGroup, method, relativePath string, chain []gin.HandlerFunc, docs []fuego.RouteOption) {
	group.Handle(method, relativePath, chain...)
	path := joinPaths(group.BasePath(), relativePath)
	m.declared[routeKey(method, path)] = openapi.Route{Method: method, Path: path, Docs: docs}
}

func routeKey(method, path string) string {
	return method + " " + path
}

// require returns the handler that enforces the scope on a route
// An optional route lets requests without a usable credential through as anonymous instead of rejecting them
func (m *Middleware) require(scope Scope, optional bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		principal, rejectedKind, err := m.authenticate(c, scope.GrantableTo)
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
			_ = c.Error(rejectionError(rejectedKind))
			return
		}

		// A valid credential without the scope is forbidden even on optional routes, so a caller is never silently downgraded to anonymous
		if !principal.Scopes.Has(scope) {
			c.Abort()
			// A scope that is only withheld until the email is verified tells the user how to regain it
			if principal.WithheldUntilVerified.Has(scope) {
				_ = c.Error(apperror.EmailVerificationRequired())
				return
			}
			_ = c.Error(apperror.MissingScope(string(scope)))
			return
		}

		SetPrincipal(c, principal)
		c.Next()
	}
}

// requireDispatch returns the handler of a dispatch route, which authenticates every credential except a browser session and requires some admin scope
func (m *Middleware) requireDispatch() gin.HandlerFunc {
	return func(c *gin.Context) {
		principal, rejectedKind, err := m.authenticate(c, func(kind PrincipalKind) bool { return kind != KindSession })
		if err != nil {
			c.Abort()
			_ = c.Error(err)
			return
		}
		if principal == nil {
			c.Abort()
			_ = c.Error(rejectionError(rejectedKind))
			return
		}

		// Dispatch routes only serve admin operations, so a caller without any admin scope is turned away before the handler
		if !principal.Scopes.hasAdmin() {
			c.Abort()
			_ = c.Error(apperror.MissingPermission())
			return
		}

		SetPrincipal(c, principal)
		c.Next()
	}
}

// rejectionError tells the caller which credential a route needs when the one it sent was rejected by kind
func rejectionError(rejectedKind PrincipalKind) error {
	if rejectedKind&(KindAPIKey|KindStaticAPIKey) != 0 {
		return apperror.APIKeyAuthNotAllowed()
	}
	if rejectedKind == KindOAuthClient {
		return apperror.ClientTokenAuthNotAllowed()
	}
	if rejectedKind&KindOAuthUser != 0 {
		return apperror.OAuthTokenAuthNotAllowed()
	}
	return apperror.NotSignedIn()
}

// authenticate resolves the principal from the first credential whose kind the route accepts and that validates
// Credentials of kinds the route never accepts are not validated at all, and rejectedKind reports the last one that was present
func (m *Middleware) authenticate(c *gin.Context, accepts func(PrincipalKind) bool) (principal *Principal, rejectedKind PrincipalKind, err error) {
	for _, authenticator := range m.authenticators {
		if !authenticator.Present(c) {
			continue
		}

		// Skip credentials that could never satisfy the route so they are not validated or marked as used
		if !accepts(authenticator.Kinds()) {
			rejectedKind = authenticator.Kinds()
			continue
		}

		principal, err = authenticator.Authenticate(c)
		// An authenticator that produces several kinds only knows the kind once the credential is validated
		if err == nil && !accepts(principal.Kind) {
			rejectedKind = principal.Kind
			continue
		}
		if err == nil {
			return principal, 0, nil
		}

		// An invalid credential falls through to the next authenticator, while a valid but rejected one ends the request
		if !apperror.IsCode(err, apperror.CodeNotSignedIn) {
			return nil, 0, err
		}
	}

	return nil, rejectedKind, nil
}
