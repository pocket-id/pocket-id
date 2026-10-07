package auditlogs

import (
	"testing"
	"time"

	"github.com/lestrrat-go/jwx/v4/jwk"
	"github.com/lestrrat-go/jwx/v4/jwt"
	"github.com/stretchr/testify/require"

	jwkutils "github.com/pocket-id/pocket-id/backend/internal/utils/jwk"
)

type browserTokenTestSigner struct{ key jwk.Key }

func (s browserTokenTestSigner) SignSessionToken(token jwt.Token) (string, error) {
	signed, err := jwt.Sign(token, jwt.WithKey(jwkutils.SessionKeyAlg(), s.key))
	return string(signed), err
}

func (s browserTokenTestSigner) VerifySessionToken(token string, options ...jwt.ValidateOption) (jwt.Token, error) {
	parseOptions := []jwt.ParseOption{jwt.WithKey(jwkutils.SessionKeyAlg(), s.key), jwt.WithValidate(true)}
	for _, option := range options {
		parseOptions = append(parseOptions, option)
	}
	return jwt.ParseString(token, parseOptions...)
}

func newBrowserTokenService(t *testing.T) *browserTokens {
	t.Helper()
	key, err := jwkutils.GenerateSessionKey()
	require.NoError(t, err)
	return &browserTokens{signer: browserTokenTestSigner{key: key}, appURL: "https://test.example.com"}
}

func TestKnownBrowserToken(t *testing.T) {
	s := newBrowserTokenService(t)
	const lifetime = 180 * 24 * time.Hour
	encoded, err := s.generate("user", lifetime)
	require.NoError(t, err)
	require.NoError(t, s.verify(encoded, "user"))
	require.Error(t, s.verify(encoded, "other-user"))
	require.Error(t, s.verify(encoded, ""))

	token, err := jwt.ParseString(encoded, jwt.WithKey(jwkutils.SessionKeyAlg(), s.signer.(browserTokenTestSigner).key))
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
			encoded, err := jwt.Sign(token, jwt.WithKey(jwkutils.SessionKeyAlg(), s.signer.(browserTokenTestSigner).key))
			require.NoError(t, err)
			require.Error(t, s.verify(string(encoded), "user"))
		})
	}
}
