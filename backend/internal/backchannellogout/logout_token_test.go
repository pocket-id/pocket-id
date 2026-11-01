package backchannellogout

import (
	"testing"

	"github.com/lestrrat-go/jwx/v4/jwk"
	"github.com/lestrrat-go/jwx/v4/jws"
	"github.com/lestrrat-go/jwx/v4/jwt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/instanceid"
	"github.com/pocket-id/pocket-id/backend/internal/signingkey"
	testutils "github.com/pocket-id/pocket-id/backend/internal/utils/testing"
)

func TestGenerateLogoutToken(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	instanceID, err := instanceid.Load(t.Context(), db)
	require.NoError(t, err)
	key, err := signingkey.Load(t.Context(), db, &common.EnvConfigSchema{EncryptionKey: []byte("0123456789abcdef0123456789abcdef")}, instanceID)
	require.NoError(t, err)

	signed, err := NewTokenSigner(key, "https://test.example.com").GenerateLogoutToken("user-id", "client-id")
	require.NoError(t, err)

	// The spec requires the typ header, and clients pick the verification key from the JWKS by the key ID
	message, err := jws.Parse([]byte(signed))
	require.NoError(t, err)
	require.Len(t, message.Signatures(), 1)
	headers := message.Signatures()[0].ProtectedHeaders()
	typ, _ := headers.Type()
	assert.Equal(t, logoutTokenType, typ)
	keyID, _ := headers.KeyID()
	expectedKeyID, _ := key.GetKeyID()
	assert.Equal(t, expectedKeyID, keyID)

	jwks, err := jwk.Parse(key.PublicJWKS())
	require.NoError(t, err)
	token, err := jwt.ParseString(signed, jwt.WithValidate(true), jwt.WithKeySet(jwks))
	require.NoError(t, err)

	subject, _ := token.Subject()
	assert.Equal(t, "user-id", subject)
	audience, _ := token.Audience()
	assert.Equal(t, []string{"client-id"}, audience)
	issuer, _ := token.Issuer()
	assert.Equal(t, "https://test.example.com", issuer)
	jti, _ := token.JwtID()
	assert.NotEmpty(t, jti)

	events, err := jwt.Get[map[string]any](token, "events")
	require.NoError(t, err)
	assert.Contains(t, events, logoutEvent)

	// The spec forbids a nonce claim in logout tokens
	assert.False(t, token.Has("nonce"))
}
