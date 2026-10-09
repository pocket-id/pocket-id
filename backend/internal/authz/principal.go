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

	// AuthenticationMethod and AuthenticationTime describe how the session was established and are only set for KindSession
	AuthenticationMethod string
	AuthenticationTime   time.Time
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
