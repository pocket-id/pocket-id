package jwk

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/base64"
	"os"
	"path/filepath"
	"testing"

	"github.com/lestrrat-go/jwx/v3/jwk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
	cryptoutils "github.com/pocket-id/pocket-id/backend/internal/utils/crypto"
)

func TestKeyProviderFile_LoadKey(t *testing.T) {
	// Generate a test key to use in our tests
	pk, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	key, err := jwk.Import(pk)
	require.NoError(t, err)

	t.Run("LoadKey with no existing key", func(t *testing.T) {
		tempDir := t.TempDir()

		provider := &KeyProviderFile{}
		err := provider.Init(KeyProviderOpts{
			EnvConfig: &common.EnvConfigSchema{
				KeysPath: tempDir,
			},
		})
		require.NoError(t, err)

		// Load key when none exists
		loadedKey, err := provider.LoadKey()
		require.NoError(t, err)
		assert.Nil(t, loadedKey, "Expected nil key when no key exists")
	})

	t.Run("LoadKey with no existing key (with kek)", func(t *testing.T) {
		tempDir := t.TempDir()

		provider := &KeyProviderFile{}
		err = provider.Init(KeyProviderOpts{
			EnvConfig: &common.EnvConfigSchema{
				KeysPath: tempDir,
			},
			Kek: makeKEK(t),
		})
		require.NoError(t, err)

		// Load key when none exists
		loadedKey, err := provider.LoadKey()
		require.NoError(t, err)
		assert.Nil(t, loadedKey, "Expected nil key when no key exists")
	})

	t.Run("LoadKey with unencrypted key", func(t *testing.T) {
		tempDir := t.TempDir()

		provider := &KeyProviderFile{}
		err := provider.Init(KeyProviderOpts{
			EnvConfig: &common.EnvConfigSchema{
				KeysPath: tempDir,
			},
		})
		require.NoError(t, err)

		// Save a key
		err = provider.SaveKey(key)
		require.NoError(t, err)

		// Make sure the key file exists
		keyPath := filepath.Join(tempDir, PrivateKeyFile)
		exists, err := utils.FileExists(keyPath)
		require.NoError(t, err)
		assert.True(t, exists, "Expected key file to exist")

		// Load the key
		loadedKey, err := provider.LoadKey()
		require.NoError(t, err)
		assert.NotNil(t, loadedKey, "Expected non-nil key when key exists")

		// Verify the loaded key is the same as the original
		keyBytes, err := EncodeJWKBytes(key)
		require.NoError(t, err)

		loadedKeyBytes, err := EncodeJWKBytes(loadedKey)
		require.NoError(t, err)

		assert.Equal(t, keyBytes, loadedKeyBytes, "Expected loaded key to match original key")
	})

	t.Run("LoadKey with encrypted key", func(t *testing.T) {
		tempDir := t.TempDir()

		provider := &KeyProviderFile{}
		err = provider.Init(KeyProviderOpts{
			EnvConfig: &common.EnvConfigSchema{
				KeysPath: tempDir,
			},
			Kek: makeKEK(t),
		})
		require.NoError(t, err)

		// Save a key (will be encrypted)
		err = provider.SaveKey(key)
		require.NoError(t, err)

		// Make sure the encrypted key file exists
		encKeyPath := filepath.Join(tempDir, PrivateKeyFileEncrypted)
		exists, err := utils.FileExists(encKeyPath)
		require.NoError(t, err)
		assert.True(t, exists, "Expected encrypted key file to exist")

		// Make sure the unencrypted key file does not exist
		keyPath := filepath.Join(tempDir, PrivateKeyFile)
		exists, err = utils.FileExists(keyPath)
		require.NoError(t, err)
		assert.False(t, exists, "Expected unencrypted key file to not exist")

		// Load the key
		loadedKey, err := provider.LoadKey()
		require.NoError(t, err)
		assert.NotNil(t, loadedKey, "Expected non-nil key when encrypted key exists")

		// Verify the loaded key is the same as the original
		keyBytes, err := EncodeJWKBytes(key)
		require.NoError(t, err)

		loadedKeyBytes, err := EncodeJWKBytes(loadedKey)
		require.NoError(t, err)

		assert.Equal(t, keyBytes, loadedKeyBytes, "Expected loaded key to match original key")
	})

	t.Run("LoadKey replaces unencrypted key with encrypted key when kek is provided", func(t *testing.T) {
		tempDir := t.TempDir()

		// First, create an unencrypted key
		providerNoKek := &KeyProviderFile{}
		err := providerNoKek.Init(KeyProviderOpts{
			EnvConfig: &common.EnvConfigSchema{
				KeysPath: tempDir,
			},
		})
		require.NoError(t, err)

		// Save an unencrypted key
		err = providerNoKek.SaveKey(key)
		require.NoError(t, err)

		// Verify unencrypted key exists
		keyPath := filepath.Join(tempDir, PrivateKeyFile)
		exists, err := utils.FileExists(keyPath)
		require.NoError(t, err)
		assert.True(t, exists, "Expected unencrypted key file to exist")

		// Now create a provider with a kek
		kek := make([]byte, 32)
		_, err = rand.Read(kek)
		require.NoError(t, err)

		providerWithKek := &KeyProviderFile{}
		err = providerWithKek.Init(KeyProviderOpts{
			EnvConfig: &common.EnvConfigSchema{
				KeysPath: tempDir,
			},
			Kek: kek,
		})
		require.NoError(t, err)

		// Load the key - this should convert the unencrypted key to encrypted
		loadedKey, err := providerWithKek.LoadKey()
		require.NoError(t, err)
		assert.NotNil(t, loadedKey, "Expected non-nil key when loading and converting key")

		// Verify the unencrypted key no longer exists
		exists, err = utils.FileExists(keyPath)
		require.NoError(t, err)
		assert.False(t, exists, "Expected unencrypted key file to be removed")

		// Verify the encrypted key file exists
		encKeyPath := filepath.Join(tempDir, PrivateKeyFileEncrypted)
		exists, err = utils.FileExists(encKeyPath)
		require.NoError(t, err)
		assert.True(t, exists, "Expected encrypted key file to exist after conversion")

		// Verify the key data
		keyBytes, err := EncodeJWKBytes(key)
		require.NoError(t, err)

		loadedKeyBytes, err := EncodeJWKBytes(loadedKey)
		require.NoError(t, err)

		assert.Equal(t, keyBytes, loadedKeyBytes, "Expected loaded key to match original key after conversion")
	})
}

