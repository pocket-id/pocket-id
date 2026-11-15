package authz

import (
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/pocket-id/pocket-id/backend/internal/apperror"
)

const principalContextKey = "authz.principal"

// Principal is the authenticated caller of a request together with the scopes it holds
type Principal struct {
	Kind PrincipalKind
	// UserID is empty for KindOAuthClient, where ClientID identifies the client acting as itself instead, and for KindStaticAPIKey
	UserID string
	// ClientID is the client the OAuth access token was issued to, for KindOAuthUser and KindOAuthClient
	ClientID string
	// APIKeyID is the key a KindAPIKey principal authenticated with
	APIKeyID string
	Scopes   ScopeSet

	// WithheldUntilVerified are the scopes the user's role grants but that are withheld until the user verifies their email
	WithheldUntilVerified ScopeSet

	// SessionID, AuthenticationMethod and AuthenticationTime identify the session and describe how it was established, and are only set for KindSession
	SessionID            string
	AuthenticationMethod string
	AuthenticationTime   time.Time

	// SudoUntil is when the session's sudo mode ends unless it is used before then, and is only set for KindSession
	SudoUntil time.Time
	// sudoUsed is shared by every copy of the principal, so the middleware learns after the handler whether the request relied on sudo mode
	sudoUsed *atomic.Bool
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

// Require returns the error a route returns to a principal lacking the scope, for checks that depend on the target of a request
func (p *Principal) Require(scope Scope) error {
	if p.Scopes.Has(scope) {
		return nil
	}
	return apperror.MissingScope(string(scope))
}

// RequireSudo returns the error that makes a browser session prove its passkey again, unless it is in sudo mode
// It guards what would turn a hijacked session into lasting access, which the session cookie alone must not be enough for
// Every other kind passes, because an API key or token can't prove a passkey and only holds the scopes it was deliberately given
func (p *Principal) RequireSudo() error {
	if p.Kind != KindSession {
		return nil
	}
	if !p.SudoUntil.After(time.Now()) {
		return apperror.SudoRequired()
	}

	// A principal that didn't pass through the middleware, such as one built by a test, has nowhere to record the use
	if p.sudoUsed != nil {
		p.sudoUsed.Store(true)
	}
	return nil
}

// RequireAdminAccess returns the error for a principal that may not decide who holds admin access, for checks that depend on the target of a request
func (p *Principal) RequireAdminAccess() error {
	err := p.Require(AdminAccessWrite)
	if err != nil {
		return err
	}
	return p.RequireSudo()
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
