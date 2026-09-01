package service

import (
	"context"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/lestrrat-go/jwx/v4/jwa"
	"github.com/lestrrat-go/jwx/v4/jwk"
	"github.com/lestrrat-go/jwx/v4/jws"
	"github.com/lestrrat-go/jwx/v4/jwt"
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

func initJwtService(t *testing.T, db *gorm.DB, instanceID string, _ *appconfig.AppConfigService, envConfig *common.EnvConfigSchema) *JwtService {
	t.Helper()

	service := &JwtService{}
	err := service.init(t.Context(), db, instanceID, envConfig)
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

	err = keyProvider.ReplaceKey(t.Context(), key)
	require.NoError(t, err, "Failed to save key")

	kid, ok := key.KeyID()
	require.True(t, ok, "Key ID must be set")
	require.NotEmpty(t, kid, "Key ID must not be empty")

	return kid
}

func TestJwtService_Init(t *testing.T) {
	mockConfig := appconfig.NewTestAppConfigService(nil)

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

		curve, err := jwk.Get[jwa.EllipticCurveAlgorithm](svc.privateKey, "crv")
		_ = assert.NoError(t, err, "Failed to get 'crv' claim") &&
			assert.Equal(t, jwa.Ed25519().String(), curve.String(), "Curve does not match expected value")

		// Verify the loaded key has the same ID as the original
		loadedKeyID, ok := svc.privateKey.KeyID()
		_ = assert.True(t, ok) &&
			assert.Equal(t, origKeyID, loadedKeyID, "Loaded key should have the same ID as the original")
	})

	for _, dbKey := range []string{jwkutils.PrivateKeyDBKey, jwkutils.SessionKeyDBKey} {
		t.Run("should not retry a failed database write for "+dbKey, func(t *testing.T) {
			// Configure the database to fail the first attempt for the selected key
			db := testutils.NewDatabaseForTest(t)
			mockEnvConfig := newTestEnvConfig()
			instanceID := newInstanceID(t, db)
			storeAttempts := 0
			storeErr := errors.New("test database error")

			err := db.Callback().Create().Before("gorm:create").Register("fail_first_key_storage", func(tx *gorm.DB) {
				row, ok := tx.Statement.Dest.(*model.KV)
				if !ok || row.Key != dbKey {
					return
				}

				storeAttempts++
				if storeAttempts == 1 {
					_ = tx.AddError(storeErr)
				}
			})
			require.NoError(t, err)

			// Initialize the service and preserve the ordinary database failure
			service := &JwtService{}
			err = service.init(t.Context(), db, instanceID, mockEnvConfig)
			require.ErrorIs(t, err, storeErr)

			// Verify the failed write was not retried
			require.Equal(t, 1, storeAttempts)
		})
	}

}

func TestRetryKeyStorageStopsAfterThreeRetries(t *testing.T) {
	// Return a conflict on every attempt so the retry limit is reached
	attempts := 0
	err := retryKeyStorage(t.Context(), func(_ context.Context) error {
		attempts++
		return jwkutils.ErrRetryKeyStorage
	})

	// Verify the initial attempt was followed by exactly three retries
	require.ErrorIs(t, err, jwkutils.ErrRetryKeyStorage)
	require.Equal(t, 1+maxKeyStorageRetries, attempts)
}

