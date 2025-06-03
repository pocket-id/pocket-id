package service

import (
	"context"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/lestrrat-go/jwx/v3/jwa"
	"github.com/lestrrat-go/jwx/v3/jwk"
	"github.com/lestrrat-go/jwx/v3/jwt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
)

func TestJwtService_Init(t *testing.T) {
	mockConfig := NewTestAppConfigService(&model.AppConfig{
		SessionDuration: model.AppConfigVariable{Value: "60"}, // 60 minutes
	})

	t.Run("should generate new key when none exists", func(t *testing.T) {
		// Create a temporary directory for the test
		tempDir := t.TempDir()

		// Initialize the JWT service
		service := &JwtService{}
		err := service.init(mockConfig, tempDir)
		require.NoError(t, err, "Failed to initialize JWT service")

		// Verify the private key was set
		require.NotNil(t, service.privateKey, "Private key should be set")

		// Verify the key has been saved to disk as JWK
		jwkPath := filepath.Join(tempDir, PrivateKeyFile)
		_, err = os.Stat(jwkPath)
		require.NoError(t, err, "JWK file should exist")

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
		err := firstService.init(mockConfig, tempDir)
		require.NoError(t, err)

		// Get the key ID of the first service
		origKeyID, ok := firstService.privateKey.KeyID()
		require.True(t, ok)

		// Now create a new service that should load the existing key
		secondService := &JwtService{}
		err = secondService.init(mockConfig, tempDir)
		require.NoError(t, err)

		// Verify the loaded key has the same ID as the original
		loadedKeyID, ok := secondService.privateKey.KeyID()
		require.True(t, ok)
		assert.Equal(t, origKeyID, loadedKeyID, "Loaded key should have the same ID as the original")
	})

	t.Run("should load existing JWK for ECDSA keys", func(t *testing.T) {
		// Create a temporary directory for the test
		tempDir := t.TempDir()

		// Create a new JWK and save it to disk
		origKeyID := createECDSAKeyJWK(t, tempDir)

		// Now create a new service that should load the existing key
		svc := &JwtService{}
		err := svc.init(mockConfig, tempDir)
		require.NoError(t, err)

		// Ensure loaded key has the right algorithm
		alg, ok := svc.privateKey.Algorithm()
		_ = assert.True(t, ok) &&
			assert.Equal(t, jwa.ES256().String(), alg.String(), "Loaded key has the incorrect algorithm")

		// Verify the loaded key has the same ID as the original
		loadedKeyID, ok := svc.privateKey.KeyID()
		_ = assert.True(t, ok) &&
			assert.Equal(t, origKeyID, loadedKeyID, "Loaded key should have the same ID as the original")
	})

	t.Run("should load existing JWK for EdDSA keys", func(t *testing.T) {
		// Create a temporary directory for the test
		tempDir := t.TempDir()

		// Create a new JWK and save it to disk
		origKeyID := createEdDSAKeyJWK(t, tempDir)

		// Now create a new service that should load the existing key
		svc := &JwtService{}
		err := svc.init(mockConfig, tempDir)
		require.NoError(t, err)

		// Ensure loaded key has the right algorithm and curve
		alg, ok := svc.privateKey.Algorithm()
		_ = assert.True(t, ok) &&
			assert.Equal(t, jwa.EdDSA().String(), alg.String(), "Loaded key has the incorrect algorithm")

		var curve jwa.EllipticCurveAlgorithm
		err = svc.privateKey.Get("crv", &curve)
		_ = assert.NoError(t, err, "Failed to get 'crv' claim") &&
			assert.Equal(t, jwa.Ed25519().String(), curve.String(), "Curve does not match expected value")

		// Verify the loaded key has the same ID as the original
		loadedKeyID, ok := svc.privateKey.KeyID()
		_ = assert.True(t, ok) &&
			assert.Equal(t, origKeyID, loadedKeyID, "Loaded key should have the same ID as the original")
	})
}

