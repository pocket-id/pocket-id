package service

import (
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"sync"
	"testing"
	"time"

	"github.com/lestrrat-go/jwx/v3/jwa"
	"github.com/lestrrat-go/jwx/v3/jwk"
	"github.com/lestrrat-go/jwx/v3/jwt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/appconfig"
	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/instanceid"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	jwkutils "github.com/pocket-id/pocket-id/backend/internal/utils/jwk"
	testutils "github.com/pocket-id/pocket-id/backend/internal/utils/testing"
)

const testEncryptionKey = "0123456789abcdef0123456789abcdef"

const uuidRegexPattern = "^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$"

func newTestEnvConfig() *common.EnvConfigSchema {
	return &common.EnvConfigSchema{
		AppURL:        "https://test.example.com",
		EncryptionKey: []byte(testEncryptionKey),
	}
}

func initJwtService(t *testing.T, db *gorm.DB, instanceID string, appConfig *appconfig.AppConfigService, envConfig *common.EnvConfigSchema) *JwtService {
	t.Helper()

	service := &JwtService{}
	err := service.init(t.Context(), db, instanceID, appConfig, envConfig)
	require.NoError(t, err, "Failed to initialize JWT service")

	return service
}

func setupJwtService(t *testing.T, instanceID string, appConfig *appconfig.AppConfigService) (*JwtService, *gorm.DB, *common.EnvConfigSchema) {
	t.Helper()

	db := testutils.NewDatabaseForTest(t)
	envConfig := newTestEnvConfig()

	service := initJwtService(t, db, instanceID, appConfig, envConfig)
	return service, db, envConfig
}

func newInstanceID(t *testing.T, db *gorm.DB) string {
	t.Helper()

	instanceID, err := instanceid.Load(t.Context(), db)
	require.NoError(t, err)

	return instanceID
}

func newTestDbAndEnv(t *testing.T) (*gorm.DB, *common.EnvConfigSchema) {
	t.Helper()

	return testutils.NewDatabaseForTest(t), newTestEnvConfig()
}

func saveKeyToDatabase(t *testing.T, db *gorm.DB, instanceID string, envConfig *common.EnvConfigSchema, appConfig *appconfig.AppConfigService, key jwk.Key) string {
	t.Helper()

	keyProvider, err := jwkutils.GetKeyProvider(db, envConfig, instanceID)
	require.NoError(t, err, "Failed to init key provider")

	err = keyProvider.SaveKey(t.Context(), key)
	require.NoError(t, err, "Failed to save key")

	kid, ok := key.KeyID()
	require.True(t, ok, "Key ID must be set")
	require.NotEmpty(t, kid, "Key ID must not be empty")

	return kid
}

