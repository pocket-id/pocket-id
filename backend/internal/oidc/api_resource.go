package oidc

import (
	"context"
	"slices"
	"strings"

	"github.com/ory/fosite"
	"github.com/pocket-id/pocket-id/backend/internal/dto"
	"gorm.io/gorm"
)

// SubjectType identifies the subject the requested access token will represent
type SubjectType string

const (
	// SubjectTypeUser covers user-delegated access: every flow whose access token acts on behalf of an end user
	SubjectTypeUser SubjectType = "user"
	// SubjectTypeClient covers client access: the client credentials grant, where the client acts as itself without a user
	SubjectTypeClient SubjectType = "client"
)

var standardScopes = fosite.Arguments{"openid", "profile", "email", "groups", "offline_access"}

func isStandardScope(scope string) bool {
	return slices.Contains(standardScopes, scope)
}

// APIAccessProvider is implemented by the api feature module
// It lets the OIDC module widen per-client scope and audience validation and resolve RFC 8707 resources to the permission keys a client may be granted
type APIAccessProvider interface {
	// ClientAPIScopes returns the custom-API permission keys and the distinct API audiences a client is allowed to request across all subject types
	ClientAPIScopes(ctx context.Context, tx *gorm.DB, clientID string, isCIMDClient bool) (scopes []string, audiences []string, err error)
	// AllowedScopesForAudience returns the permission keys the client is allowed for the API identified by the given audience and subject type, whether such an API exists, and whether the client may access it at all
	// A client can have access without any permission, in which case scopes is empty but hasAccess is true
	AllowedScopesForAudience(ctx context.Context, tx *gorm.DB, clientID, audience string, subjectType SubjectType) (scopes []string, apiExists bool, hasAccess bool, err error)
	// DescribePermissions returns the display information for the given permission keys of the API identified by audience
	// Unknown keys are omitted
	DescribePermissions(ctx context.Context, audience string, keys []string) ([]dto.ScopeInfoDto, error)
}

// resolveResource maps an RFC 8707 resource, which may be empty, to the audience to stamp on the issued token and the subset of requestedScopes that may be granted
// An empty resource is a plain login token bound to the requesting client and yields only identity scopes
// The subject type selects which of the client's grants apply: user-delegated flows only see user grants, the client credentials grant only sees client grants
func resolveResource(ctx context.Context, tx *gorm.DB, provider APIAccessProvider, clientID, resource string, requestedScopes []string, subjectType SubjectType) (audience string, grantedScopes []string, err error) {
	grantable := make(map[string]struct{}, len(standardScopes))
	for _, scope := range standardScopes {
		grantable[scope] = struct{}{}
	}

	if resource == "" {
		// A plain login token is audienced to the requesting client
		audience = clientID
	} else {
		if !fosite.IsValidResourceIndicatorURI(resource) || provider == nil {
			return "", nil, fosite.ErrInvalidTarget.WithHintf("The requested resource '%s' is invalid, missing, unknown, or malformed.", resource)
		}
		// Resolve every trailing-slash variant against the same canonical resource and stamp that value into the token audience
		resource = strings.TrimRight(resource, "/")

		allowedScopes, apiExists, hasAccess, err := provider.AllowedScopesForAudience(ctx, tx, clientID, resource, subjectType)
		if err != nil {
			return "", nil, err
		}
		if !apiExists {
			return "", nil, fosite.ErrInvalidTarget.WithHintf("The requested resource '%s' is invalid, missing, unknown, or malformed.", resource)
		}
		// Access is per API: a client allowed to reach one without any permission still gets a token, only without a scope
		if !hasAccess {
			return "", nil, fosite.ErrAccessDenied.WithHintf("The OAuth 2.0 Client is not allowed to access resource '%s'.", resource)
		}

		audience = resource
		for _, scope := range allowedScopes {
			grantable[scope] = struct{}{}
		}
		// A client credentials request without explicit scopes gets everything the client is granted for the API
		if subjectType == SubjectTypeClient && len(requestedScopes) == 0 {
			requestedScopes = allowedScopes
		}
	}

	granted := make(fosite.Arguments, 0, len(requestedScopes))
	for _, scope := range requestedScopes {
		if _, ok := grantable[scope]; !ok {
			return "", nil, fosite.ErrInvalidScope.WithHintf("The scope '%s' is not available for the requested resource.", scope)
		}
		if !granted.Has(scope) {
			granted = append(granted, scope)
		}
	}

	return audience, granted, nil
}

// grantResourceIndicator applies a resolved resource audience and scopes to the request
func grantResourceIndicator(requester fosite.Requester, audience string, scopes []string) {
	for _, scope := range scopes {
		requester.GrantScope(scope)
	}
	if audience != "" {
		requester.GrantAudience(audience)
	}
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

// consentAudienceKey records consent to reach the API itself, independent of any scope, so an audience-only request still prompts instead of reusing a plain login's consent
// The trailing unit separator with an empty scope cannot collide with a real key because permission keys are non-empty
func consentAudienceKey(audience string) string {
	return audience + "\x1f"
}

// consentKeysForGrant is the full set of keys a resolved grant has to be consented to: one per granted scope, plus the audience itself when a custom API was targeted
func consentKeysForGrant(audience, resource string, grantedScopes []string) []string {
	keys := consentScopeKeys(audience, grantedScopes)
	if resource != "" {
		keys = append(keys, consentAudienceKey(audience))
	}
	return keys
}
