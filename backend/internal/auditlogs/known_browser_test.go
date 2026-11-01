package auditlogs

import (
	"crypto/rand"
	"testing"
	"time"

	"github.com/lestrrat-go/jwx/v4/jwa"
	"github.com/lestrrat-go/jwx/v4/jwt"
	"github.com/stretchr/testify/require"

	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/instanceid"
	jwkutils "github.com/pocket-id/pocket-id/backend/internal/utils/jwk"
	testutils "github.com/pocket-id/pocket-id/backend/internal/utils/testing"
)

func newBrowserTokenService(t *testing.T) *browserTokens {
	t.Helper()
	key, err := jwkutils.GenerateSessionKey()
	require.NoError(t, err)
	return &browserTokens{key: key, appURL: "https://test.example.com"}
}

func TestKnownBrowserToken(t *testing.T) {
	s := newBrowserTokenService(t)
	const lifetime = 180 * 24 * time.Hour
	encoded, err := s.generate("user", lifetime)
	require.NoError(t, err)
	require.NoError(t, s.verify(encoded, "user"))
	require.Error(t, s.verify(encoded, "other-user"))
	require.Error(t, s.verify(encoded, ""))

	token, err := jwt.ParseString(encoded, jwt.WithKey(jwkutils.SessionKeyAlg(), s.key))
	require.NoError(t, err)
	issued, ok := token.IssuedAt()
	require.True(t, ok)
	expires, ok := token.Expiration()
	require.True(t, ok)
	require.Equal(t, lifetime, expires.Sub(issued))
}

func TestKnownBrowserTokenRejectsInvalidClaims(t *testing.T) {
	s := newBrowserTokenService(t)
	for _, tt := range []struct {
		name  string
		claim string
		value any
	}{
		{"expired", jwt.ExpirationKey, time.Now().Add(-time.Second)},
		{"missing expiry", jwt.ExpirationKey, nil},
		{"missing issued at", jwt.IssuedAtKey, nil},
		{"future issued at", jwt.IssuedAtKey, time.Now().Add(time.Hour)},
		{"wrong issuer", jwt.IssuerKey, "https://other.example.com"},
		{"missing issuer", jwt.IssuerKey, nil},
		{"wrong audience", jwt.AudienceKey, []string{"other"}},
		{"missing audience", jwt.AudienceKey, nil},
		{"missing user", jwt.SubjectKey, nil},
		{"wrong purpose", "type", "access-token"},
		{"missing purpose", "type", nil},
	} {
		t.Run(tt.name, func(t *testing.T) {
			// Sign invalid claims with the real key so validation, rather than signature failure, rejects them
			token, err := jwt.NewBuilder().Subject("user").Issuer(s.appURL).
				Audience([]string{s.appURL}).IssuedAt(time.Now()).Expiration(time.Now().Add(time.Hour)).
				Claim("type", knownBrowserJWTType).Build()
			require.NoError(t, err)
			if tt.value == nil {
				require.NoError(t, token.Remove(tt.claim))
			} else {
				require.NoError(t, token.Set(tt.claim, tt.value))
			}
			encoded, err := jwt.Sign(token, jwt.WithKey(jwkutils.SessionKeyAlg(), s.key))
			require.NoError(t, err)
			require.Error(t, s.verify(string(encoded), "user"))
		})
	}
}

func TestKnownBrowserKey(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	instanceID, err := instanceid.Load(t.Context(), db)
	require.NoError(t, err)
	envConfig := &common.EnvConfigSchema{EncryptionKey: []byte("0123456789abcdef0123456789abcdef")}

	t.Run("a token stays valid after the stored key is loaded again", func(t *testing.T) {
		first := &browserTokens{db: db, envConfig: envConfig, appURL: "https://test.example.com"}
		require.NoError(t, first.loadKey(t.Context(), instanceID))
		encoded, err := first.generate("user", time.Hour)
		require.NoError(t, err)

		second := &browserTokens{db: db, envConfig: envConfig, appURL: "https://test.example.com"}
		require.NoError(t, second.loadKey(t.Context(), instanceID))
		require.NoError(t, second.verify(encoded, "user"))
	})

	t.Run("rejects keys for another algorithm", func(t *testing.T) {
		asymmetric, err := jwkutils.GenerateKey(jwa.ES256().String(), "")
		require.NoError(t, err)
		require.ErrorContains(t, validateBrowserKey(asymmetric), "not a symmetric key")

		rawKey := make([]byte, 32)
		_, err = rand.Read(rawKey)
		require.NoError(t, err)
		otherAlg, err := jwkutils.ImportRawKey(rawKey, jwa.HS512().String(), "")
		require.NoError(t, err)
		require.ErrorContains(t, validateBrowserKey(otherAlg), "not valid for the HS256 algorithm")
	})
}