func TestJwtService_Init(t *testing.T) {
	mockConfig := appconfig.NewTestAppConfigService(&model.AppConfig{
		SessionDuration: model.AppConfigVariable{Value: "60"}, // 60 minutes
	})

	t.Run("should generate new key when none exists", func(t *testing.T) {
		db := testutils.NewDatabaseForTest(t)
		mockEnvConfig := newTestEnvConfig()
		instanceID := newInstanceID(t, db)

		// Initialize the JWT service
		service := initJwtService(t, db, instanceID, mockConfig, mockEnvConfig)

		// Verify the private key was set
		require.NotNil(t, service.privateKey, "Private key should be set")

		// Verify the key has been persisted in the database
		keyProvider, err := jwkutils.GetKeyProvider(db, mockEnvConfig, instanceID)
		require.NoError(t, err, "Failed to init key provider")
		key, err := keyProvider.LoadKey(t.Context())
		require.NoError(t, err, "Failed to load key from provider")
		require.NotNil(t, key, "Key should be present in the database")

		// Key should have required properties
		keyID, ok := key.KeyID()
		assert.True(t, ok, "Key should have a key ID")
		assert.NotEmpty(t, keyID)

		keyUsage, ok := key.KeyUsage()
		assert.True(t, ok, "Key should have a key usage")
		assert.Equal(t, KeyUsageSigning, keyUsage)
	})

	t.Run("should load existing JWK key", func(t *testing.T) {
		db := testutils.NewDatabaseForTest(t)
		mockEnvConfig := newTestEnvConfig()
		instanceID := newInstanceID(t, db)

		// First create a service to generate a key
		firstService := initJwtService(t, db, instanceID, mockConfig, mockEnvConfig)

		// Get the key ID of the first service
		origKeyID, ok := firstService.privateKey.KeyID()
		require.True(t, ok)

		// Now create a new service that should load the existing key
		secondService := initJwtService(t, db, instanceID, mockConfig, mockEnvConfig)

		// Verify the loaded key has the same ID as the original
		loadedKeyID, ok := secondService.privateKey.KeyID()
		require.True(t, ok)
		assert.Equal(t, origKeyID, loadedKeyID, "Loaded key should have the same ID as the original")
	})

	t.Run("should load existing JWK for ECDSA keys", func(t *testing.T) {
		db := testutils.NewDatabaseForTest(t)
		mockEnvConfig := newTestEnvConfig()
		instanceID := newInstanceID(t, db)

		// Create a new JWK and save it to the database
		origKeyID := createECDSAKeyJWK(t, db, instanceID, mockEnvConfig, mockConfig)

		// Now create a new service that should load the existing key
		svc := initJwtService(t, db, instanceID, mockConfig, mockEnvConfig)

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
		db := testutils.NewDatabaseForTest(t)
		mockEnvConfig := newTestEnvConfig()
		instanceID := newInstanceID(t, db)

		// Create a new JWK and save it to the database
		origKeyID := createEdDSAKeyJWK(t, db, instanceID, mockEnvConfig, mockConfig)

		// Now create a new service that should load the existing key
		svc := initJwtService(t, db, instanceID, mockConfig, mockEnvConfig)

		// Ensure loaded key has the right algorithm and curve
		alg, ok := svc.privateKey.Algorithm()
		_ = assert.True(t, ok) &&
			assert.Equal(t, jwa.EdDSA().String(), alg.String(), "Loaded key has the incorrect algorithm")

		var curve jwa.EllipticCurveAlgorithm
		err := svc.privateKey.Get("crv", &curve)
		_ = assert.NoError(t, err, "Failed to get 'crv' claim") &&
			assert.Equal(t, jwa.Ed25519().String(), curve.String(), "Curve does not match expected value")

		// Verify the loaded key has the same ID as the original
		loadedKeyID, ok := svc.privateKey.KeyID()
		_ = assert.True(t, ok) &&
			assert.Equal(t, origKeyID, loadedKeyID, "Loaded key should have the same ID as the original")
	})

}

