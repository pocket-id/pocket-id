package authz

import (
	"github.com/getkin/kin-openapi/openapi3"
)

// APIKeyHeader is the header that carries a personal API key or the static API key
// #nosec G101 -- this is the name of the header, not a credential
const APIKeyHeader = "X-API-Key"

// tokenURL is the token endpoint relative to the server, since the spec doesn't know the URL of an instance
const tokenURL = "/api/oidc/token" // #nosec G101 -- the URL of the token endpoint, not a credential

// securitySchemes lists the scheme that documents each kind of credential, in the order the API reference offers them
var securitySchemes = []struct {
	name  string
	kinds PrincipalKind
}{
	{"apiKey", KindAPIKey | KindStaticAPIKey},
	{"oauth2", KindOAuthUser},
	{"oauth2ClientCredentials", KindOAuthClient},
	{"session", KindSession},
}

// SecuritySchemes returns the OpenAPI security schemes of every kind of credential, where the OAuth flows list the scopes a token of their kind can hold
func SecuritySchemes() openapi3.SecuritySchemes {
	scopes := func(kind PrincipalKind) map[string]string {
		names := map[string]string{}
		for _, entry := range catalog {
			if entry.grantableTo&kind != 0 {
				names[string(entry.scope)] = ""
			}
		}
		return names
	}

	return openapi3.SecuritySchemes{
		"apiKey": {Value: &openapi3.SecurityScheme{
			Type:        "apiKey",
			In:          "header",
			Name:        APIKeyHeader,
			Description: "A personal API key, which holds the scopes chosen when it was created, or the static API key from `STATIC_API_KEY`.",
		}},
		"oauth2": {Value: &openapi3.SecurityScheme{
			Type:        "oauth2",
			Description: "An access token of an OAuth client acting on behalf of a user, requested with the `resource` parameter set to `<APP_URL>/api`.",
			Flows: &openapi3.OAuthFlows{AuthorizationCode: &openapi3.OAuthFlow{
				AuthorizationURL: "/authorize",
				TokenURL:         tokenURL,
				Scopes:           scopes(KindOAuthUser),
			}},
		}},
		"oauth2ClientCredentials": {Value: &openapi3.SecurityScheme{
			Type:        "oauth2",
			Description: "An access token of an OAuth client acting as itself, requested with the `resource` parameter set to `<APP_URL>/api`.",
			Flows: &openapi3.OAuthFlows{ClientCredentials: &openapi3.OAuthFlow{
				TokenURL: tokenURL,
				Scopes:   scopes(KindOAuthClient),
			}},
		}},
		"session": {Value: &openapi3.SecurityScheme{
			Type:        "apiKey",
			In:          "cookie",
			Name:        "__Host-session_token",
			Description: "The session of a user signed in to Pocket ID in the browser, whose cookie is named `session_token` when `APP_URL` uses plain HTTP.",
		}},
	}
}

// security returns the alternative credentials that may call a route, each with the scope it must hold
func security(access Access, optional bool) openapi3.SecurityRequirements {
	if access == Public {
		return nil
	}

	// Authenticated routes accept every kind of credential without a scope
	scope, scoped := access.(Scope)
	var requirements openapi3.SecurityRequirements
	for _, scheme := range securitySchemes {
		switch {
		case !scoped:
			requirements = append(requirements, openapi3.SecurityRequirement{scheme.name: {}})
		case definitions[scope].grantableTo&scheme.kinds != 0:
			requirements = append(requirements, openapi3.SecurityRequirement{scheme.name: {string(scope)}})
		}
	}

	// An empty requirement tells the reader that anonymous requests are allowed too
	if optional {
		requirements = append(requirements, openapi3.SecurityRequirement{})
	}
	return requirements
}
