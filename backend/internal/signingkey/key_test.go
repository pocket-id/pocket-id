package signingkey

import (
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"testing"

	"github.com/lestrrat-go/jwx/v4/jwa"
	"github.com/lestrrat-go/jwx/v4/jwk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/instanceid"
	jwkutils "github.com/pocket-id/pocket-id/backend/internal/utils/jwk"
	testutils "github.com/pocket-id/pocket-id/backend/internal/utils/testing"
)

func newTestEnv(t *testing.T) (*gorm.DB, *common.EnvConfigSchema, string) {
	t.Helper()

	db := testutils.NewDatabaseForTest(t)
	instanceID, err := instanceid.Load(t.Context(), db)
	require.NoError(t, err)
	envConfig := &common.EnvConfigSchema{
		AppURL:        "https://test.example.com",
		EncryptionKey: []byte("0123456789abcdef0123456789abcdef"),
	}
	return db, envConfig, instanceID
}

func storeKey(t *testing.T, db *gorm.DB, envConfig *common.EnvConfigSchema, instanceID string, rawKey any) string {
	t.Helper()

	key, err := jwkutils.ImportRawKey(rawKey, "", "")
	require.NoError(t, err)
	provider, err := jwkutils.GetKeyProvider(db, envConfig, instanceID)
	require.NoError(t, err)
	require.NoError(t, provider.ReplaceKey(t.Context(), key))

	keyID, ok := key.KeyID()
	require.True(t, ok)
	return keyID
}

func publicKeyFromJWKS(t *testing.T, k *Key) jwk.Key {
	t.Helper()

	set, err := jwk.Parse(k.PublicJWKS())
	require.NoError(t, err)
	require.Equal(t, 1, set.Len())
	publicKey, ok := set.Key(0)
	require.True(t, ok)

	isPrivate, err := jwk.IsPrivateKey(publicKey)
	require.NoError(t, err)
	require.False(t, isPrivate, "the JWKS must only publish the public key")
	return publicKey
}

func TestLoad(t *testing.T) {
	t.Run("generates and stores an RS256 key when none exists", func(t *testing.T) {
		db, envConfig, instanceID := newTestEnv(t)

		k, err := Load(t.Context(), db, envConfig, instanceID)
		require.NoError(t, err)

		alg, err := k.GetKeyAlg()
		require.NoError(t, err)
		assert.Equal(t, jwa.RS256().String(), alg.String())

		reloaded, err := Load(t.Context(), db, envConfig, instanceID)
		require.NoError(t, err)
		keyID, _ := k.GetKeyID()
		reloadedKeyID, _ := reloaded.GetKeyID()
		assert.Equal(t, keyID, reloadedKeyID, "a second load must reuse the stored key")
	})

	for name, rawKey := range map[string]func(t *testing.T) any{
		"ES256": func(t *testing.T) any {
			key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
			require.NoError(t, err)
			return key
		},
		"EdDSA": func(t *testing.T) any {
			_, key, err := ed25519.GenerateKey(rand.Reader)
			require.NoError(t, err)
			return key
		},
	} {
		t.Run("loads a stored "+name+" key", func(t *testing.T) {
			db, envConfig, instanceID := newTestEnv(t)
			storedKeyID := storeKey(t, db, envConfig, instanceID, rawKey(t))

			k, err := Load(t.Context(), db, envConfig, instanceID)
			require.NoError(t, err)

			alg, err := k.GetKeyAlg()
			require.NoError(t, err)
			assert.Equal(t, name, alg.String())

			publicKey := publicKeyFromJWKS(t, k)
			keyID, _ := publicKey.KeyID()
			assert.Equal(t, storedKeyID, keyID)
			publicAlg, ok := publicKey.Algorithm()
			require.True(t, ok)
			assert.Equal(t, name, publicAlg.String())
		})
	}
}
