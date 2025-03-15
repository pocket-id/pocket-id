package service

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/lestrrat-go/jwx/v3/jwk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJwtService_Init(t *testing.T) {
	t.Run("should generate new key when none exists", func(t *testing.T) {
		// Create a temporary directory for the test
		tempDir := t.TempDir()

		// Create a mock AppConfigService
		appConfigService := &AppConfigService{}

		// Initialize the JWT service
		service := &JwtService{}
		err := service.init(appConfigService, tempDir)
		require.NoError(t, err, "Failed to initialize JWT service")

		// Verify the private key was set
		require.NotNil(t, service.privateKey, "Private key should be set")

		// Verify the key has been saved to disk as JWK
		jwkPath := filepath.Join(tempDir, privateKeyFileJwk)
		_, err = os.Stat(jwkPath)
		assert.NoError(t, err, "JWK file should exist")

		// Verify the generated key is valid
		keyData, err := os.ReadFile(jwkPath)
		require.NoError(t, err)
		key, err := jwk.ParseKey(keyData)
		require.NoError(t, err)

		// Key should have required properties
		keyID, ok := key.KeyID()
		assert.True(t, ok, "Key should have a key ID")
		assert.NotEmpty(t, keyID)

		keyUsage, ok := key.KeyUsage()
		assert.True(t, ok, "Key should have a key usage")
		assert.Equal(t, "sig", keyUsage)
	})

	t.Run("should load existing JWK key", func(t *testing.T) {
		// Create a temporary directory for the test
		tempDir := t.TempDir()

		// First create a service to generate a key
		firstService := &JwtService{}
		err := firstService.init(&AppConfigService{}, tempDir)
		require.NoError(t, err)

		// Get the key ID of the first service
		origKeyID, ok := firstService.privateKey.KeyID()
		require.True(t, ok)

		// Now create a new service that should load the existing key
		secondService := &JwtService{}
		err = secondService.init(&AppConfigService{}, tempDir)
		require.NoError(t, err)

		// Verify the loaded key has the same ID as the original
		loadedKeyID, ok := secondService.privateKey.KeyID()
		require.True(t, ok)
		assert.Equal(t, origKeyID, loadedKeyID, "Loaded key should have the same ID as the original")
	})

	t.Run("should load PEM key and save as JWK", func(t *testing.T) {
		// Create a temporary directory for the test
		tempDir := t.TempDir()

		// Create a PEM encoded key file
		// Note: For simplicity, we'll use init to generate a key first, then convert it to PEM
		initialService := &JwtService{}
		err := initialService.init(&AppConfigService{}, tempDir)
		require.NoError(t, err)

		// Delete the JWK file that was created
		err = os.Remove(filepath.Join(tempDir, privateKeyFileJwk))
		require.NoError(t, err)

		// Export the key as PEM
		var rawKey any
		err = jwk.Export(initialService.privateKey, &rawKey)
		require.NoError(t, err)

		// Write the PEM file
		// For this test, we'll use the jwk library to write a PEM file
		pemPath := filepath.Join(tempDir, privateKeyFilePem)
		pemData, err := jwk.EncodePEM(initialService.privateKey)
		require.NoError(t, err)
		err = os.WriteFile(pemPath, pemData, 0666)
		require.NoError(t, err)

		// Now create a new service that should load the PEM key
		pemLoadService := &JwtService{}
		err = pemLoadService.init(&AppConfigService{}, tempDir)
		require.NoError(t, err)

		// Verify the JWK file was created
		jwkPath := filepath.Join(tempDir, privateKeyFileJwk)
		_, err = os.Stat(jwkPath)
		assert.NoError(t, err, "JWK file should have been created")

		// Verify the loaded key is valid
		require.NotNil(t, pemLoadService.privateKey)
		keyID, ok := pemLoadService.privateKey.KeyID()
		assert.True(t, ok)
		assert.NotEmpty(t, keyID)

		// Verify the loaded key is the same as the original one
		// We remove the KID since that is re-generated when loading from PEM
		initialService.privateKey.Set(jwk.KeyIDKey, "")
		pemLoadService.privateKey.Set(jwk.KeyIDKey, "")
		initialKey, err := json.Marshal(initialService.privateKey)
		require.NoError(t, err)
		pemLoadedKey, err := json.Marshal(pemLoadService.privateKey)
		require.NoError(t, err)
		assert.Equal(t, string(initialKey), string(pemLoadedKey), "Imported key does not match PEM key on disk")
	})

	t.Run("should load existing JWK for EC keys", func(t *testing.T) {
		// Create a temporary directory for the test
		tempDir := t.TempDir()

		// Create a new JWK and save it to disk
		origKeyID := createECKeyJWK(t, tempDir)

		// Now create a new service that should load the existing key
		svc := &JwtService{}
		err := svc.init(&AppConfigService{}, tempDir)
		require.NoError(t, err)

		// Verify the loaded key has the same ID as the original
		loadedKeyID, ok := svc.privateKey.KeyID()
		require.True(t, ok)
		assert.Equal(t, origKeyID, loadedKeyID, "Loaded key should have the same ID as the original")
	})
}

