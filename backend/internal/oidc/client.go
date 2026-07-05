package oidc

import (
	"github.com/ory/fosite"
	"github.com/pocket-id/pocket-id/backend/internal/model"
)

var _ fosite.Client = (*Client)(nil)
var _ fosite.ResponseModeClient = (*Client)(nil)

type Client struct {
	model.OidcClient
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
	return fosite.Arguments{"openid", "profile", "email", "groups", "offline_access"}
}

func (c Client) IsPublic() bool {
	return c.OidcClient.IsPublic
}

func (c Client) GetAudience() fosite.Arguments {
	return fosite.Arguments{c.ID}
}

func (c Client) GetResponseModes() []fosite.ResponseModeType {
	return []fosite.ResponseModeType{
		fosite.ResponseModeQuery,
		fosite.ResponseModeFragment,
		fosite.ResponseModeFormPost,
	}
}