func TestJwtService_GetPublicJWK(t *testing.T) {
	mockConfig := appconfig.NewTestAppConfigService(&model.AppConfig{
		SessionDuration: model.AppConfigVariable{Value: "60"}, // 60 minutes
	})
	db := testutils.NewDatabaseForTest(t)
	mockEnvConfig := newTestEnvConfig()
	instanceID := newInstanceID(t, db)

	t.Run("returns public key when private key is initialized", func(t *testing.T) {
		service, _, _ := setupJwtService(t, instanceID, mockConfig)

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
		// Create an ECDSA key and save it in the database
		originalKeyID := createECDSAKeyJWK(t, db, instanceID, mockEnvConfig, mockConfig)

		// Create a JWT service that loads the ECDSA key
		service := initJwtService(t, db, instanceID, mockConfig, mockEnvConfig)

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
		db := testutils.NewDatabaseForTest(t)
		mockEnvConfig := newTestEnvConfig()

		// Create an EdDSA key and save it in the database
		originalKeyID := createEdDSAKeyJWK(t, db, instanceID, mockEnvConfig, mockConfig)

		// Create a JWT service that loads the EdDSA key
		service := initJwtService(t, db, instanceID, mockConfig, mockEnvConfig)

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
	mockConfig := appconfig.NewTestAppConfigService(&model.AppConfig{
		SessionDuration: model.AppConfigVariable{Value: "60"}, // 60 minutes
	})
	db, envConfig := newTestDbAndEnv(t)
	instanceID := newInstanceID(t, db)

	t.Run("generates token for regular user", func(t *testing.T) {
		service, _, _ := setupJwtService(t, instanceID, mockConfig)

		user := model.User{
			Base:    model.Base{ID: "user123"},
			Email:   new("user@example.com"),
			IsAdmin: false,
		}

		tokenString, err := service.GenerateAccessToken(user, "")
		require.NoError(t, err, "Failed to generate access token")
		assert.NotEmpty(t, tokenString, "Token should not be empty")

		claims, err := service.VerifyAccessToken(tokenString)
		require.NoError(t, err, "Failed to verify generated token")

		subject, ok := claims.Subject()
		_ = assert.True(t, ok, "User ID not found in token") &&
			assert.Equal(t, user.ID, subject, "Token subject should match user ID")
		isAdmin := false
		if claims.Has(IsAdminClaim) {
			require.NoError(t, claims.Get(IsAdminClaim, &isAdmin), "Failed to get isAdmin claim")
		}
		assert.False(t, isAdmin, "isAdmin should be false")
		authenticationMethod, err := service.GetAuthenticationMethod(claims)
		_ = assert.NoError(t, err, "Failed to get amr claim") &&
			assert.Empty(t, authenticationMethod, "amr should be empty when not specified")
		audience, ok := claims.Audience()
		_ = assert.True(t, ok, "Audience not found in token") &&
			assert.Equal(t, []string{service.envConfig.AppURL}, audience, "Audience should contain the app URL")
		jwtID, ok := claims.JwtID()
		_ = assert.True(t, ok, "JWT ID not found in token") &&
			assert.Regexp(t, uuidRegexPattern, jwtID, "JWT ID is not a UUID")

		expectedExp := time.Now().Add(1 * time.Hour)
		expiration, ok := claims.Expiration()
		assert.True(t, ok, "Expiration not found in token")
		timeDiff := expectedExp.Sub(expiration).Minutes()
		assert.InDelta(t, 0, timeDiff, 1.0, "Token should expire in approximately 1 hour")
	})

	t.Run("generates token for admin user", func(t *testing.T) {
		service, _, _ := setupJwtService(t, instanceID, mockConfig)

		adminUser := model.User{
			Base:    model.Base{ID: "admin123"},
			Email:   new("admin@example.com"),
			IsAdmin: true,
		}

		tokenString, err := service.GenerateAccessToken(adminUser, "")
		require.NoError(t, err, "Failed to generate access token")

		claims, err := service.VerifyAccessToken(tokenString)
		require.NoError(t, err, "Failed to verify generated token")

		isAdmin := false
		if claims.Has(IsAdminClaim) {
			require.NoError(t, claims.Get(IsAdminClaim, &isAdmin), "Failed to get isAdmin claim")
		}
		assert.True(t, isAdmin, "isAdmin should be true")
		subject, ok := claims.Subject()
		_ = assert.True(t, ok, "User ID not found in token") &&
			assert.Equal(t, adminUser.ID, subject, "Token subject should match user ID")
	})

	t.Run("sets authentication method references claim when provided", func(t *testing.T) {
		service, _, _ := setupJwtService(t, instanceID, mockConfig)

		user := model.User{
			Base: model.Base{ID: "user-with-auth-method"},
		}

		tokenString, err := service.GenerateAccessToken(user, AuthenticationMethodPhishingResistant)
		require.NoError(t, err, "Failed to generate access token")

		claims, err := service.VerifyAccessToken(tokenString)
		require.NoError(t, err, "Failed to verify generated token")

		authenticationMethod, err := service.GetAuthenticationMethod(claims)
		_ = assert.NoError(t, err, "Failed to get amr claim") &&
			assert.Equal(t, AuthenticationMethodPhishingResistant, authenticationMethod, "amr should match")
	})

	t.Run("uses session duration from config", func(t *testing.T) {
		customMockConfig := appconfig.NewTestAppConfigService(&model.AppConfig{
			SessionDuration: model.AppConfigVariable{Value: "30"}, // 30 minutes
		})
		service, _, _ := setupJwtService(t, instanceID, customMockConfig)

		user := model.User{
			Base: model.Base{ID: "user456"},
		}

		tokenString, err := service.GenerateAccessToken(user, "")
		require.NoError(t, err, "Failed to generate access token")

		claims, err := service.VerifyAccessToken(tokenString)
		require.NoError(t, err, "Failed to verify generated token")

		expectedExp := time.Now().Add(30 * time.Minute)
		expiration, ok := claims.Expiration()
		assert.True(t, ok, "Expiration not found in token")
		timeDiff := expectedExp.Sub(expiration).Minutes()
		assert.InDelta(t, 0, timeDiff, 1.0, "Token should expire in approximately 30 minutes")
	})

	t.Run("works with Ed25519 keys", func(t *testing.T) {
		origKeyID := createEdDSAKeyJWK(t, db, instanceID, envConfig, mockConfig)
		service := initJwtService(t, db, instanceID, mockConfig, envConfig)

		loadedKeyID, ok := service.privateKey.KeyID()
		require.True(t, ok)
		assert.Equal(t, origKeyID, loadedKeyID, "Loaded key should have the same ID as the original")

		user := model.User{
			Base:    model.Base{ID: "eddsauser123"},
			Email:   new("eddsauser@example.com"),
			IsAdmin: true,
		}

		tokenString, err := service.GenerateAccessToken(user, "")
		require.NoError(t, err, "Failed to generate access token with Ed25519 key")
		assert.NotEmpty(t, tokenString, "Token should not be empty")

		claims, err := service.VerifyAccessToken(tokenString)
		require.NoError(t, err, "Failed to verify generated token with Ed25519 key")

		subject, ok := claims.Subject()
		_ = assert.True(t, ok, "User ID not found in token") &&
			assert.Equal(t, user.ID, subject, "Token subject should match user ID")
		isAdmin := false
		if claims.Has(IsAdminClaim) {
			require.NoError(t, claims.Get(IsAdminClaim, &isAdmin), "Failed to get isAdmin claim")
		}
		assert.True(t, isAdmin, "isAdmin should be true")

		publicKey, err := service.GetPublicJWK()
		require.NoError(t, err)
		assert.Equal(t, "OKP", publicKey.KeyType().String(), "Key type should be OKP")
		alg, ok := publicKey.Algorithm()
		require.True(t, ok)
		assert.Equal(t, "EdDSA", alg.String(), "Algorithm should be EdDSA")
	})

	t.Run("works with P-256 keys", func(t *testing.T) {
		origKeyID := createECDSAKeyJWK(t, db, instanceID, envConfig, mockConfig)
		service := initJwtService(t, db, instanceID, mockConfig, envConfig)

		loadedKeyID, ok := service.privateKey.KeyID()
		require.True(t, ok)
		assert.Equal(t, origKeyID, loadedKeyID, "Loaded key should have the same ID as the original")

		user := model.User{
			Base:    model.Base{ID: "ecdsauser123"},
			Email:   new("ecdsauser@example.com"),
			IsAdmin: true,
		}

		tokenString, err := service.GenerateAccessToken(user, "")
		require.NoError(t, err, "Failed to generate access token with ECDSA key")
		assert.NotEmpty(t, tokenString, "Token should not be empty")

		claims, err := service.VerifyAccessToken(tokenString)
		require.NoError(t, err, "Failed to verify generated token with ECDSA key")

		subject, ok := claims.Subject()
		_ = assert.True(t, ok, "User ID not found in token") &&
			assert.Equal(t, user.ID, subject, "Token subject should match user ID")
		isAdmin := false
		if claims.Has(IsAdminClaim) {
			require.NoError(t, claims.Get(IsAdminClaim, &isAdmin), "Failed to get isAdmin claim")
		}
		assert.True(t, isAdmin, "isAdmin should be true")

		publicKey, err := service.GetPublicJWK()
		require.NoError(t, err)
		assert.Equal(t, "EC", publicKey.KeyType().String(), "Key type should be EC")
		alg, ok := publicKey.Algorithm()
		require.True(t, ok)
		assert.Equal(t, "ES256", alg.String(), "Algorithm should be ES256")
	})

	t.Run("works with RSA-4096 keys", func(t *testing.T) {
		origKeyID := createRSA4096KeyJWK(t, db, instanceID, envConfig, mockConfig)
		service := initJwtService(t, db, instanceID, mockConfig, envConfig)

		loadedKeyID, ok := service.privateKey.KeyID()
		require.True(t, ok)
		assert.Equal(t, origKeyID, loadedKeyID, "Loaded key should have the same ID as the original")

		user := model.User{
			Base:    model.Base{ID: "rsauser123"},
			Email:   new("rsauser@example.com"),
			IsAdmin: true,
		}

		tokenString, err := service.GenerateAccessToken(user, "")
		require.NoError(t, err, "Failed to generate access token with RSA key")
		assert.NotEmpty(t, tokenString, "Token should not be empty")

		claims, err := service.VerifyAccessToken(tokenString)
		require.NoError(t, err, "Failed to verify generated token with RSA key")

		subject, ok := claims.Subject()
		_ = assert.True(t, ok, "User ID not found in token") &&
			assert.Equal(t, user.ID, subject, "Token subject should match user ID")
		isAdmin := false
		if claims.Has(IsAdminClaim) {
			require.NoError(t, claims.Get(IsAdminClaim, &isAdmin), "Failed to get isAdmin claim")
		}
		assert.True(t, isAdmin, "isAdmin should be true")

		publicKey, err := service.GetPublicJWK()
		require.NoError(t, err)
		assert.Equal(t, jwa.RSA().String(), publicKey.KeyType().String(), "Key type should be RSA")
		alg, ok := publicKey.Algorithm()
		require.True(t, ok)
		assert.Equal(t, jwa.RS256().String(), alg.String(), "Algorithm should be RS256")
	})
}

func TestTokenTypeValidator(t *testing.T) {
	t.Run("succeeds when token type matches expected type", func(t *testing.T) {
		// Create a token with the expected type
		token := jwt.New()
		err := token.Set(TokenTypeClaim, AccessTokenJWTType)
		require.NoError(t, err, "Failed to set token type claim")

		// Create a validator function for the expected type
		validator := TokenTypeValidator(AccessTokenJWTType)

		// Validate the token
		err = validator(t.Context(), token)
		assert.NoError(t, err, "Validator should accept token with matching type")
	})

	t.Run("fails when token type doesn't match expected type", func(t *testing.T) {
		// Create a token with a different type
		token := jwt.New()
		err := token.Set(TokenTypeClaim, "other-token")
		require.NoError(t, err, "Failed to set token type claim")

		// Create a validator function for a different expected type
		validator := TokenTypeValidator(AccessTokenJWTType)

		// Validate the token
		err = validator(t.Context(), token)
		require.Error(t, err, "Validator should reject token with non-matching type")
		assert.Contains(t, err.Error(), "invalid token type: expected access-token, got other-token")
	})

	t.Run("fails when token type claim is missing", func(t *testing.T) {
		// Create a token without a type claim
		token := jwt.New()

		// Create a validator function
		validator := TokenTypeValidator(AccessTokenJWTType)

		// Validate the token
		err := validator(t.Context(), token)
		require.Error(t, err, "Validator should reject token without type claim")
		assert.Contains(t, err.Error(), "failed to get token type claim")
	})
}

func importKey(t *testing.T, db *gorm.DB, instanceID string, envConfig *common.EnvConfigSchema, appConfig *appconfig.AppConfigService, privateKeyRaw any) string {
	t.Helper()

	privateKey, err := jwkutils.ImportRawKey(privateKeyRaw, "", "")
	require.NoError(t, err, "Failed to import private key")

	return saveKeyToDatabase(t, db, instanceID, envConfig, appConfig, privateKey)
}

// Because generating a RSA-406 key isn't immediate, we pre-compute one
var (
	rsaKeyPrecomputed    *rsa.PrivateKey
	rsaKeyPrecomputeOnce sync.Once
)

func createRSA4096KeyJWK(t *testing.T, db *gorm.DB, instanceID string, envConfig *common.EnvConfigSchema, appConfig *appconfig.AppConfigService) string {
	t.Helper()

	rsaKeyPrecomputeOnce.Do(func() {
		var err error
		rsaKeyPrecomputed, err = rsa.GenerateKey(rand.Reader, 4096)
		if err != nil {
			panic("failed to precompute RSA key: " + err.Error())
		}
	})

	// Import as JWK and save it
	return importKey(t, db, instanceID, envConfig, appConfig, rsaKeyPrecomputed)
}

func createECDSAKeyJWK(t *testing.T, db *gorm.DB, instanceID string, envConfig *common.EnvConfigSchema, appConfig *appconfig.AppConfigService) string {
	t.Helper()

	// Generate a new P-256 ECDSA key
	privateKeyRaw, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err, "Failed to generate ECDSA key")

	// Import as JWK and save it
	return importKey(t, db, instanceID, envConfig, appConfig, privateKeyRaw)
}

// Helper function to create an Ed25519 key and save it as JWK
func createEdDSAKeyJWK(t *testing.T, db *gorm.DB, instanceID string, envConfig *common.EnvConfigSchema, appConfig *appconfig.AppConfigService) string {
	t.Helper()

	// Generate a new Ed25519 key pair
	_, privateKeyRaw, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err, "Failed to generate Ed25519 key")

	// Import as JWK and save it
	return importKey(t, db, instanceID, envConfig, appConfig, privateKeyRaw)
}
