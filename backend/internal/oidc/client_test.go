package oidc

import (
	"strconv"
	"testing"
	"time"

	"github.com/ory/fosite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pocket-id/pocket-id/backend/internal/model"
	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
)

// Interface assertions
var (
	_ fosite.Client                         = (*Client)(nil)
	_ fosite.ResponseModeClient             = (*Client)(nil)
	_ fosite.ClientWithCustomTokenLifespans = (*Client)(nil)
	_ fosite.ClientWithSecretRotation       = (*Client)(nil)
)

// testClientCredentials returns credentials with a single never-expiring secret with the given value
func testClientCredentials(values ...string) model.OidcClientCredentials {
	credentials := model.OidcClientCredentials{
		Secrets: make([]model.OidcClientSecret, len(values)),
	}
	for i, value := range values {
		credentials.Secrets[i] = model.OidcClientSecret{
			ID:        "secret-" + strconv.Itoa(i),
			Algorithm: model.OidcClientSecretHashSHA256,
			Hash:      utils.CreateSha256Hash(value),
			Prefix:    value[:model.OidcClientSecretPrefixLength],
			CreatedAt: datatype.DateTime(time.Now()),
		}
	}
	return credentials
}

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

func TestClientSecretHashes(t *testing.T) {
	past := datatype.DateTime(time.Now().Add(-time.Hour))
	future := datatype.DateTime(time.Now().Add(time.Hour))
	older := datatype.DateTime(time.Now().Add(-48 * time.Hour))
	newer := datatype.DateTime(time.Now().Add(-time.Minute))

	t.Run("client without secrets", func(t *testing.T) {
		client := Client{}
		assert.Nil(t, client.GetHashedSecret())
		assert.Empty(t, client.GetRotatedHashes())
	})

	t.Run("expired secrets are never returned", func(t *testing.T) {
		client := Client{OidcClient: model.OidcClient{Credentials: model.OidcClientCredentials{
			Secrets: []model.OidcClientSecret{
				{ID: "expired", Algorithm: model.OidcClientSecretHashSHA256, Hash: "expired-hash", ExpiresAt: &past},
			},
		}}}
		assert.Nil(t, client.GetHashedSecret())
		assert.Empty(t, client.GetRotatedHashes())
	})

	t.Run("the most recent active secret comes first", func(t *testing.T) {
		client := Client{OidcClient: model.OidcClient{Credentials: model.OidcClientCredentials{
			Secrets: []model.OidcClientSecret{
				{ID: "older", Algorithm: model.OidcClientSecretHashSHA256, Hash: "older-hash", CreatedAt: older},
				{ID: "expired", Algorithm: model.OidcClientSecretHashSHA256, Hash: "expired-hash", CreatedAt: newer, ExpiresAt: &past},
				{ID: "newer", Algorithm: model.OidcClientSecretHashSHA256, Hash: "newer-hash", CreatedAt: newer, ExpiresAt: &future},
			},
		}}}

		assert.Equal(t, "sha256:newer-hash", string(client.GetHashedSecret()))
		rotated := client.GetRotatedHashes()
		require.Len(t, rotated, 1)
		assert.Equal(t, "sha256:older-hash", string(rotated[0]))
	})
}
