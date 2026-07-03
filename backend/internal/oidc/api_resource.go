package oidc

import (
	"context"
	"slices"

	"github.com/ory/fosite"
	"gorm.io/gorm"
)

func isStandardScope(scope string) bool {
	// standardScopes are the built-in identity scopes that any client may request
	switch scope {
	case "openid", "profile", "email", "groups", "offline_access":
		return true
	default:
		return false
	}
}

// PermissionInfo is the display information for an API permission used to show friendly names and descriptions on the consent screen
type PermissionInfo struct {
	Key         string
	Name        string
	Description string
}

// SubjectType qualifies for whom an API grant applies, mirroring Auth0's client grant subject types
// A permission granted to a client for one subject type says nothing about the other, so user-delegated access and machine-to-machine access are configured independently
type SubjectType string

const (
	// SubjectTypeUser covers user-delegated access: every flow whose access token acts on behalf of an end user
	SubjectTypeUser SubjectType = "user"
	// SubjectTypeClient covers client access: the client credentials grant, where the client acts as itself without a user
	SubjectTypeClient SubjectType = "client"
)

// APIAccessProvider is implemented by the api feature module
// It lets the OIDC module widen per-client scope and audience validation and resolve RFC 8707 resources to the permission keys a client may be granted
type APIAccessProvider interface {
	// ClientAPIScopes returns the custom-API permission keys and the distinct API audiences a client is allowed to request across all subject types
	ClientAPIScopes(ctx context.Context, tx *gorm.DB, clientID string) (scopes []string, audiences []string, err error)
	// AllowedScopesForAudience returns the permission keys the client is allowed for the API identified by the given audience and subject type, and whether such an API exists
	AllowedScopesForAudience(ctx context.Context, clientID, audience string, subjectType SubjectType) (scopes []string, apiExists bool, err error)
	// DescribePermissions returns the display information for the given permission keys of the API identified by audience
	// Unknown keys are omitted
	DescribePermissions(ctx context.Context, audience string, keys []string) ([]PermissionInfo, error)
}

// resolveResource maps an RFC 8707 resource, which may be empty, to the audience to stamp on the issued token and the subset of requestedScopes that may be granted
// An empty resource is a plain login token bound to the requesting client and yields only identity scopes
// The subject type selects which of the client's grants apply: user-delegated flows only see user grants, the client credentials grant only sees client grants
func resolveResource(ctx context.Context, provider APIAccessProvider, clientID, resource string, requestedScopes []string, subjectType SubjectType) (audience string, grantedScopes []string, err error) {
	// Include the standard scopes by default
	// These are released as ID-token and userinfo claims and are always available without targeting an API
	grantable := map[string]struct{}{
		"openid":         {},
		"profile":        {},
		"email":          {},
		"groups":         {},
		"offline_access": {},
	}

	if resource == "" {
		// A plain login token is audienced to the requesting client and carries only identity scopes
		audience = clientID
	} else {
		if provider == nil {
			return "", nil, fosite.ErrInvalidRequest.WithHintf("The resource %q is not a known API.", resource)
		}

		allowed, exists, allowErr := provider.AllowedScopesForAudience(ctx, clientID, resource, subjectType)
		if allowErr != nil {
			return "", nil, allowErr
		}
		if !exists {
			return "", nil, fosite.ErrInvalidRequest.WithHintf("The resource %q is not a known API.", resource)
		}
		if len(allowed) == 0 {
			if subjectType == SubjectTypeClient {
				return "", nil, fosite.ErrAccessDenied.WithHintf("This client is not allowed machine-to-machine access to the API %q.", resource)
			}
			return "", nil, fosite.ErrAccessDenied.WithHintf("This client is not allowed to access the API %q on behalf of users.", resource)
		}

		audience = resource
		// Identity scopes stay grantable alongside a custom API so the ID token and its claims still work when openid or profile are requested
		// Custom scopes are limited to what the client is allowed for this specific API
		for _, a := range allowed {
			grantable[a] = struct{}{}
		}
	}

	// Every requested scope must belong to the targeted API, though identity scopes are always allowed
	// A custom scope requested without its API, or for a different API than the one targeted, is rejected rather than silently dropped
	granted := make([]string, 0, len(requestedScopes))
	for _, scope := range requestedScopes {
		_, ok := grantable[scope]
		if !ok {
			return "", nil, fosite.ErrInvalidScope.WithHintf("The scope %q is not available for the requested resource.", scope)
		}

		if !slices.Contains(granted, scope) {
			granted = append(granted, scope)
		}
	}

	return audience, granted, nil
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
