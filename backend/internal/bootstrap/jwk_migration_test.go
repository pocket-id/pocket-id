package bootstrap

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"

	"github.com/lestrrat-go/jwx/v3/jwa"
	"github.com/lestrrat-go/jwx/v3/jwk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pocket-id/pocket-id/backend/internal/service"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
)

func TestMigrateKey(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()

	t.Run("no keys exist", func(t *testing.T) {
		// Test when no keys exist
		err := migrateKeyInternal(tempDir)
		require.NoError(t, err)
	})

	t.Run("jwk already exists", func(t *testing.T) {
		// Create a JWK file
		jwkPath := filepath.Join(tempDir, service.PrivateKeyFile)
		key, err := createTestRSAKey()
		require.NoError(t, err)
		err = service.SaveKeyJWK(key, jwkPath)
		require.NoError(t, err)

		// Run migration - should do nothing
		err = migrateKeyInternal(tempDir)
		require.NoError(t, err)

		// Check the file still exists
		exists, err := utils.FileExists(jwkPath)
		require.NoError(t, err)
		assert.True(t, exists)

		// Delete for next test
		err = os.Remove(jwkPath)
		require.NoError(t, err)
	})

	t.Run("migrate pem to jwk", func(t *testing.T) {
		// Create a PEM file
		pemPath := filepath.Join(tempDir, privateKeyFilePem)
		jwkPath := filepath.Join(tempDir, service.PrivateKeyFile)

		// Generate RSA key and save as PEM
		createRSAPrivateKeyPEM(t, pemPath)

		// Run migration
		err := migrateKeyInternal(tempDir)
		require.NoError(t, err)

		// Check PEM file is gone
		exists, err := utils.FileExists(pemPath)
		require.NoError(t, err)
		assert.False(t, exists)

		// Check JWK file exists
		exists, err = utils.FileExists(jwkPath)
		require.NoError(t, err)
		assert.True(t, exists)

		// Verify the JWK can be loaded
		data, err := os.ReadFile(jwkPath)
		require.NoError(t, err)

		_, err = jwk.ParseKey(data)
		require.NoError(t, err)
	})
}

func TestLoadKeyPEM(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()

	t.Run("successfully load PEM key", func(t *testing.T) {
		pemPath := filepath.Join(tempDir, "test_key.pem")

		// Generate RSA key and save as PEM
		createRSAPrivateKeyPEM(t, pemPath)

		// Load the key
		key, err := loadKeyPEM(pemPath)
		require.NoError(t, err)

		// Verify key properties
		assert.NotEmpty(t, key)

		// Check key ID is set
		var keyID string
		err = key.Get(jwk.KeyIDKey, &keyID)
		require.NoError(t, err)
		assert.NotEmpty(t, keyID)

		// Check algorithm is set
		var alg jwa.SignatureAlgorithm
		err = key.Get(jwk.AlgorithmKey, &alg)
		require.NoError(t, err)
		assert.NotEmpty(t, alg)

		// Check key usage is set
		var keyUsage string
		err = key.Get(jwk.KeyUsageKey, &keyUsage)
		require.NoError(t, err)
		assert.Equal(t, service.KeyUsageSigning, keyUsage)
	})

	t.Run("file not found", func(t *testing.T) {
		key, err := loadKeyPEM(filepath.Join(tempDir, "nonexistent.pem"))
		require.Error(t, err)
		assert.Nil(t, key)
	})

	t.Run("invalid file content", func(t *testing.T) {
		invalidPath := filepath.Join(tempDir, "invalid.pem")
		err := os.WriteFile(invalidPath, []byte("not a valid PEM"), 0600)
		require.NoError(t, err)

		key, err := loadKeyPEM(invalidPath)
		require.Error(t, err)
		assert.Nil(t, key)
	})
}

func TestGenerateKeyID(t *testing.T) {
	key, err := createTestRSAKey()
	require.NoError(t, err)

	keyID, err := generateKeyID(key)
	require.NoError(t, err)

	// Key ID should be non-empty
	assert.NotEmpty(t, keyID)

	// Generate another key ID to prove it depends on the key
	key2, err := createTestRSAKey()
	require.NoError(t, err)

	keyID2, err := generateKeyID(key2)
	require.NoError(t, err)

	// The two key IDs should be different
	assert.NotEqual(t, keyID, keyID2)
}

// Helper functions

func createTestRSAKey() (jwk.Key, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}

	key, err := jwk.Import(privateKey)
	if err != nil {
		return nil, err
	}

	return key, nil
}

// createRSAPrivateKeyPEM generates an RSA private key and returns its PEM-encoded form
func createRSAPrivateKeyPEM(t *testing.T, pemPath string) ([]byte, *rsa.PrivateKey) {
	// Generate RSA key
	privKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	// Encode to PEM format
	pemData := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privKey),
	})

	err = os.WriteFile(pemPath, pemData, 0600)
	require.NoError(t, err)

	return pemData, privKey
}
