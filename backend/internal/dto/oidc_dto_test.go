package dto

import (
	"encoding/json"
	"testing"

	"github.com/gin-gonic/gin/binding"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
