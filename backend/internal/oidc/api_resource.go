package oidc

import (
	"context"
	"slices"

	"github.com/ory/fosite"
	"gorm.io/gorm"
)

type SubjectType = fosite.ResourceIndicatorSubjectType

const (
	// SubjectTypeUser covers user-delegated access: every flow whose access token acts on behalf of an end user
	SubjectTypeUser = fosite.ResourceOwnerSubject
	// SubjectTypeClient covers client access: the client credentials grant, where the client acts as itself without a user
	SubjectTypeClient = fosite.ClientSubject
)

var standardScopes = fosite.Arguments{"openid", "profile", "email", "groups", "offline_access"}

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
	// ClientAPIScopes returns the custom-API permission keys and the distinct API audiences a client is allowed to request across all subject types
	ClientAPIScopes(ctx context.Context, tx *gorm.DB, clientID string) (scopes []string, audiences []string, err error)
	// AllowedScopesForAudience returns the permission keys the client is allowed for the API identified by the given audience and subject type, and whether such an API exists
	AllowedScopesForAudience(ctx context.Context, clientID, audience string, subjectType SubjectType) (scopes []string, apiExists bool, err error)
	// DescribePermissions returns the display information for the given permission keys of the API identified by audience
	// Unknown keys are omitted
	DescribePermissions(ctx context.Context, audience string, keys []string) ([]PermissionInfo, error)
}

type resourceAccessProvider struct {
	provider APIAccessProvider
}

func fositeResourceAccess(provider APIAccessProvider) fosite.ResourceIndicatorAccessProvider {
	if provider == nil {
		return nil
	}
	return resourceAccessProvider{provider: provider}
}

func (p resourceAccessProvider) AllowedScopesForResource(ctx context.Context, client fosite.Client, resource string, subjectType fosite.ResourceIndicatorSubjectType) (fosite.Arguments, bool, error) {
	scopes, exists, err := p.provider.AllowedScopesForAudience(ctx, client.GetID(), resource, subjectType)
	return scopes, exists, err
}

// resolveResource maps an RFC 8707 resource, which may be empty, to the audience to stamp on the issued token and the subset of requestedScopes that may be granted
// An empty resource is a plain login token bound to the requesting client and yields only identity scopes
// The subject type selects which of the client's grants apply: user-delegated flows only see user grants, the client credentials grant only sees client grants
func resolveResource(ctx context.Context, provider APIAccessProvider, clientID, resource string, requestedScopes []string, subjectType SubjectType) (audience string, grantedScopes []string, err error) {
	grant, err := fosite.ResolveResourceIndicatorGrant(ctx, fositeResourceAccess(provider), &fosite.DefaultClient{ID: clientID}, resource, requestedScopes, standardScopes, subjectType)
	if err != nil {
		return "", nil, err
	}

	return grant.Audience, grant.Scopes, nil
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