func TestJwtService_GetPublicJWK(t *testing.T) {
	mockConfig := NewTestAppConfigService(&model.AppConfig{
		SessionDuration: model.AppConfigVariable{Value: "60"}, // 60 minutes
	})

	t.Run("returns public key when private key is initialized", func(t *testing.T) {
		// Create a temporary directory for the test
		tempDir := t.TempDir()

		// Create a JWT service with initialized key
		service := &JwtService{}
		err := service.init(mockConfig, tempDir)
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
		originalKeyID := createECDSAKeyJWK(t, tempDir)

		// Create a JWT service that loads the ECDSA key
		service := &JwtService{}
		err := service.init(mockConfig, tempDir)
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

	t.Run("returns public key when EdDSA private key is initialized", func(t *testing.T) {
		// Create a temporary directory for the test
		tempDir := t.TempDir()

		// Create an EdDSA key and save it as JWK
		originalKeyID := createEdDSAKeyJWK(t, tempDir)

		// Create a JWT service that loads the EdDSA key
		service := &JwtService{}
		err := service.init(mockConfig, tempDir)
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

		// Check that the key type is OKP
		assert.Equal(t, "OKP", publicKey.KeyType().String(), "Key type should be OKP")

		// Check that the algorithm is EdDSA
		alg, ok := publicKey.Algorithm()
		require.True(t, ok, "Public key should have an algorithm")
		assert.Equal(t, "EdDSA", alg.String(), "Algorithm should be EdDSA")
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
	mockConfig := NewTestAppConfigService(&model.AppConfig{
		SessionDuration: model.AppConfigVariable{Value: "60"}, // 60 minutes
	})

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
		subject, ok := claims.Subject()
		_ = assert.True(t, ok, "User ID not found in token") &&
			assert.Equal(t, user.ID, subject, "Token subject should match user ID")
		isAdmin, err := GetIsAdmin(claims)
		_ = assert.NoError(t, err, "Failed to get isAdmin claim") &&
			assert.False(t, isAdmin, "isAdmin should be false")
		audience, ok := claims.Audience()
		_ = assert.True(t, ok, "Audience not found in token") &&
			assert.Equal(t, []string{"https://test.example.com"}, audience, "Audience should contain the app URL")

		// Check token expiration time is approximately 1 hour from now
		expectedExp := time.Now().Add(1 * time.Hour)
		expiration, ok := claims.Expiration()
		assert.True(t, ok, "Expiration not found in token")
		timeDiff := expectedExp.Sub(expiration).Minutes()
		assert.InDelta(t, 0, timeDiff, 1.0, "Token should expire in approximately 1 hour")
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
		isAdmin, err := GetIsAdmin(claims)
		_ = assert.NoError(t, err, "Failed to get isAdmin claim") &&
			assert.True(t, isAdmin, "isAdmin should be true")
		subject, ok := claims.Subject()
		_ = assert.True(t, ok, "User ID not found in token") &&
			assert.Equal(t, adminUser.ID, subject, "Token subject should match user ID")
	})

	t.Run("uses session duration from config", func(t *testing.T) {
		// Create a JWT service with a different session duration
		customMockConfig := NewTestAppConfigService(&model.AppConfig{
			SessionDuration: model.AppConfigVariable{Value: "30"}, // 30 minutes
		})

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
		expiration, ok := claims.Expiration()
		assert.True(t, ok, "Expiration not found in token")
		timeDiff := expectedExp.Sub(expiration).Minutes()
		assert.InDelta(t, 0, timeDiff, 1.0, "Token should expire in approximately 30 minutes")
	})

	t.Run("works with Ed25519 keys", func(t *testing.T) {
		// Create a temporary directory for the test
		tempDir := t.TempDir()

		// Create an Ed25519 key and save it as JWK
		origKeyID := createEdDSAKeyJWK(t, tempDir)

		// Create a JWT service that loads the key
		service := &JwtService{}
		err := service.init(mockConfig, tempDir)
		require.NoError(t, err, "Failed to initialize JWT service")

		// Verify it loaded the right key
		loadedKeyID, ok := service.privateKey.KeyID()
		require.True(t, ok)
		assert.Equal(t, origKeyID, loadedKeyID, "Loaded key should have the same ID as the original")

		// Create a test user
		user := model.User{
			Base: model.Base{
				ID: "eddsauser123",
			},
			Email:   "eddsauser@example.com",
			IsAdmin: true,
		}

		// Generate a token
		tokenString, err := service.GenerateAccessToken(user)
		require.NoError(t, err, "Failed to generate access token with Ed25519 key")
		assert.NotEmpty(t, tokenString, "Token should not be empty")

		// Verify the token
		claims, err := service.VerifyAccessToken(tokenString)
		require.NoError(t, err, "Failed to verify generated token with Ed25519 key")

		// Check the claims
		subject, ok := claims.Subject()
		_ = assert.True(t, ok, "User ID not found in token") &&
			assert.Equal(t, user.ID, subject, "Token subject should match user ID")
		isAdmin, err := GetIsAdmin(claims)
		_ = assert.NoError(t, err, "Failed to get isAdmin claim") &&
			assert.True(t, isAdmin, "isAdmin should be true")

		// Verify the key type is OKP
		publicKey, err := service.GetPublicJWK()
		require.NoError(t, err)
		assert.Equal(t, "OKP", publicKey.KeyType().String(), "Key type should be OKP")

		// Verify the algorithm is EdDSA
		alg, ok := publicKey.Algorithm()
		require.True(t, ok)
		assert.Equal(t, "EdDSA", alg.String(), "Algorithm should be EdDSA")
	})

	t.Run("works with P-256 keys", func(t *testing.T) {
		// Create a temporary directory for the test
		tempDir := t.TempDir()

		// Create an ECDSA key and save it as JWK
		origKeyID := createECDSAKeyJWK(t, tempDir)

		// Create a JWT service that loads the key
		service := &JwtService{}
		err := service.init(mockConfig, tempDir)
		require.NoError(t, err, "Failed to initialize JWT service")

		// Verify it loaded the right key
		loadedKeyID, ok := service.privateKey.KeyID()
		require.True(t, ok)
		assert.Equal(t, origKeyID, loadedKeyID, "Loaded key should have the same ID as the original")

		// Create a test user
		user := model.User{
			Base: model.Base{
				ID: "ecdsauser123",
			},
			Email:   "ecdsauser@example.com",
			IsAdmin: true,
		}

		// Generate a token
		tokenString, err := service.GenerateAccessToken(user)
		require.NoError(t, err, "Failed to generate access token with ECDSA key")
		assert.NotEmpty(t, tokenString, "Token should not be empty")

		// Verify the token
		claims, err := service.VerifyAccessToken(tokenString)
		require.NoError(t, err, "Failed to verify generated token with ECDSA key")

		// Check the claims
		subject, ok := claims.Subject()
		_ = assert.True(t, ok, "User ID not found in token") &&
			assert.Equal(t, user.ID, subject, "Token subject should match user ID")
		isAdmin, err := GetIsAdmin(claims)
		_ = assert.NoError(t, err, "Failed to get isAdmin claim") &&
			assert.True(t, isAdmin, "isAdmin should be true")

		// Verify the key type is EC
		publicKey, err := service.GetPublicJWK()
		require.NoError(t, err)
		assert.Equal(t, jwa.EC().String(), publicKey.KeyType().String(), "Key type should be EC")

		// Verify the algorithm is ES256
		alg, ok := publicKey.Algorithm()
		require.True(t, ok)
		assert.Equal(t, jwa.ES256().String(), alg.String(), "Algorithm should be ES256")
	})

	t.Run("works with RSA-4096 keys", func(t *testing.T) {
		// Create a temporary directory for the test
		tempDir := t.TempDir()

		// Create an RSA-4096 key and save it as JWK
		origKeyID := createRSA4096KeyJWK(t, tempDir)

		// Create a JWT service that loads the key
		service := &JwtService{}
		err := service.init(mockConfig, tempDir)
		require.NoError(t, err, "Failed to initialize JWT service")

		// Verify it loaded the right key
		loadedKeyID, ok := service.privateKey.KeyID()
		require.True(t, ok)
		assert.Equal(t, origKeyID, loadedKeyID, "Loaded key should have the same ID as the original")

		// Create a test user
		user := model.User{
			Base: model.Base{
				ID: "rsauser123",
			},
			Email:   "rsauser@example.com",
			IsAdmin: true,
		}

		// Generate a token
		tokenString, err := service.GenerateAccessToken(user)
		require.NoError(t, err, "Failed to generate access token with RSA key")
		assert.NotEmpty(t, tokenString, "Token should not be empty")

		// Verify the token
		claims, err := service.VerifyAccessToken(tokenString)
		require.NoError(t, err, "Failed to verify generated token with RSA key")

		// Check the claims
		subject, ok := claims.Subject()
		_ = assert.True(t, ok, "User ID not found in token") &&
			assert.Equal(t, user.ID, subject, "Token subject should match user ID")
		isAdmin, err := GetIsAdmin(claims)
		_ = assert.NoError(t, err, "Failed to get isAdmin claim") &&
			assert.True(t, isAdmin, "isAdmin should be true")

		// Verify the key type is RSA
		publicKey, err := service.GetPublicJWK()
		require.NoError(t, err)
		assert.Equal(t, jwa.RSA().String(), publicKey.KeyType().String(), "Key type should be RSA")

		// Verify the algorithm is RS256
		alg, ok := publicKey.Algorithm()
		require.True(t, ok)
		assert.Equal(t, jwa.RS256().String(), alg.String(), "Algorithm should be RS256")
	})
}

func TestGenerateVerifyIdToken(t *testing.T) {
	// Create a temporary directory for the test
	tempDir := t.TempDir()

	// Initialize the JWT service with a mock AppConfigService
	mockConfig := NewTestAppConfigService(&model.AppConfig{
		SessionDuration: model.AppConfigVariable{Value: "60"}, // 60 minutes
	})

	// Setup the environment variable required by the token verification
	originalAppURL := common.EnvConfig.AppURL
	common.EnvConfig.AppURL = "https://test.example.com"
	defer func() {
		common.EnvConfig.AppURL = originalAppURL
	}()

	t.Run("generates and verifies ID token with standard claims", func(t *testing.T) {
		// Create a JWT service
		service := &JwtService{}
		err := service.init(mockConfig, tempDir)
		require.NoError(t, err, "Failed to initialize JWT service")

		// Create test claims
		userClaims := map[string]interface{}{
			"sub":   "user123",
			"name":  "Test User",
			"email": "user@example.com",
		}
		const clientID = "test-client-123"

		// Generate a token
		tokenString, err := service.GenerateIDToken(userClaims, clientID, "")
		require.NoError(t, err, "Failed to generate ID token")
		assert.NotEmpty(t, tokenString, "Token should not be empty")

		// Verify the token
		claims, err := service.VerifyIdToken(tokenString, false)
		require.NoError(t, err, "Failed to verify generated ID token")

		// Check the claims
		subject, ok := claims.Subject()
		_ = assert.True(t, ok, "User ID not found in token") &&
			assert.Equal(t, "user123", subject, "Token subject should match user ID")
		audience, ok := claims.Audience()
		_ = assert.True(t, ok, "Audience not found in token") &&
			assert.Equal(t, []string{clientID}, audience, "Audience should contain the client ID")
		issuer, ok := claims.Issuer()
		_ = assert.True(t, ok, "Issuer not found in token") &&
			assert.Equal(t, common.EnvConfig.AppURL, issuer, "Issuer should match app URL")

		// Check token expiration time is approximately 1 hour from now
		expectedExp := time.Now().Add(1 * time.Hour)
		expiration, ok := claims.Expiration()
		assert.True(t, ok, "Expiration not found in token")
		timeDiff := expectedExp.Sub(expiration).Minutes()
		assert.InDelta(t, 0, timeDiff, 1.0, "Token should expire in approximately 1 hour")
	})

	t.Run("can accept expired tokens if told so", func(t *testing.T) {
		// Create a JWT service
		service := &JwtService{}
		err := service.init(mockConfig, tempDir)
		require.NoError(t, err, "Failed to initialize JWT service")

		// Create test claims
		userClaims := map[string]interface{}{
			"sub":   "user123",
			"name":  "Test User",
			"email": "user@example.com",
		}
		const clientID = "test-client-123"

		// Create a token that's already expired
		token, err := jwt.NewBuilder().
			Subject(userClaims["sub"].(string)).
			Issuer(common.EnvConfig.AppURL).
			Audience([]string{clientID}).
			IssuedAt(time.Now().Add(-2 * time.Hour)).
			Expiration(time.Now().Add(-1 * time.Hour)). // Expired 1 hour ago
			Build()
		require.NoError(t, err, "Failed to build token")

		err = SetTokenType(token, IDTokenJWTType)
		require.NoError(t, err, "Failed to set token type")

		// Add custom claims
		for k, v := range userClaims {
			if k != "sub" { // Already set above
				err = token.Set(k, v)
				require.NoError(t, err, "Failed to set claim")
			}
		}

		// Sign the token
		signed, err := jwt.Sign(token, jwt.WithKey(jwa.RS256(), service.privateKey))
		require.NoError(t, err, "Failed to sign token")
		tokenString := string(signed)

		// Verify the token without allowExpired flag - should fail
		_, err = service.VerifyIdToken(tokenString, false)
		require.Error(t, err, "Verification should fail with expired token when not allowing expired tokens")
		assert.Contains(t, err.Error(), `"exp" not satisfied`, "Error message should indicate token verification failure")

		// Verify the token with allowExpired flag - should succeed
		claims, err := service.VerifyIdToken(tokenString, true)
		require.NoError(t, err, "Verification should succeed with expired token when allowing expired tokens")

		// Validate the claims
		subject, ok := claims.Subject()
		_ = assert.True(t, ok, "User ID not found in token") &&
			assert.Equal(t, userClaims["sub"], subject, "Token subject should match user ID")
		issuer, ok := claims.Issuer()
		_ = assert.True(t, ok, "Issuer not found in token") &&
			assert.Equal(t, common.EnvConfig.AppURL, issuer, "Issuer should match app URL")
	})

	t.Run("generates and verifies ID token with nonce", func(t *testing.T) {
		// Create a JWT service
		service := &JwtService{}
		err := service.init(mockConfig, tempDir)
		require.NoError(t, err, "Failed to initialize JWT service")

		// Create test claims with nonce
		userClaims := map[string]interface{}{
			"sub":  "user456",
			"name": "Another User",
		}
		const clientID = "test-client-456"
		nonce := "random-nonce-value"

		// Generate a token with nonce
		tokenString, err := service.GenerateIDToken(userClaims, clientID, nonce)
		require.NoError(t, err, "Failed to generate ID token with nonce")

		// Parse the token manually to check nonce
		publicKey, err := service.GetPublicJWK()
		require.NoError(t, err, "Failed to get public key")
		token, err := jwt.Parse([]byte(tokenString), jwt.WithKey(jwa.RS256(), publicKey))
		require.NoError(t, err, "Failed to parse token")

		var tokenNonce string
		err = token.Get("nonce", &tokenNonce)
		require.NoError(t, err, "Failed to get claims")

		assert.Equal(t, nonce, tokenNonce, "Token should contain the correct nonce")
	})

	t.Run("fails verification with incorrect issuer", func(t *testing.T) {
		// Create a JWT service
		service := &JwtService{}
		err := service.init(mockConfig, tempDir)
		require.NoError(t, err, "Failed to initialize JWT service")

		// Generate a token with standard claims
		userClaims := map[string]interface{}{
			"sub": "user789",
		}
		tokenString, err := service.GenerateIDToken(userClaims, "client-789", "")
		require.NoError(t, err, "Failed to generate ID token")

		// Temporarily change the app URL to simulate wrong issuer
		common.EnvConfig.AppURL = "https://wrong-issuer.com"

		// Verify should fail due to issuer mismatch
		_, err = service.VerifyIdToken(tokenString, false)
		require.Error(t, err, "Verification should fail with incorrect issuer")
		assert.Contains(t, err.Error(), `"iss" not satisfied`, "Error message should indicate token verification failure")
	})

	t.Run("works with Ed25519 keys", func(t *testing.T) {
		// Create a temporary directory for the test
		tempDir := t.TempDir()

		// Create an Ed25519 key and save it as JWK
		origKeyID := createEdDSAKeyJWK(t, tempDir)

		// Create a JWT service that loads the key
		service := &JwtService{}
		err := service.init(mockConfig, tempDir)
		require.NoError(t, err, "Failed to initialize JWT service")

		// Verify it loaded the right key
		loadedKeyID, ok := service.privateKey.KeyID()
		require.True(t, ok)
		assert.Equal(t, origKeyID, loadedKeyID, "Loaded key should have the same ID as the original")

		// Create test claims
		userClaims := map[string]interface{}{
			"sub":   "eddsauser456",
			"name":  "EdDSA User",
			"email": "eddsauser@example.com",
		}
		const clientID = "eddsa-client-123"

		// Generate a token
		tokenString, err := service.GenerateIDToken(userClaims, clientID, "")
		require.NoError(t, err, "Failed to generate ID token with key")
		assert.NotEmpty(t, tokenString, "Token should not be empty")

		// Verify the token
		claims, err := service.VerifyIdToken(tokenString, false)
		require.NoError(t, err, "Failed to verify generated ID token with key")

		// Check the claims
		subject, ok := claims.Subject()
		_ = assert.True(t, ok, "User ID not found in token") &&
			assert.Equal(t, "eddsauser456", subject, "Token subject should match user ID")
		issuer, ok := claims.Issuer()
		_ = assert.True(t, ok, "Issuer not found in token") &&
			assert.Equal(t, common.EnvConfig.AppURL, issuer, "Issuer should match app URL")

		// Verify the key type is OKP
		publicKey, err := service.GetPublicJWK()
		require.NoError(t, err)
		assert.Equal(t, jwa.OKP().String(), publicKey.KeyType().String(), "Key type should be OKP")

		// Verify the algorithm is EdDSA
		alg, ok := publicKey.Algorithm()
		require.True(t, ok)
		assert.Equal(t, jwa.EdDSA().String(), alg.String(), "Algorithm should be EdDSA")
	})

	t.Run("works with P-256 keys", func(t *testing.T) {
		// Create a temporary directory for the test
		tempDir := t.TempDir()

		// Create an ECDSA key and save it as JWK
		origKeyID := createECDSAKeyJWK(t, tempDir)

		// Create a JWT service that loads the key
		service := &JwtService{}
		err := service.init(mockConfig, tempDir)
		require.NoError(t, err, "Failed to initialize JWT service")

		// Verify it loaded the right key
		loadedKeyID, ok := service.privateKey.KeyID()
		require.True(t, ok)
		assert.Equal(t, origKeyID, loadedKeyID, "Loaded key should have the same ID as the original")

		// Create test claims
		userClaims := map[string]interface{}{
			"sub":   "ecdsauser456",
			"name":  "ECDSA User",
			"email": "ecdsauser@example.com",
		}
		const clientID = "ecdsa-client-123"

		// Generate a token
		tokenString, err := service.GenerateIDToken(userClaims, clientID, "")
		require.NoError(t, err, "Failed to generate ID token with key")
		assert.NotEmpty(t, tokenString, "Token should not be empty")

		// Verify the token
		claims, err := service.VerifyIdToken(tokenString, false)
		require.NoError(t, err, "Failed to verify generated ID token with key")

		// Check the claims
		subject, ok := claims.Subject()
		_ = assert.True(t, ok, "User ID not found in token") &&
			assert.Equal(t, "ecdsauser456", subject, "Token subject should match user ID")
		issuer, ok := claims.Issuer()
		_ = assert.True(t, ok, "Issuer not found in token") &&
			assert.Equal(t, common.EnvConfig.AppURL, issuer, "Issuer should match app URL")

		// Verify the key type is EC
		publicKey, err := service.GetPublicJWK()
		require.NoError(t, err)
		assert.Equal(t, jwa.EC().String(), publicKey.KeyType().String(), "Key type should be EC")

		// Verify the algorithm is ES256
		alg, ok := publicKey.Algorithm()
		require.True(t, ok)
		assert.Equal(t, jwa.ES256().String(), alg.String(), "Algorithm should be ES256")
	})

	t.Run("works with RSA-4096 keys", func(t *testing.T) {
		// Create a temporary directory for the test
		tempDir := t.TempDir()

		// Create an RSA-4096 key and save it as JWK
		origKeyID := createRSA4096KeyJWK(t, tempDir)

		// Create a JWT service that loads the key
		service := &JwtService{}
		err := service.init(mockConfig, tempDir)
		require.NoError(t, err, "Failed to initialize JWT service")

		// Verify it loaded the right key
		loadedKeyID, ok := service.privateKey.KeyID()
		require.True(t, ok)
		assert.Equal(t, origKeyID, loadedKeyID, "Loaded key should have the same ID as the original")

		// Create test claims
		userClaims := map[string]interface{}{
			"sub":   "rsauser456",
			"name":  "RSA User",
			"email": "rsauser@example.com",
		}
		const clientID = "rsa-client-123"

		// Generate a token
		tokenString, err := service.GenerateIDToken(userClaims, clientID, "")
		require.NoError(t, err, "Failed to generate ID token with key")
		assert.NotEmpty(t, tokenString, "Token should not be empty")

		// Verify the token
		claims, err := service.VerifyIdToken(tokenString, false)
		require.NoError(t, err, "Failed to verify generated ID token with key")

		// Check the claims
		subject, ok := claims.Subject()
		_ = assert.True(t, ok, "User ID not found in token") &&
			assert.Equal(t, "rsauser456", subject, "Token subject should match user ID")
		issuer, ok := claims.Issuer()
		_ = assert.True(t, ok, "Issuer not found in token") &&
			assert.Equal(t, common.EnvConfig.AppURL, issuer, "Issuer should match app URL")

		// Verify the key type is RSA
		publicKey, err := service.GetPublicJWK()
		require.NoError(t, err)
		assert.Equal(t, jwa.RSA().String(), publicKey.KeyType().String(), "Key type should be RSA")

		// Verify the algorithm is RS256
		alg, ok := publicKey.Algorithm()
		require.True(t, ok)
		assert.Equal(t, jwa.RS256().String(), alg.String(), "Algorithm should be RS256")
	})
}

func TestGenerateVerifyOauthAccessToken(t *testing.T) {
	// Create a temporary directory for the test
	tempDir := t.TempDir()

	// Initialize the JWT service with a mock AppConfigService
	mockConfig := NewTestAppConfigService(&model.AppConfig{
		SessionDuration: model.AppConfigVariable{Value: "60"}, // 60 minutes
	})

	// Setup the environment variable required by the token verification
	originalAppURL := common.EnvConfig.AppURL
	common.EnvConfig.AppURL = "https://test.example.com"
	defer func() {
		common.EnvConfig.AppURL = originalAppURL
	}()

	t.Run("generates and verifies OAuth access token with standard claims", func(t *testing.T) {
		// Create a JWT service
		service := &JwtService{}
		err := service.init(mockConfig, tempDir)
		require.NoError(t, err, "Failed to initialize JWT service")

		// Create a test user
		user := model.User{
			Base: model.Base{
				ID: "user123",
			},
			Email: "user@example.com",
		}
		const clientID = "test-client-123"

		// Generate a token
		tokenString, err := service.GenerateOauthAccessToken(user, clientID)
		require.NoError(t, err, "Failed to generate OAuth access token")
		assert.NotEmpty(t, tokenString, "Token should not be empty")

		// Verify the token
		claims, err := service.VerifyOauthAccessToken(tokenString)
		require.NoError(t, err, "Failed to verify generated OAuth access token")

		// Check the claims
		subject, ok := claims.Subject()
		_ = assert.True(t, ok, "User ID not found in token") &&
			assert.Equal(t, user.ID, subject, "Token subject should match user ID")
		audience, ok := claims.Audience()
		_ = assert.True(t, ok, "Audience not found in token") &&
			assert.Equal(t, []string{clientID}, audience, "Audience should contain the client ID")
		issuer, ok := claims.Issuer()
		_ = assert.True(t, ok, "Issuer not found in token") &&
			assert.Equal(t, common.EnvConfig.AppURL, issuer, "Issuer should match app URL")

		// Check token expiration time is approximately 1 hour from now
		expectedExp := time.Now().Add(1 * time.Hour)
		expiration, ok := claims.Expiration()
		assert.True(t, ok, "Expiration not found in token")
		timeDiff := expectedExp.Sub(expiration).Minutes()
		assert.InDelta(t, 0, timeDiff, 1.0, "Token should expire in approximately 1 hour")
	})

	t.Run("fails verification for expired token", func(t *testing.T) {
		// Create a JWT service with a mock function to generate an expired token
		service := &JwtService{}
		err := service.init(mockConfig, tempDir)
		require.NoError(t, err, "Failed to initialize JWT service")

		// Create a test user
		user := model.User{
			Base: model.Base{
				ID: "user456",
			},
		}
		const clientID = "test-client-456"

		// Generate a token using JWT directly to create an expired token
		token, err := jwt.NewBuilder().
			Subject(user.ID).
			Expiration(time.Now().Add(-1 * time.Hour)). // Expired 1 hour ago
			IssuedAt(time.Now().Add(-2 * time.Hour)).
			Audience([]string{clientID}).
			Issuer(common.EnvConfig.AppURL).
			Build()
		require.NoError(t, err, "Failed to build token")

		err = SetTokenType(token, OAuthAccessTokenJWTType)
		require.NoError(t, err, "Failed to set token type")

		signed, err := jwt.Sign(token, jwt.WithKey(jwa.RS256(), service.privateKey))
		require.NoError(t, err, "Failed to sign token")

		// Verify should fail due to expiration
		_, err = service.VerifyOauthAccessToken(string(signed))
		require.Error(t, err, "Verification should fail with expired token")
		assert.Contains(t, err.Error(), `"exp" not satisfied`, "Error message should indicate token verification failure")
	})

	t.Run("fails verification with invalid signature", func(t *testing.T) {
		// Create two JWT services with different keys
		service1 := &JwtService{}
		err := service1.init(mockConfig, t.TempDir()) // Use a different temp dir
		require.NoError(t, err, "Failed to initialize first JWT service")

		service2 := &JwtService{}
		err = service2.init(mockConfig, t.TempDir()) // Use a different temp dir
		require.NoError(t, err, "Failed to initialize second JWT service")

		// Create a test user
		user := model.User{
			Base: model.Base{
				ID: "user789",
			},
		}
		const clientID = "test-client-789"

		// Generate a token with the first service
		tokenString, err := service1.GenerateOauthAccessToken(user, clientID)
		require.NoError(t, err, "Failed to generate OAuth access token")

		// Verify with the second service should fail due to different keys
		_, err = service2.VerifyOauthAccessToken(tokenString)
		require.Error(t, err, "Verification should fail with invalid signature")
		assert.Contains(t, err.Error(), "verification error", "Error message should indicate token verification failure")
	})

	t.Run("works with Ed25519 keys", func(t *testing.T) {
		// Create a temporary directory for the test
		tempDir := t.TempDir()

		// Create an Ed25519 key and save it as JWK
		origKeyID := createEdDSAKeyJWK(t, tempDir)

		// Create a JWT service that loads the key
		service := &JwtService{}
		err := service.init(mockConfig, tempDir)
		require.NoError(t, err, "Failed to initialize JWT service")

		// Verify it loaded the right key
		loadedKeyID, ok := service.privateKey.KeyID()
		require.True(t, ok)
		assert.Equal(t, origKeyID, loadedKeyID, "Loaded key should have the same ID as the original")

		// Create a test user
		user := model.User{
			Base: model.Base{
				ID: "eddsauser789",
			},
			Email: "eddsaoauth@example.com",
		}
		const clientID = "eddsa-oauth-client"

		// Generate a token
		tokenString, err := service.GenerateOauthAccessToken(user, clientID)
		require.NoError(t, err, "Failed to generate OAuth access token with key")
		assert.NotEmpty(t, tokenString, "Token should not be empty")

		// Verify the token
		claims, err := service.VerifyOauthAccessToken(tokenString)
		require.NoError(t, err, "Failed to verify generated OAuth access token with key")

		// Check the claims
		subject, ok := claims.Subject()
		_ = assert.True(t, ok, "User ID not found in token") &&
			assert.Equal(t, user.ID, subject, "Token subject should match user ID")
		audience, ok := claims.Audience()
		_ = assert.True(t, ok, "Audience not found in token") &&
			assert.Equal(t, []string{clientID}, audience, "Audience should contain the client ID")

		// Verify the key type is OKP
		publicKey, err := service.GetPublicJWK()
		require.NoError(t, err)
		assert.Equal(t, jwa.OKP().String(), publicKey.KeyType().String(), "Key type should be OKP")

		// Verify the algorithm is EdDSA
		alg, ok := publicKey.Algorithm()
		require.True(t, ok)
		assert.Equal(t, jwa.EdDSA().String(), alg.String(), "Algorithm should be EdDSA")
	})

	t.Run("works with ECDSA keys", func(t *testing.T) {
		// Create a temporary directory for the test
		tempDir := t.TempDir()

		// Create an ECDSA key and save it as JWK
		origKeyID := createECDSAKeyJWK(t, tempDir)

		// Create a JWT service that loads the key
		service := &JwtService{}
		err := service.init(mockConfig, tempDir)
		require.NoError(t, err, "Failed to initialize JWT service")

		// Verify it loaded the right key
		loadedKeyID, ok := service.privateKey.KeyID()
		require.True(t, ok)
		assert.Equal(t, origKeyID, loadedKeyID, "Loaded key should have the same ID as the original")

		// Create a test user
		user := model.User{
			Base: model.Base{
				ID: "ecdsauser789",
			},
			Email: "ecdsaoauth@example.com",
		}
		const clientID = "ecdsa-oauth-client"

		// Generate a token
		tokenString, err := service.GenerateOauthAccessToken(user, clientID)
		require.NoError(t, err, "Failed to generate OAuth access token with key")
		assert.NotEmpty(t, tokenString, "Token should not be empty")

		// Verify the token
		claims, err := service.VerifyOauthAccessToken(tokenString)
		require.NoError(t, err, "Failed to verify generated OAuth access token with key")

		// Check the claims
		subject, ok := claims.Subject()
		_ = assert.True(t, ok, "User ID not found in token") &&
			assert.Equal(t, user.ID, subject, "Token subject should match user ID")
		audience, ok := claims.Audience()
		_ = assert.True(t, ok, "Audience not found in token") &&
			assert.Equal(t, []string{clientID}, audience, "Audience should contain the client ID")

		// Verify the key type is EC
		publicKey, err := service.GetPublicJWK()
		require.NoError(t, err)
		assert.Equal(t, jwa.EC().String(), publicKey.KeyType().String(), "Key type should be EC")

		// Verify the algorithm is ES256
		alg, ok := publicKey.Algorithm()
		require.True(t, ok)
		assert.Equal(t, jwa.ES256().String(), alg.String(), "Algorithm should be ES256")
	})

	t.Run("works with RSA-4096 keys", func(t *testing.T) {
		// Create a temporary directory for the test
		tempDir := t.TempDir()

		// Create an RSA-4096 key and save it as JWK
		origKeyID := createRSA4096KeyJWK(t, tempDir)

		// Create a JWT service that loads the key
		service := &JwtService{}
		err := service.init(mockConfig, tempDir)
		require.NoError(t, err, "Failed to initialize JWT service")

		// Verify it loaded the right key
		loadedKeyID, ok := service.privateKey.KeyID()
		require.True(t, ok)
		assert.Equal(t, origKeyID, loadedKeyID, "Loaded key should have the same ID as the original")

		// Create a test user
		user := model.User{
			Base: model.Base{
				ID: "rsauser789",
			},
			Email: "rsaoauth@example.com",
		}
		const clientID = "rsa-oauth-client"

		// Generate a token
		tokenString, err := service.GenerateOauthAccessToken(user, clientID)
		require.NoError(t, err, "Failed to generate OAuth access token with key")
		assert.NotEmpty(t, tokenString, "Token should not be empty")

		// Verify the token
		claims, err := service.VerifyOauthAccessToken(tokenString)
		require.NoError(t, err, "Failed to verify generated OAuth access token with key")

		// Check the claims
		subject, ok := claims.Subject()
		_ = assert.True(t, ok, "User ID not found in token") &&
			assert.Equal(t, user.ID, subject, "Token subject should match user ID")
		audience, ok := claims.Audience()
		_ = assert.True(t, ok, "Audience not found in token") &&
			assert.Equal(t, []string{clientID}, audience, "Audience should contain the client ID")

		// Verify the key type is RSA
		publicKey, err := service.GetPublicJWK()
		require.NoError(t, err)
		assert.Equal(t, jwa.RSA().String(), publicKey.KeyType().String(), "Key type should be RSA")

		// Verify the algorithm is RS256
		alg, ok := publicKey.Algorithm()
		require.True(t, ok)
		assert.Equal(t, jwa.RS256().String(), alg.String(), "Algorithm should be RS256")
	})
}

func TestTokenTypeValidator(t *testing.T) {
	// Create a context for the validator function
	ctx := context.Background()

	t.Run("succeeds when token type matches expected type", func(t *testing.T) {
		// Create a token with the expected type
		token := jwt.New()
		err := token.Set(TokenTypeClaim, AccessTokenJWTType)
		require.NoError(t, err, "Failed to set token type claim")

		// Create a validator function for the expected type
		validator := TokenTypeValidator(AccessTokenJWTType)

		// Validate the token
		err = validator(ctx, token)
		assert.NoError(t, err, "Validator should accept token with matching type")
	})

	t.Run("fails when token type doesn't match expected type", func(t *testing.T) {
		// Create a token with a different type
		token := jwt.New()
		err := token.Set(TokenTypeClaim, OAuthAccessTokenJWTType)
		require.NoError(t, err, "Failed to set token type claim")

		// Create a validator function for a different expected type
		validator := TokenTypeValidator(IDTokenJWTType)

		// Validate the token
		err = validator(ctx, token)
		require.Error(t, err, "Validator should reject token with non-matching type")
		assert.Contains(t, err.Error(), "invalid token type: expected id-token, got oauth-access-token")
	})

	t.Run("fails when token type claim is missing", func(t *testing.T) {
		// Create a token without a type claim
		token := jwt.New()

		// Create a validator function
		validator := TokenTypeValidator(AccessTokenJWTType)

		// Validate the token
		err := validator(ctx, token)
		require.Error(t, err, "Validator should reject token without type claim")
		assert.Contains(t, err.Error(), "failed to get token type claim")
	})

}

func importKey(t *testing.T, privateKeyRaw any, path string) string {
	t.Helper()

	privateKey, err := utils.ImportRawKey(privateKeyRaw)
	require.NoError(t, err, "Failed to import private key")

	err = SaveKeyJWK(privateKey, filepath.Join(path, PrivateKeyFile))
	require.NoError(t, err, "Failed to save key")

	kid, _ := privateKey.KeyID()
	require.NotEmpty(t, kid, "Key ID must be set")

	return kid
}

// Because generating a RSA-406 key isn't immediate, we pre-compute one
var (
	rsaKeyPrecomputed    *rsa.PrivateKey
	rsaKeyPrecomputeOnce sync.Once
)

func createRSA4096KeyJWK(t *testing.T, path string) string {
	t.Helper()

	rsaKeyPrecomputeOnce.Do(func() {
		var err error
		rsaKeyPrecomputed, err = rsa.GenerateKey(rand.Reader, 4096)
		if err != nil {
			panic("failed to precompute RSA key: " + err.Error())
		}
	})

	// Import as JWK and save to disk
	return importKey(t, rsaKeyPrecomputed, path)
}

func createECDSAKeyJWK(t *testing.T, path string) string {
	t.Helper()

	// Generate a new P-256 ECDSA key
	privateKeyRaw, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err, "Failed to generate ECDSA key")

	// Import as JWK and save to disk
	return importKey(t, privateKeyRaw, path)
}

// Helper function to create an Ed25519 key and save it as JWK
func createEdDSAKeyJWK(t *testing.T, path string) string {
	t.Helper()

	// Generate a new Ed25519 key pair
	_, privateKeyRaw, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err, "Failed to generate Ed25519 key")

	// Import as JWK and save to disk
	return importKey(t, privateKeyRaw, path)
}
