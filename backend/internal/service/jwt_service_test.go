package service

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/lestrrat-go/jwx/v3/jwk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/model"
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
		jwkPath := filepath.Join(tempDir, PrivateKeyFile)
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

func TestGenerateVerifyAccessToken(t *testing.T) {
	// Create a temporary directory for the test
	tempDir := t.TempDir()

	// Initialize the JWT service with a mock AppConfigService
	mockConfig := &AppConfigService{
		DbConfig: &model.AppConfig{
			SessionDuration: model.AppConfigVariable{Value: "60"}, // 60 minutes
		},
	}

	// Setup the environment variable required by the token verification
	originalAppURL := common.EnvConfig.AppURL
	common.EnvConfig.AppURL = "https://test.example.com"
	defer func() {
		common.EnvConfig.AppURL = originalAppURL
	}()

	t.Run("generates token for regular user", func(t *testing.T) {
		// Create a JWT service
		service := &JwtService{}
		err := service.init(mockConfig, tempDir)
		require.NoError(t, err, "Failed to initialize JWT service")

		// Create a test user
		user := model.User{
			Base: model.Base{
				ID: "user123",
			},
			Email:   "user@example.com",
			IsAdmin: false,
		}

		// Generate a token
		tokenString, err := service.GenerateAccessToken(user)
		require.NoError(t, err, "Failed to generate access token")
		assert.NotEmpty(t, tokenString, "Token should not be empty")

		// Verify the token
		claims, err := service.VerifyAccessToken(tokenString)
		require.NoError(t, err, "Failed to verify generated token")

		// Check the claims
		assert.Equal(t, user.ID, claims.Subject, "Token subject should match user ID")
		assert.Equal(t, false, claims.IsAdmin, "IsAdmin should be false")
		assert.Contains(t, claims.Audience, "https://test.example.com", "Audience should contain the app URL")

		// Check token expiration time is approximately 60 minutes from now
		expectedExp := time.Now().Add(60 * time.Minute)
		tokenExp := claims.ExpiresAt.Time
		timeDiff := expectedExp.Sub(tokenExp).Minutes()
		assert.InDelta(t, 0, timeDiff, 1.0, "Token should expire in approximately 60 minutes")
	})

	t.Run("generates token for admin user", func(t *testing.T) {
		// Create a JWT service
		service := &JwtService{}
		err := service.init(mockConfig, tempDir)
		require.NoError(t, err, "Failed to initialize JWT service")

		// Create a test admin user
		adminUser := model.User{
			Base: model.Base{
				ID: "admin123",
			},
			Email:   "admin@example.com",
			IsAdmin: true,
		}

		// Generate a token
		tokenString, err := service.GenerateAccessToken(adminUser)
		require.NoError(t, err, "Failed to generate access token")

		// Verify the token
		claims, err := service.VerifyAccessToken(tokenString)
		require.NoError(t, err, "Failed to verify generated token")

		// Check the IsAdmin claim is true
		assert.Equal(t, true, claims.IsAdmin, "IsAdmin should be true for admin users")
		assert.Equal(t, adminUser.ID, claims.Subject, "Token subject should match admin ID")
	})

	t.Run("uses session duration from config", func(t *testing.T) {
		// Create a JWT service with a different session duration
		customMockConfig := &AppConfigService{
			DbConfig: &model.AppConfig{
				SessionDuration: model.AppConfigVariable{Value: "30"}, // 30 minutes
			},
		}

		service := &JwtService{}
		err := service.init(customMockConfig, tempDir)
		require.NoError(t, err, "Failed to initialize JWT service")

		// Create a test user
		user := model.User{
			Base: model.Base{
				ID: "user456",
			},
		}

		// Generate a token
		tokenString, err := service.GenerateAccessToken(user)
		require.NoError(t, err, "Failed to generate access token")

		// Verify the token
		claims, err := service.VerifyAccessToken(tokenString)
		require.NoError(t, err, "Failed to verify generated token")

		// Check token expiration time is approximately 30 minutes from now
		expectedExp := time.Now().Add(30 * time.Minute)
		tokenExp := claims.ExpiresAt.Time
		timeDiff := expectedExp.Sub(tokenExp).Minutes()
		assert.InDelta(t, 0, timeDiff, 1.0, "Token should expire in approximately 30 minutes")
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

	err = SaveKeyJWK(privateKey, filepath.Join(path, PrivateKeyFile))
	require.NoError(t, err, "Failed to save key")

	kid, _ := privateKey.KeyID()
	require.NotEmpty(t, kid, "Key ID must be set")

	return kid
}