func TestJwtService_SessionKey(t *testing.T) {
	mockConfig := appconfig.NewTestAppConfigService(nil)

	t.Run("should generate a new session key when none exists", func(t *testing.T) {
		db := testutils.NewDatabaseForTest(t)
		mockEnvConfig := newTestEnvConfig()
		instanceID := newInstanceID(t, db)

		// Initialize the JWT service
		service := initJwtService(t, db, instanceID, mockConfig, mockEnvConfig)

		// Verify the session key was set and is a symmetric HS256 key
		require.NotNil(t, service.sessionKey, "Session key should be set")
		assert.Equal(t, jwa.OctetSeq(), service.sessionKey.KeyType(), "Session key should be a symmetric key")
		alg, ok := service.sessionKey.Algorithm()
		_ = assert.True(t, ok, "Session key should have an algorithm") &&
			assert.Equal(t, jwa.HS256().String(), alg.String(), "Session key should use HS256")

		// Verify the session key has been persisted in its own row in the database
		keyProvider, err := jwkutils.GetSessionKeyProvider(db, mockEnvConfig, instanceID)
		require.NoError(t, err, "Failed to init session key provider")
		key, err := keyProvider.LoadKey(t.Context())
		require.NoError(t, err, "Failed to load session key from provider")
		require.NotNil(t, key, "Session key should be present in the database")

		keyID, ok := key.KeyID()
		_ = assert.True(t, ok, "Session key should have a key ID") &&
			assert.NotEmpty(t, keyID)
	})

	t.Run("should load an existing session key", func(t *testing.T) {
		db := testutils.NewDatabaseForTest(t)
		mockEnvConfig := newTestEnvConfig()
		instanceID := newInstanceID(t, db)

		// First create a service to generate a session key
		firstService := initJwtService(t, db, instanceID, mockConfig, mockEnvConfig)
		origKeyID, ok := firstService.sessionKey.KeyID()
		require.True(t, ok)

		// Now create a new service that should load the existing session key
		secondService := initJwtService(t, db, instanceID, mockConfig, mockEnvConfig)
		loadedKeyID, ok := secondService.sessionKey.KeyID()
		require.True(t, ok)
		assert.Equal(t, origKeyID, loadedKeyID, "Loaded session key should have the same ID as the original")

		// A session token issued by the first service must be accepted by the second one
		tokenString, err := firstService.GenerateAccessToken(model.User{Base: model.Base{ID: "user123"}}, "", time.Hour)
		require.NoError(t, err)
		_, err = secondService.VerifyAccessToken(tokenString)
		require.NoError(t, err, "Session token should be verified by a service that loaded the same session key")
	})

	t.Run("session key is separate from the token signing key", func(t *testing.T) {
		db := testutils.NewDatabaseForTest(t)
		mockEnvConfig := newTestEnvConfig()
		instanceID := newInstanceID(t, db)

		service := initJwtService(t, db, instanceID, mockConfig, mockEnvConfig)

		signingKeyID, ok := service.privateKey.KeyID()
		require.True(t, ok)
		sessionKeyID, ok := service.sessionKey.KeyID()
		require.True(t, ok)
		assert.NotEqual(t, signingKeyID, sessionKeyID, "Session key and token signing key should be different keys")

		// The session key is a shared secret, so it must never be published in the JWKS
		jwks, err := service.GetPublicJWKSAsJSON()
		require.NoError(t, err)
		assert.NotContains(t, string(jwks), sessionKeyID, "Session key must not be included in the JWKS")
		assert.NotContains(t, string(jwks), jwa.OctetSeq().String(), "JWKS must not contain symmetric keys")
	})

	t.Run("session tokens are signed with HS256 and the session key", func(t *testing.T) {
		db := testutils.NewDatabaseForTest(t)
		mockEnvConfig := newTestEnvConfig()
		instanceID := newInstanceID(t, db)

		service := initJwtService(t, db, instanceID, mockConfig, mockEnvConfig)

		tokenString, err := service.GenerateAccessToken(model.User{Base: model.Base{ID: "user123"}}, "", time.Hour)
		require.NoError(t, err)

		// Inspect the JWS header to confirm the algorithm and key used
		msg, err := jws.ParseString(tokenString)
		require.NoError(t, err)
		require.Len(t, msg.Signatures(), 1)

		headers := msg.Signatures()[0].ProtectedHeaders()
		headerAlg, ok := headers.Algorithm()
		_ = assert.True(t, ok, "Session token should declare an algorithm") &&
			assert.Equal(t, jwa.HS256().String(), headerAlg.String(), "Session token should be signed with HS256")

		sessionKeyID, _ := service.sessionKey.KeyID()
		kid, ok := headers.KeyID()
		_ = assert.True(t, ok, "Session token should reference a key ID") &&
			assert.Equal(t, sessionKeyID, kid, "Session token should be signed with the session key")
	})

	t.Run("session tokens signed with a different session key are rejected", func(t *testing.T) {
		db := testutils.NewDatabaseForTest(t)
		mockEnvConfig := newTestEnvConfig()
		instanceID := newInstanceID(t, db)

		service := initJwtService(t, db, instanceID, mockConfig, mockEnvConfig)

		tokenString, err := service.GenerateAccessToken(model.User{Base: model.Base{ID: "user123"}}, "", time.Hour)
		require.NoError(t, err)

		// Rotate the session key, as the key-rotate command does, then reload it
		rotatedKey, err := jwkutils.GenerateSessionKey()
		require.NoError(t, err)
		keyProvider, err := jwkutils.GetSessionKeyProvider(db, mockEnvConfig, instanceID)
		require.NoError(t, err)
		require.NoError(t, keyProvider.ReplaceKey(t.Context(), rotatedKey))
		require.NoError(t, service.LoadOrGenerateKey(t.Context()))

		// Tokens issued with the previous session key must no longer be accepted
		_, err = service.VerifyAccessToken(tokenString)
		require.Error(t, err, "Session token signed with the previous session key should be rejected")
	})

	t.Run("rejects an invalid session key", func(t *testing.T) {
		service := &JwtService{}

		// A key for tokens meant for external consumption is not valid as a session key
		signingKey, err := jwkutils.GenerateKey(jwa.ES256().String(), "")
		require.NoError(t, err)
		err = service.SetSessionKey(signingKey)
		require.Error(t, err, "An asymmetric key should not be accepted as a session key")
		require.ErrorContains(t, err, "not a symmetric key")

		// A symmetric key for another algorithm is not valid either
		rawKey := make([]byte, 32)
		_, err = rand.Read(rawKey)
		require.NoError(t, err)
		otherAlgKey, err := jwkutils.ImportRawKey(rawKey, jwa.HS512().String(), "")
		require.NoError(t, err)
		err = service.SetSessionKey(otherAlgKey)
		require.Error(t, err, "A key for another algorithm should not be accepted as a session key")
		assert.ErrorContains(t, err, "not valid for the HS256 algorithm")
	})

	t.Run("returns an error when the session key is not initialized", func(t *testing.T) {
		service := &JwtService{}

		_, err := service.GenerateAccessToken(model.User{Base: model.Base{ID: "user123"}}, "", time.Hour)
		require.Error(t, err)
		assert.ErrorContains(t, err, "session key is not initialized")

		_, err = service.VerifyAccessToken("some-token")
		require.Error(t, err)
		assert.ErrorContains(t, err, "session key is not initialized")
	})
}

