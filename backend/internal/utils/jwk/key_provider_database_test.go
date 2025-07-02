package jwk

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/base64"
	"testing"

	"github.com/lestrrat-go/jwx/v3/jwk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pocket-id/pocket-id/backend/internal/model"
	cryptoutils "github.com/pocket-id/pocket-id/backend/internal/utils/crypto"
	testutils "github.com/pocket-id/pocket-id/backend/internal/utils/testing"
)

func TestKeyProviderDatabase_Init(t *testing.T) {
	t.Run("Init fails when KEK is not provided", func(t *testing.T) {
		db := testutils.NewDatabaseForTest(t)
		provider := &KeyProviderDatabase{}
		err := provider.Init(KeyProviderOpts{
			DB:  db,
			Kek: nil, // No KEK
		})
		require.Error(t, err, "Expected error when KEK is not provided")
		require.ErrorContains(t, err, "encryption key is required")
	})

	t.Run("Init succeeds with KEK", func(t *testing.T) {
		db := testutils.NewDatabaseForTest(t)
		provider := &KeyProviderDatabase{}
		err := provider.Init(KeyProviderOpts{
			DB:  db,
			Kek: generateTestKEK(t),
		})
		require.NoError(t, err, "Expected no error when KEK is provided")
	})
}

func TestKeyProviderDatabase_LoadKey(t *testing.T) {
	// Generate a test key to use in our tests
	pk, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	key, err := jwk.Import(pk)
	require.NoError(t, err)

	t.Run("LoadKey with no existing key", func(t *testing.T) {
		db := testutils.NewDatabaseForTest(t)
		kek := generateTestKEK(t)

		provider := &KeyProviderDatabase{}
		err := provider.Init(KeyProviderOpts{
			DB:  db,
			Kek: kek,
		})
		require.NoError(t, err)

		// Load key when none exists
		loadedKey, err := provider.LoadKey()
		require.NoError(t, err)
		assert.Nil(t, loadedKey, "Expected nil key when no key exists in database")
	})

	t.Run("LoadKey with existing key", func(t *testing.T) {
		db := testutils.NewDatabaseForTest(t)
		kek := generateTestKEK(t)

		provider := &KeyProviderDatabase{}
		err := provider.Init(KeyProviderOpts{
			DB:  db,
			Kek: kek,
		})
		require.NoError(t, err)

		// Save a key
		err = provider.SaveKey(key)
		require.NoError(t, err)

		// Load the key
		loadedKey, err := provider.LoadKey()
		require.NoError(t, err)
		assert.NotNil(t, loadedKey, "Expected non-nil key when key exists in database")

		// Verify the loaded key is the same as the original
		keyBytes, err := EncodeJWKBytes(key)
		require.NoError(t, err)

		loadedKeyBytes, err := EncodeJWKBytes(loadedKey)
		require.NoError(t, err)

		assert.Equal(t, keyBytes, loadedKeyBytes, "Expected loaded key to match original key")
	})

	t.Run("LoadKey with invalid base64", func(t *testing.T) {
		db := testutils.NewDatabaseForTest(t)
		kek := generateTestKEK(t)

		provider := &KeyProviderDatabase{}
		err := provider.Init(KeyProviderOpts{
			DB:  db,
			Kek: kek,
		})
		require.NoError(t, err)

		// Insert invalid base64 data
		invalidBase64 := "not-valid-base64"
		err = db.Create(&model.KV{
			Key:   PrivateKeyDBKey,
			Value: &invalidBase64,
		}).Error
		require.NoError(t, err)

		// Attempt to load the key
		loadedKey, err := provider.LoadKey()
		require.Error(t, err, "Expected error when loading key with invalid base64")
		require.ErrorContains(t, err, "not a valid base64-encoded value")
		assert.Nil(t, loadedKey, "Expected nil key when loading fails")
	})

	t.Run("LoadKey with invalid encrypted data", func(t *testing.T) {
		db := testutils.NewDatabaseForTest(t)
		kek := generateTestKEK(t)

		provider := &KeyProviderDatabase{}
		err := provider.Init(KeyProviderOpts{
			DB:  db,
			Kek: kek,
		})
		require.NoError(t, err)

		// Insert valid base64 but invalid encrypted data
		invalidData := base64.StdEncoding.EncodeToString([]byte("not-valid-encrypted-data"))
		err = db.Create(&model.KV{
			Key:   PrivateKeyDBKey,
			Value: &invalidData,
		}).Error
		require.NoError(t, err)

		// Attempt to load the key
		loadedKey, err := provider.LoadKey()
		require.Error(t, err, "Expected error when loading key with invalid encrypted data")
		require.ErrorContains(t, err, "failed to decrypt")
		assert.Nil(t, loadedKey, "Expected nil key when loading fails")
	})

	t.Run("LoadKey with valid encrypted data but wrong KEK", func(t *testing.T) {
		db := testutils.NewDatabaseForTest(t)
		originalKek := generateTestKEK(t)

		// Save a key with the original KEK
		originalProvider := &KeyProviderDatabase{}
		err := originalProvider.Init(KeyProviderOpts{
			DB:  db,
			Kek: originalKek,
		})
		require.NoError(t, err)

		err = originalProvider.SaveKey(key)
		require.NoError(t, err)

		// Now try to load with a different KEK
		differentKek := generateTestKEK(t)
		differentProvider := &KeyProviderDatabase{}
		err = differentProvider.Init(KeyProviderOpts{
			DB:  db,
			Kek: differentKek,
		})
		require.NoError(t, err)

		// Attempt to load the key with the wrong KEK
		loadedKey, err := differentProvider.LoadKey()
		require.Error(t, err, "Expected error when loading key with wrong KEK")
		require.ErrorContains(t, err, "failed to decrypt")
		assert.Nil(t, loadedKey, "Expected nil key when loading fails")
	})

	t.Run("LoadKey with invalid key data", func(t *testing.T) {
		db := testutils.NewDatabaseForTest(t)
		kek := generateTestKEK(t)

		provider := &KeyProviderDatabase{}
		err := provider.Init(KeyProviderOpts{
			DB:  db,
			Kek: kek,
		})
		require.NoError(t, err)

		// Create invalid key data (valid JSON but not a valid JWK)
		invalidKeyData := []byte(`{"not": "a valid jwk"}`)

		// Encrypt the invalid key data
		encryptedData, err := cryptoutils.Encrypt(kek, invalidKeyData, nil)
		require.NoError(t, err)

		// Base64 encode the encrypted data
		encodedData := base64.StdEncoding.EncodeToString(encryptedData)

		// Save to database
		err = db.Create(&model.KV{
			Key:   PrivateKeyDBKey,
			Value: &encodedData,
		}).Error
		require.NoError(t, err)

		// Attempt to load the key
		loadedKey, err := provider.LoadKey()
		require.Error(t, err, "Expected error when loading invalid key data")
		require.ErrorContains(t, err, "failed to parse")
		assert.Nil(t, loadedKey, "Expected nil key when loading fails")
	})
}

