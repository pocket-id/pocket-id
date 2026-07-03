package oidc

import (
	"context"
	"slices"

	"github.com/ory/fosite"
	fositeoauth2 "github.com/ory/fosite/handler/oauth2"
)

// isIdentityScope reports whether the scope is an OIDC identity scope whose presence lets a token be presented to Pocket ID's own identity endpoints such as /userinfo
// offline_access is deliberately excluded: it only requests a refresh token and is not tied to any resource server
func isIdentityScope(scope string) bool {
	return scope != "offline_access" && isStandardScope(scope)
}

// hasIdentityScope reports whether any of the granted scopes is an OIDC identity scope
func hasIdentityScope(scopes fosite.Arguments) bool {
	return slices.ContainsFunc(scopes, isIdentityScope)
}

// withIdentityAudience returns a view of the requester whose granted audience additionally includes the issuer when the token carries an identity scope
// This lets an access token that was granted an identity scope be presented to Pocket ID's own identity endpoints such as /userinfo, even when it was also audienced to a custom API
// The issuer is added only to the materialized access token and never to the underlying grant, so a refresh does not have to re-whitelist the issuer against the client and a machine token whose identity scopes were stripped never receives it
func withIdentityAudience(requester fosite.Requester, issuer string) fosite.Requester {
	if issuer == "" || !hasIdentityScope(requester.GetGrantedScopes()) {
		return requester
	}

	granted := requester.GetGrantedAudience()
	if granted.Has(issuer) {
		return requester
	}

	audience := make(fosite.Arguments, 0, len(granted)+1)
	audience = append(audience, granted...)
	audience = append(audience, issuer)
	return identityAudienceRequester{Requester: requester, grantedAudience: audience}
}

// identityAudienceRequester overrides only the granted audience of the wrapped requester
type identityAudienceRequester struct {
	fosite.Requester
	grantedAudience fosite.Arguments
}

func (r identityAudienceRequester) GetGrantedAudience() fosite.Arguments {
	return r.grantedAudience
}

// identityAudienceAccessTokenStrategy wraps the access token strategy so an access token granted an identity scope also lists the issuer in its audience, keeping the self-contained JWT consistent with what is persisted for introspection and userinfo
type identityAudienceAccessTokenStrategy struct {
	fositeoauth2.CoreStrategy
	issuer string
}

func (s identityAudienceAccessTokenStrategy) GenerateAccessToken(ctx context.Context, requester fosite.Requester) (string, string, error) {
	return s.CoreStrategy.GenerateAccessToken(ctx, withIdentityAudience(requester, s.issuer))
}
