package authz

import (
	"time"

	"github.com/gin-gonic/gin"
)

const principalContextKey = "authz.principal"

// Principal is the authenticated caller of a request together with the scopes it holds
type Principal struct {
	Kind   PrincipalKind
	UserID string
	Scopes ScopeSet

	// WithheldUntilVerified are the scopes the user's role grants but that are withheld until the user verifies their email
	WithheldUntilVerified ScopeSet

	// AuthenticationMethod and AuthenticationTime describe how the session was established and are only set for KindSession
	AuthenticationMethod string
	AuthenticationTime   time.Time

	// SessionID is the jti of the session access token, only set for KindSession
	SessionID string
}

// RestrictUnverified withholds every scope except those a user needs to verify their email, such as signing out or correcting the address
func (p *Principal) RestrictUnverified() {
	kept := make(ScopeSet, len(unverifiedScopes))
	withheld := make(ScopeSet, len(p.Scopes))
	for scope := range p.Scopes {
		if unverifiedScopes.Has(scope) {
			kept[scope] = struct{}{}
		} else {
			withheld[scope] = struct{}{}
		}
	}
	p.Scopes = kept
	p.WithheldUntilVerified = withheld
}

// PrincipalFrom returns the principal the authorization middleware attached to the request
// Anonymous requests on optional and public routes get the zero Principal, whose UserID is empty
func PrincipalFrom(c *gin.Context) Principal {
	value, _ := c.Get(principalContextKey)
	if principal, ok := value.(*Principal); ok && principal != nil {
		return *principal
	}
	return Principal{}
}

// SetPrincipal attaches the principal to the request
// The middleware calls it after authorizing a request, and tests use it to call handlers directly
func SetPrincipal(c *gin.Context, principal *Principal) {
	c.Set(principalContextKey, principal)
}
