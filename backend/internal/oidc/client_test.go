package oidc

import (
	"testing"
	"time"

	"github.com/ory/fosite"
	"github.com/stretchr/testify/require"

	"github.com/pocket-id/pocket-id/backend/internal/model"
)

// Interface assertions
var (
	_ fosite.Client                         = (*Client)(nil)
	_ fosite.ResponseModeClient             = (*Client)(nil)
	_ fosite.ClientWithCustomTokenLifespans = (*Client)(nil)
)

func TestClientGetEffectiveLifespan(t *testing.T) {
	client := Client{OidcClient: model.OidcClient{
		AccessTokenDurationMinutes:  2 * 60,
		RefreshTokenDurationMinutes: 7 * 24 * 60,
	}}
	fallback := 13 * time.Minute

	for _, test := range []struct {
		name      string
		grantType fosite.GrantType
		tokenType fosite.TokenType
		want      time.Duration
	}{
		{name: "authorization code access token", grantType: fosite.GrantTypeAuthorizationCode, tokenType: fosite.AccessToken, want: 2 * time.Hour},
		{name: "authorization code refresh token", grantType: fosite.GrantTypeAuthorizationCode, tokenType: fosite.RefreshToken, want: 7 * 24 * time.Hour},
		{name: "refresh grant access token", grantType: fosite.GrantTypeRefreshToken, tokenType: fosite.AccessToken, want: 2 * time.Hour},
		{name: "refresh grant refresh token", grantType: fosite.GrantTypeRefreshToken, tokenType: fosite.RefreshToken, want: 7 * 24 * time.Hour},
		{name: "device grant access token", grantType: fosite.GrantTypeDeviceCode, tokenType: fosite.AccessToken, want: 2 * time.Hour},
		{name: "device grant refresh token", grantType: fosite.GrantTypeDeviceCode, tokenType: fosite.RefreshToken, want: 7 * 24 * time.Hour},
		{name: "client credentials access token", grantType: fosite.GrantTypeClientCredentials, tokenType: fosite.AccessToken, want: 2 * time.Hour},
		{name: "client credentials refresh token falls back", grantType: fosite.GrantTypeClientCredentials, tokenType: fosite.RefreshToken, want: fallback},
		{name: "ID token falls back", grantType: fosite.GrantTypeAuthorizationCode, tokenType: fosite.IDToken, want: fallback},
		{name: "unsupported grant falls back", grantType: fosite.GrantTypePassword, tokenType: fosite.AccessToken, want: fallback},
	} {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.want, client.GetEffectiveLifespan(test.grantType, test.tokenType, fallback))
		})
	}

	client.AccessTokenDurationMinutes = 0
	client.RefreshTokenDurationMinutes = model.MaxTokenDurationMinutes + 1
	require.Equal(t, fallback, client.GetEffectiveLifespan(fosite.GrantTypeAuthorizationCode, fosite.AccessToken, fallback))
	require.Equal(t, fallback, client.GetEffectiveLifespan(fosite.GrantTypeAuthorizationCode, fosite.RefreshToken, fallback))
}