func TestKeyProviderFile_SaveKey(t *testing.T) {
	// Generate a test key to use in our tests
	pk, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	key, err := jwk.Import(pk)
	require.NoError(t, err)

	t.Run("SaveKey unencrypted", func(t *testing.T) {
		tempDir := t.TempDir()

		provider := &KeyProviderFile{}
		err := provider.Init(KeyProviderOpts{
			EnvConfig: &common.EnvConfigSchema{
				KeysPath: tempDir,
			},
		})
		require.NoError(t, err)

		// Save the key
		err = provider.SaveKey(key)
		require.NoError(t, err)

		// Verify the key file exists
		keyPath := filepath.Join(tempDir, PrivateKeyFile)
		exists, err := utils.FileExists(keyPath)
		require.NoError(t, err)
		assert.True(t, exists, "Expected key file to exist")

		// Verify the content of the key file
		data, err := os.ReadFile(keyPath)
		require.NoError(t, err)

		parsedKey, err := jwk.ParseKey(data)
		require.NoError(t, err)

		// Compare the saved key with the original
		keyBytes, err := EncodeJWKBytes(key)
		require.NoError(t, err)

		parsedKeyBytes, err := EncodeJWKBytes(parsedKey)
		require.NoError(t, err)

		assert.Equal(t, keyBytes, parsedKeyBytes, "Expected saved key to match original key")
	})

	t.Run("SaveKey encrypted", func(t *testing.T) {
		tempDir := t.TempDir()

		// Generate a 64-byte kek
		kek := makeKEK(t)

		provider := &KeyProviderFile{}
		err = provider.Init(KeyProviderOpts{
			EnvConfig: &common.EnvConfigSchema{
				KeysPath: tempDir,
			},
			Kek: kek,
		})
		require.NoError(t, err)

		// Save the key (will be encrypted)
		err = provider.SaveKey(key)
		require.NoError(t, err)

		// Verify the encrypted key file exists
		encKeyPath := filepath.Join(tempDir, PrivateKeyFileEncrypted)
		exists, err := utils.FileExists(encKeyPath)
		require.NoError(t, err)
		assert.True(t, exists, "Expected encrypted key file to exist")

		// Verify the unencrypted key file doesn't exist
		keyPath := filepath.Join(tempDir, PrivateKeyFile)
		exists, err = utils.FileExists(keyPath)
		require.NoError(t, err)
		assert.False(t, exists, "Expected unencrypted key file to not exist")

		// Manually decrypt the encrypted key file to verify it contains the correct key
		encB64, err := os.ReadFile(encKeyPath)
		require.NoError(t, err)

		// Decode from base64
		enc := make([]byte, base64.StdEncoding.DecodedLen(len(encB64)))
		n, err := base64.StdEncoding.Decode(enc, encB64)
		require.NoError(t, err)
		enc = enc[:n] // Trim any padding

		// Decrypt the data
		data, err := cryptoutils.Decrypt(kek, enc, nil)
		require.NoError(t, err)

		// Parse the key
		parsedKey, err := jwk.ParseKey(data)
		require.NoError(t, err)

		// Compare the decrypted key with the original
		keyBytes, err := EncodeJWKBytes(key)
		require.NoError(t, err)

		parsedKeyBytes, err := EncodeJWKBytes(parsedKey)
		require.NoError(t, err)

		assert.Equal(t, keyBytes, parsedKeyBytes, "Expected decrypted key to match original key")
	})
}

func makeKEK(t *testing.T) []byte {
	t.Helper()

	// Generate a 32-byte kek
	kek := make([]byte, 32)
	_, err := rand.Read(kek)
	require.NoError(t, err)
	return kek
}
