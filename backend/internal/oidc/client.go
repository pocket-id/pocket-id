package oidc

import (
	"github.com/ory/fosite"
	"github.com/pocket-id/pocket-id/backend/internal/model"
)

type Client struct {
	model.OidcClient

	apiScopes    []string
	apiAudiences []string
}

func (c Client) GetID() string {
	return c.ID
}

func (c Client) GetHashedSecret() []byte {
	return []byte(c.Secret)
}

func (c Client) GetRedirectURIs() []string {
	return c.CallbackURLs
}

func (c Client) GetGrantTypes() fosite.Arguments {
	grantTypes := fosite.Arguments{
		string(fosite.GrantTypeAuthorizationCode),
		string(fosite.GrantTypeRefreshToken),
		string(fosite.GrantTypeDeviceCode),
	}
	if !c.IsPublic() {
		grantTypes = append(grantTypes, string(fosite.GrantTypeClientCredentials))
	}
	return grantTypes
}

func (c Client) GetResponseTypes() fosite.Arguments {
	return fosite.Arguments{"code"}
}

func (c Client) GetScopes() fosite.Arguments {
	scopes := make(fosite.Arguments, 5, 5+len(c.apiScopes))
	scopes[0] = "openid"
	scopes[1] = "profile"
	scopes[2] = "email"
	scopes[3] = "groups"
	scopes[4] = "offline_access"
	scopes = append(scopes, c.apiScopes...)
	return scopes
}

func (c Client) IsPublic() bool {
	return c.OidcClient.IsPublic
}

func (c Client) GetAudience() fosite.Arguments {
	audience := make(fosite.Arguments, 0, len(c.apiAudiences)+1)
	audience = append(audience, c.ID)
	audience = append(audience, c.apiAudiences...)
	return audience
}

func (c Client) GetResponseModes() []fosite.ResponseModeType {
	return []fosite.ResponseModeType{
		fosite.ResponseModeQuery,
		fosite.ResponseModeFragment,
		fosite.ResponseModeFormPost,
	}
}