func TestKeyProviderDatabase_SaveKey(t *testing.T) {
	// Generate a test key to use in our tests
	pk, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	key, err := jwk.Import(pk)
	require.NoError(t, err)

	t.Run("SaveKey and verify database record", func(t *testing.T) {
		db := testutils.NewDatabaseForTest(t)
		kek := generateTestKEK(t)

		provider := &KeyProviderDatabase{}
		err := provider.Init(KeyProviderOpts{
			DB:  db,
			Kek: kek,
		})
		require.NoError(t, err)

		// Save the key
		err = provider.SaveKey(key)
		require.NoError(t, err, "Expected no error when saving key")

		// Verify record exists in database
		var kv model.KV
		err = db.Where("key = ?", PrivateKeyDBKey).First(&kv).Error
		require.NoError(t, err, "Expected to find key in database")
		require.NotNil(t, kv.Value, "Expected non-nil value in database")
		assert.NotEmpty(t, *kv.Value, "Expected non-empty value in database")

		// Decode and decrypt to verify content
		encBytes, err := base64.StdEncoding.DecodeString(*kv.Value)
		require.NoError(t, err, "Expected valid base64 encoding")

		decBytes, err := cryptoutils.Decrypt(kek, encBytes, nil)
		require.NoError(t, err, "Expected valid encrypted data")

		parsedKey, err := jwk.ParseKey(decBytes)
		require.NoError(t, err, "Expected valid JWK data")

		// Compare keys
		keyBytes, err := EncodeJWKBytes(key)
		require.NoError(t, err)

		parsedKeyBytes, err := EncodeJWKBytes(parsedKey)
		require.NoError(t, err)

		assert.Equal(t, keyBytes, parsedKeyBytes, "Expected saved key to match original key")
	})
}

func generateTestKEK(t *testing.T) []byte {
	t.Helper()

	// Generate a 32-byte kek
	kek := make([]byte, 32)
	_, err := rand.Read(kek)
	require.NoError(t, err)
	return kek
}
