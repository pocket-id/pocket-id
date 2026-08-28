package dto

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/gin-gonic/gin/binding"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pocket-id/pocket-id/backend/internal/model"
	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
)

func TestOidcClientUpdateDto_tokenLifetimes(t *testing.T) {
	const baseFields = `"name":"Test Client","callbackURLs":["https://example.com/callback"]`

	for _, test := range []struct {
		name        string
		body        string
		wantErr     bool
		wantAccess  int64
		wantRefresh int64
	}{
		{
			name: "lifetimes omitted",
			body: `{` + baseFields + `}`,
		},
		{
			name: "lifetimes null",
			body: `{` + baseFields + `,"accessTokenDurationMinutes":null,"refreshTokenDurationMinutes":null}`,
		},
		{
			name: "lifetimes zero",
			body: `{` + baseFields + `,"accessTokenDurationMinutes":0,"refreshTokenDurationMinutes":0}`,
		},
		{
			name:       "only the access token lifetime set",
			body:       `{` + baseFields + `,"accessTokenDurationMinutes":120}`,
			wantAccess: 120,
		},
		{
			name:        "both lifetimes set",
			body:        `{` + baseFields + `,"accessTokenDurationMinutes":120,"refreshTokenDurationMinutes":10080}`,
			wantAccess:  120,
			wantRefresh: 10080,
		},
		{
			name:    "negative value is rejected",
			body:    `{` + baseFields + `,"accessTokenDurationMinutes":-1}`,
			wantErr: true,
		},
		{
			name:    "value above the maximum is rejected",
			body:    `{` + baseFields + `,"refreshTokenDurationMinutes":525601}`,
			wantErr: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			var input OidcClientUpdateDto
			require.NoError(t, json.Unmarshal([]byte(test.body), &input))

			err := binding.Validator.ValidateStruct(input)
			if test.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			// A zero value is what the service turns into the default lifetime
			assert.Equal(t, test.wantAccess, input.AccessTokenDurationMinutes)
			assert.Equal(t, test.wantRefresh, input.RefreshTokenDurationMinutes)
		})
	}
}

func TestOidcClientDto_secrets(t *testing.T) {
	expired := datatype.DateTime(time.Now().Add(-time.Hour))
	valid := datatype.DateTime(time.Now().Add(time.Hour))
	client := model.OidcClient{
		ID:   "client-id",
		Name: "Test Client",
		Credentials: model.OidcClientCredentials{
			Secrets: []model.OidcClientSecret{
				{ID: "active", Algorithm: model.OidcClientSecretHashSHA256, Hash: "hash-1", Prefix: "abcd"},
				{ID: "expiring", Algorithm: model.OidcClientSecretHashSHA256, Hash: "hash-2", Prefix: "efgh", ExpiresAt: &valid},
				{ID: "expired", Algorithm: model.OidcClientSecretHashSHA256, Hash: "hash-3", Prefix: "ijkl", ExpiresAt: &expired},
			},
		},
	}

	var clientDto OidcClientDto
	require.NoError(t, MapStruct(client, &clientDto))
	require.Len(t, clientDto.Credentials.Secrets, 3)

	assert.Equal(t, "active", clientDto.Credentials.Secrets[0].ID)
	assert.Equal(t, "abcd", clientDto.Credentials.Secrets[0].Prefix)
	assert.True(t, clientDto.Credentials.Secrets[0].IsActive)
	assert.True(t, clientDto.Credentials.Secrets[1].IsActive)
	assert.False(t, clientDto.Credentials.Secrets[2].IsActive)

	// Serializing the client must never disclose the hashes of its secrets
	serialized, err := json.Marshal(clientDto)
	require.NoError(t, err)
	assert.NotContains(t, string(serialized), "hash-1")
	assert.Contains(t, string(serialized), `"isActive":true`)
}

func TestOidcClientDto_secretsAlwaysSerialized(t *testing.T) {
	client := model.OidcClient{
		ID:   "client-id",
		Name: "Test Client",
	}

	var clientDto OidcClientDto
	require.NoError(t, MapStruct(client, &clientDto))
	assert.Empty(t, clientDto.Credentials.Secrets)

	// A client without secrets must serialize an empty list rather than omitting the field, so consumers never have to handle a missing value
	serialized, err := json.Marshal(clientDto)
	require.NoError(t, err)
	assert.Contains(t, string(serialized), `"secrets":[]`)
}
