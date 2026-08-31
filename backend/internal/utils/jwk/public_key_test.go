package jwk

import (
	"encoding/json"
	"testing"

	"github.com/lestrrat-go/jwx/v4/jwa"
	"github.com/lestrrat-go/jwx/v4/jwk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// generateTestPublicKey returns a public JWK, encoded as it would be pasted in the admin UI
func generateTestPublicKey(t *testing.T, alg string) (jwk.Key, json.RawMessage) {
	t.Helper()

	privateKey, err := GenerateKey(alg, "")
	require.NoError(t, err)
	publicKey, err := privateKey.PublicKey()
	require.NoError(t, err)

	encoded, err := json.Marshal(publicKey)
	require.NoError(t, err)

	return publicKey, encoded
}

func TestParsePublicKey(t *testing.T) {
	t.Run("parses a public key", func(t *testing.T) {
		publicKey, encoded := generateTestPublicKey(t, jwa.RS256().String())

		parsed, err := ParsePublicKey(encoded)
		require.NoError(t, err)

		expectedKid, _ := publicKey.KeyID()
		parsedKid, ok := parsed.KeyID()
		require.True(t, ok)
		assert.Equal(t, expectedKid, parsedKid)
	})

	t.Run("rejects invalid JSON", func(t *testing.T) {
		_, err := ParsePublicKey([]byte("not-a-jwk"))
		require.Error(t, err)
	})

	t.Run("rejects keys with private key material", func(t *testing.T) {
		privateKey, err := GenerateKey(jwa.ES256().String(), "")
		require.NoError(t, err)
		encoded, err := json.Marshal(privateKey)
		require.NoError(t, err)

		_, err = ParsePublicKey(encoded)
		require.ErrorIs(t, err, ErrPublicKeyIsPrivate)
	})

	t.Run("rejects symmetric keys", func(t *testing.T) {
		symmetricKey, err := jwk.Import[jwk.Key]([]byte("this-is-a-shared-secret"))
		require.NoError(t, err)
		require.NoError(t, symmetricKey.Set(jwk.KeyIDKey, "symmetric"))
		encoded, err := json.Marshal(symmetricKey)
		require.NoError(t, err)

		_, err = ParsePublicKey(encoded)
		require.ErrorIs(t, err, ErrPublicKeyNotAsymmetric)
	})

	t.Run("rejects keys without a key ID", func(t *testing.T) {
		_, encoded := generateTestPublicKey(t, jwa.RS256().String())

		var key map[string]any
		require.NoError(t, json.Unmarshal(encoded, &key))
		delete(key, "kid")
		withoutKid, err := json.Marshal(key)
		require.NoError(t, err)

		_, err = ParsePublicKey(withoutKid)
		require.ErrorIs(t, err, ErrPublicKeyMissingKeyID)
	})
}

func TestParsePublicKeySet(t *testing.T) {
	t.Run("parses multiple keys", func(t *testing.T) {
		first, encodedFirst := generateTestPublicKey(t, jwa.RS256().String())
		second, encodedSecond := generateTestPublicKey(t, jwa.ES256().String())

		set, err := ParsePublicKeySet([]json.RawMessage{encodedFirst, encodedSecond})
		require.NoError(t, err)
		require.Equal(t, 2, set.Len())

		for _, key := range []jwk.Key{first, second} {
			kid, _ := key.KeyID()
			_, found := set.LookupKeyID(kid)
			assert.True(t, found, "key %s is missing from the set", kid)
		}
	})

	t.Run("returns an empty set for no keys", func(t *testing.T) {
		set, err := ParsePublicKeySet(nil)
		require.NoError(t, err)
		assert.Equal(t, 0, set.Len())
	})

	t.Run("rejects duplicate key IDs", func(t *testing.T) {
		_, encodedFirst := generateTestPublicKey(t, jwa.RS256().String())
		_, encodedSecond := generateTestPublicKey(t, jwa.RS256().String())

		var second map[string]any
		err := json.Unmarshal(encodedSecond, &second)
		require.NoError(t, err)
		var first map[string]any
		err = json.Unmarshal(encodedFirst, &first)
		require.NoError(t, err)
		second["kid"] = first["kid"]
		duplicate, err := json.Marshal(second)
		require.NoError(t, err)

		_, err = ParsePublicKeySet([]json.RawMessage{encodedFirst, duplicate})
		require.ErrorContains(t, err, "same key ID")
	})

	t.Run("reports the position of the invalid key", func(t *testing.T) {
		_, encoded := generateTestPublicKey(t, jwa.RS256().String())

		_, err := ParsePublicKeySet([]json.RawMessage{encoded, []byte(`{"kty":"RSA"}`)})
		require.ErrorContains(t, err, "key 2 is invalid")
	})
}

func TestNormalizePublicKeys(t *testing.T) {
	t.Run("re-encodes keys", func(t *testing.T) {
		publicKey, encoded := generateTestPublicKey(t, jwa.RS256().String())

		normalized, err := NormalizePublicKeys([]json.RawMessage{json.RawMessage("  " + string(encoded) + "\n")})
		require.NoError(t, err)
		require.Len(t, normalized, 1)

		parsed, err := jwk.ParseKey(normalized[0])
		require.NoError(t, err)
		expectedKid, _ := publicKey.KeyID()
		parsedKid, _ := parsed.KeyID()
		assert.Equal(t, expectedKid, parsedKid)
	})

	t.Run("returns nil for no keys", func(t *testing.T) {
		normalized, err := NormalizePublicKeys(nil)
		require.NoError(t, err)
		assert.Nil(t, normalized)
	})

	t.Run("returns an error for an invalid key", func(t *testing.T) {
		_, err := NormalizePublicKeys([]json.RawMessage{[]byte(`{"kty":"oct","k":"c2VjcmV0","kid":"shared"}`)})
		require.ErrorIs(t, err, ErrPublicKeyNotAsymmetric)
	})
}