func TestJwtService_GetPublicJWK(t *testing.T) {
	t.Run("returns public key when private key is initialized", func(t *testing.T) {
		// Create a temporary directory for the test
		tempDir := t.TempDir()

		// Create a JWT service with initialized key
		service := &JwtService{}
		err := service.init(&AppConfigService{}, tempDir)
		require.NoError(t, err, "Failed to initialize JWT service")

		// Get the JWK (public key)
		publicKey, err := service.GetPublicJWK()
		require.NoError(t, err, "GetPublicJWK should not return an error when private key is initialized")

		// Verify the returned key is valid
		require.NotNil(t, publicKey, "Public key should not be nil")

		// Validate it's actually a public key
		isPrivate, err := jwk.IsPrivateKey(publicKey)
		require.NoError(t, err)
		assert.False(t, isPrivate, "Returned key should be a public key")

		// Check that key has required properties
		keyID, ok := publicKey.KeyID()
		require.True(t, ok, "Public key should have a key ID")
		assert.NotEmpty(t, keyID, "Key ID should not be empty")

		alg, ok := publicKey.Algorithm()
		require.True(t, ok, "Public key should have an algorithm")
		assert.Equal(t, "RS256", alg.String(), "Algorithm should be RS256")
	})

	t.Run("returns public key when ECDSA private key is initialized", func(t *testing.T) {
		// Create a temporary directory for the test
		tempDir := t.TempDir()

		// Create an ECDSA key and save it as JWK
		originalKeyID := createECKeyJWK(t, tempDir)

		// Create a JWT service that loads the ECDSA key
		service := &JwtService{}
		err := service.init(&AppConfigService{}, tempDir)
		require.NoError(t, err, "Failed to initialize JWT service")

		// Get the JWK (public key)
		publicKey, err := service.GetPublicJWK()
		require.NoError(t, err, "GetPublicJWK should not return an error when private key is initialized")

		// Verify the returned key is valid
		require.NotNil(t, publicKey, "Public key should not be nil")

		// Validate it's actually a public key
		isPrivate, err := jwk.IsPrivateKey(publicKey)
		require.NoError(t, err)
		assert.False(t, isPrivate, "Returned key should be a public key")

		// Check that key has required properties
		keyID, ok := publicKey.KeyID()
		require.True(t, ok, "Public key should have a key ID")
		assert.Equal(t, originalKeyID, keyID, "Key ID should match the original key ID")

		// Check that the key type is EC
		assert.Equal(t, "EC", publicKey.KeyType().String(), "Key type should be EC")

		// Check that the algorithm is ES256
		alg, ok := publicKey.Algorithm()
		require.True(t, ok, "Public key should have an algorithm")
		assert.Equal(t, "ES256", alg.String(), "Algorithm should be ES256")
	})

	t.Run("returns error when private key is not initialized", func(t *testing.T) {
		// Create a service with nil private key
		service := &JwtService{
			privateKey: nil,
		}

		// Try to get the JWK
		publicKey, err := service.GetPublicJWK()

		// Verify it returns an error
		require.Error(t, err, "GetPublicJWK should return an error when private key is nil")
		assert.Contains(t, err.Error(), "key is not initialized", "Error message should indicate key is not initialized")
		assert.Nil(t, publicKey, "Public key should be nil when there's an error")
	})
}

func createECKeyJWK(t *testing.T, path string) string {
	t.Helper()

	// Generate a new P-256 ECDSA key
	privateKeyRaw, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err, "Failed to generate ECDSA key")

	// Import as JWK and save to disk
	privateKey, err := importRawKey(privateKeyRaw)
	require.NoError(t, err, "Failed to import private key")

	err = saveKeyJWK(privateKey, filepath.Join(path, privateKeyFileJwk))
	require.NoError(t, err, "Failed to save key")

	kid, _ := privateKey.KeyID()
	require.NotEmpty(t, kid, "Key ID must be set")

	return kid
}
