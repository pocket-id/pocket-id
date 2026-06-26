package oidc

import (
	"context"
	"slices"

	"github.com/ory/fosite"
)

// standardScopes are the built-in identity scopes that any client may request
// They are released as ID-token and userinfo claims and are always available without targeting an API
var standardScopes = []string{"openid", "profile", "email", "groups", "offline_access"}

func isStandardScope(scope string) bool {
	return slices.Contains(standardScopes, scope)
}

// PermissionInfo is the display information for an API permission used to show friendly names and descriptions on the consent screen
type PermissionInfo struct {
	Key         string
	Name        string
	Description string
}

// APIAccessProvider is implemented by the api feature module
// It lets the OIDC module widen per-client scope and audience validation and resolve RFC 8707 resources to the permission keys a client may be granted
type APIAccessProvider interface {
	// ClientAPIScopes returns the custom-API permission keys and the distinct API audiences a client is allowed to request
	ClientAPIScopes(ctx context.Context, clientID string) (scopes []string, audiences []string, err error)
	// AllowedScopesForAudience returns the permission keys the client is allowed for the API identified by the given audience, and whether such an API exists
	AllowedScopesForAudience(ctx context.Context, clientID, audience string) (scopes []string, apiExists bool, err error)
	// DescribePermissions returns the display information for the given permission keys of the API identified by audience
	// Unknown keys are omitted
	DescribePermissions(ctx context.Context, audience string, keys []string) ([]PermissionInfo, error)
}

// resolveResource maps an RFC 8707 resource, which may be empty, to the audience to stamp on the issued token and the subset of requestedScopes that may be granted
// An empty resource is a plain login token bound to the requesting client and yields only identity scopes
func resolveResource(ctx context.Context, provider APIAccessProvider, clientID, resource string, requestedScopes []string) (audience string, grantedScopes []string, err error) {
	var grantable []string

	if resource == "" {
		// A plain login token is audienced to the requesting client and carries only identity scopes
		audience = clientID
		grantable = standardScopes
	} else {
		if provider == nil {
			return "", nil, fosite.ErrInvalidRequest.WithHintf("The resource %q is not a known API.", resource)
		}

		allowed, exists, allowErr := provider.AllowedScopesForAudience(ctx, clientID, resource)
		if allowErr != nil {
			return "", nil, allowErr
		}
		if !exists {
			return "", nil, fosite.ErrInvalidRequest.WithHintf("The resource %q is not a known API.", resource)
		}
		if len(allowed) == 0 {
			return "", nil, fosite.ErrAccessDenied.WithHintf("This client is not allowed to request the API %q.", resource)
		}

		audience = resource
		// Identity scopes stay grantable alongside a custom API so the ID token and its claims still work when openid or profile are requested
		// Custom scopes are limited to what the client is allowed for this specific API
		grantable = append(append([]string{}, standardScopes...), allowed...)
	}

	// Every requested scope must belong to the targeted API, though identity scopes are always allowed
	// A custom scope requested without its API, or for a different API than the one targeted, is rejected rather than silently dropped
	for _, scope := range requestedScopes {
		if !slices.Contains(grantable, scope) {
			return "", nil, fosite.ErrInvalidScope.WithHintf("The scope %q is not available for the requested resource.", scope)
		}
	}

	return audience, intersectScopes(requestedScopes, grantable), nil
}

func intersectScopes(requested, allowed []string) []string {
	result := make([]string, 0, len(requested))
	for _, scope := range requested {
		if slices.Contains(allowed, scope) && !slices.Contains(result, scope) {
			result = append(result, scope)
		}
	}
	return result
}

// consentScopeKey qualifies a custom-API scope by its audience so the same permission key on two different APIs is consented to separately
// The unit-separator delimiter is collision-free because audiences are validated as URIs and permission keys are restricted to RFC 6749 scope-token characters, so neither can contain it
// Standard identity scopes stay bare for backward compatibility with existing consents
func consentScopeKey(audience, scope string) string {
	if isStandardScope(scope) {
		return scope
	}
	return audience + "\x1f" + scope
}

func consentScopeKeys(audience string, scopes []string) []string {
	keys := make([]string, len(scopes))
	for i, scope := range scopes {
		keys[i] = consentScopeKey(audience, scope)
	}
	return keys
}