func TestJwtService_GetPublicJWK(t *testing.T) {
	mockConfig := appconfig.NewTestAppConfigService(nil)
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
	const sessionDuration = time.Hour
	mockConfig := appconfig.NewTestAppConfigService(nil)
	db, envConfig := newTestDbAndEnv(t)
	instanceID := newInstanceID(t, db)

	t.Run("generates token for regular user", func(t *testing.T) {
		service, _, _ := setupJwtService(t, instanceID, mockConfig)

		user := model.User{
			Base:    model.Base{ID: "user123"},
			Email:   new("user@example.com"),
			IsAdmin: false,
		}

		tokenString, err := service.GenerateAccessToken(user, "", sessionDuration)
		require.NoError(t, err, "Failed to generate access token")
		assert.NotEmpty(t, tokenString, "Token should not be empty")

		claims, err := service.VerifyAccessToken(tokenString)
		require.NoError(t, err, "Failed to verify generated token")

		subject, ok := claims.Subject()
		_ = assert.True(t, ok, "User ID not found in token") &&
			assert.Equal(t, user.ID, subject, "Token subject should match user ID")
		isAdmin := false
		if claims.Has(IsAdminClaim) {
			isAdmin, err = jwt.Get[bool](claims, IsAdminClaim)
			require.NoError(t, err, "Failed to get isAdmin claim")
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

		tokenString, err := service.GenerateAccessToken(adminUser, "", sessionDuration)
		require.NoError(t, err, "Failed to generate access token")

		claims, err := service.VerifyAccessToken(tokenString)
		require.NoError(t, err, "Failed to verify generated token")

		isAdmin := false
		if claims.Has(IsAdminClaim) {
			isAdmin, err = jwt.Get[bool](claims, IsAdminClaim)
			require.NoError(t, err, "Failed to get isAdmin claim")
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

		tokenString, err := service.GenerateAccessToken(user, AuthenticationMethodPhishingResistant, sessionDuration)
		require.NoError(t, err, "Failed to generate access token")

		claims, err := service.VerifyAccessToken(tokenString)
		require.NoError(t, err, "Failed to verify generated token")

		authenticationMethod, err := service.GetAuthenticationMethod(claims)
		_ = assert.NoError(t, err, "Failed to get amr claim") &&
			assert.Equal(t, AuthenticationMethodPhishingResistant, authenticationMethod, "amr should match")
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

		tokenString, err := service.GenerateAccessToken(user, "", sessionDuration)
		require.NoError(t, err, "Failed to generate access token with Ed25519 key")
		assert.NotEmpty(t, tokenString, "Token should not be empty")

		claims, err := service.VerifyAccessToken(tokenString)
		require.NoError(t, err, "Failed to verify generated token with Ed25519 key")

		subject, ok := claims.Subject()
		_ = assert.True(t, ok, "User ID not found in token") &&
			assert.Equal(t, user.ID, subject, "Token subject should match user ID")
		isAdmin := false
		if claims.Has(IsAdminClaim) {
			isAdmin, err = jwt.Get[bool](claims, IsAdminClaim)
			require.NoError(t, err, "Failed to get isAdmin claim")
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

		tokenString, err := service.GenerateAccessToken(user, "", sessionDuration)
		require.NoError(t, err, "Failed to generate access token with ECDSA key")
		assert.NotEmpty(t, tokenString, "Token should not be empty")

		claims, err := service.VerifyAccessToken(tokenString)
		require.NoError(t, err, "Failed to verify generated token with ECDSA key")

		subject, ok := claims.Subject()
		_ = assert.True(t, ok, "User ID not found in token") &&
			assert.Equal(t, user.ID, subject, "Token subject should match user ID")
		isAdmin := false
		if claims.Has(IsAdminClaim) {
			isAdmin, err = jwt.Get[bool](claims, IsAdminClaim)
			require.NoError(t, err, "Failed to get isAdmin claim")
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

		tokenString, err := service.GenerateAccessToken(user, "", sessionDuration)
		require.NoError(t, err, "Failed to generate access token with RSA key")
		assert.NotEmpty(t, tokenString, "Token should not be empty")

		claims, err := service.VerifyAccessToken(tokenString)
		require.NoError(t, err, "Failed to verify generated token with RSA key")

		subject, ok := claims.Subject()
		_ = assert.True(t, ok, "User ID not found in token") &&
			assert.Equal(t, user.ID, subject, "Token subject should match user ID")
		isAdmin := false
		if claims.Has(IsAdminClaim) {
			isAdmin, err = jwt.Get[bool](claims, IsAdminClaim)
			require.NoError(t, err, "Failed to get isAdmin claim")
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
