package oidc

import (
	"slices"
	"time"

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

// GetHashedSecret returns the hash of the client's most recently created secret that is still active
// Fosite checks this one first and then falls back to GetRotatedHashes for the others
func (c Client) GetHashedSecret() []byte {
	active := c.Credentials.ActiveSecrets()
	if len(active) == 0 {
		return nil
	}
	return active[0].EncodedHash()
}

// GetRotatedHashes returns the hashes of every active secret except the one returned by GetHashedSecret
func (c Client) GetRotatedHashes() [][]byte {
	active := c.Credentials.ActiveSecrets()
	if len(active) < 2 {
		return nil
	}

	hashes := make([][]byte, len(active)-1)
	for i, secret := range active[1:] {
		hashes[i] = secret.EncodedHash()
	}
	return hashes
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

	if !c.IsMetadataDocument() {
		return grantTypes
	}
	if len(c.MetadataGrantTypes) == 0 {
		return fosite.Arguments{string(fosite.GrantTypeAuthorizationCode)}
	}

	// If the client is a CIMD client, we need to filter the grant types based on the metadata document.
	allowed := make(fosite.Arguments, 0, len(c.MetadataGrantTypes))
	for _, value := range c.MetadataGrantTypes {
		if slices.Contains([]string(grantTypes), value) {
			allowed = append(allowed, value)
		}
	}
	return allowed
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

func (c Client) GetEffectiveLifespan(grantType fosite.GrantType, tokenType fosite.TokenType, fallback time.Duration) time.Duration {
	var minutes int64
	switch tokenType {
	case fosite.AccessToken:
		switch grantType {
		case fosite.GrantTypeAuthorizationCode, fosite.GrantTypeRefreshToken, fosite.GrantTypeDeviceCode, fosite.GrantTypeClientCredentials:
			minutes = c.AccessTokenDurationMinutes
		case fosite.GrantTypeImplicit, fosite.GrantTypePassword, fosite.GrantTypeJWTBearer:
			return fallback
		default:
			return fallback
		}
	case fosite.RefreshToken:
		switch grantType {
		case fosite.GrantTypeAuthorizationCode, fosite.GrantTypeRefreshToken, fosite.GrantTypeDeviceCode:
			minutes = c.RefreshTokenDurationMinutes
		case fosite.GrantTypeImplicit, fosite.GrantTypePassword, fosite.GrantTypeClientCredentials, fosite.GrantTypeJWTBearer:
			return fallback
		default:
			return fallback
		}
	case fosite.AuthorizeCode, fosite.IDToken, fosite.UserCode, fosite.DeviceCode, fosite.PushedAuthorizeRequestContext:
		return fallback
	default:
		return fallback
	}

	if !model.IsValidTokenDurationMinutes(minutes) {
		return fallback
	}
	return time.Duration(minutes) * time.Minute
}
