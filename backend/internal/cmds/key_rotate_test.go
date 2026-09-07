package cmds

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/instanceid"
	jwkutils "github.com/pocket-id/pocket-id/backend/internal/utils/jwk"
	testingutils "github.com/pocket-id/pocket-id/backend/internal/utils/testing"
)

func TestKeyRotate(t *testing.T) {
	tests := []struct {
		name    string
		flags   keyRotateFlags
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid RS256",
			flags: keyRotateFlags{
				Alg: "RS256",
				Yes: true,
			},
			wantErr: false,
		},
		{
			name: "valid EdDSA with Ed25519",
			flags: keyRotateFlags{
				Alg: "EdDSA",
				Crv: "Ed25519",
				Yes: true,
			},
			wantErr: false,
		},
		{
			name: "invalid algorithm",
			flags: keyRotateFlags{
				Alg: "INVALID",
				Yes: true,
			},
			wantErr: true,
			errMsg:  "unsupported key algorithm",
		},
		{
			name: "EdDSA without curve",
			flags: keyRotateFlags{
				Alg: "EdDSA",
				Yes: true,
			},
			wantErr: true,
			errMsg:  "a curve name is required when algorithm is EdDSA",
		},
		{
			name: "empty algorithm",
			flags: keyRotateFlags{
				Alg: "",
				Yes: true,
			},
			wantErr: true,
			errMsg:  "key algorithm is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testKeyRotateWithDatabaseStorage(t, tt.flags, tt.wantErr, tt.errMsg)
		})
	}
}

func testKeyRotateWithDatabaseStorage(t *testing.T, flags keyRotateFlags, wantErr bool, errMsg string) {
	// Set up database storage config
	envConfig := &common.EnvConfigSchema{
		EncryptionKey: []byte("test-encryption-key-characters-long"),
	}

	// Create test database
	db := testingutils.NewDatabaseForTest(t)

	// Load the instance ID
	instanceID, err := instanceid.Load(t.Context(), db)
	require.NoError(t, err)

	// Get key provider
	keyProvider, err := jwkutils.GetKeyProvider(db, envConfig, instanceID)
	require.NoError(t, err)

	// Run the key rotation
	err = keyRotate(t.Context(), flags, db, instanceID, envConfig)

	if wantErr {
		require.Error(t, err)
		if errMsg != "" {
			require.ErrorContains(t, err, errMsg)
		}
		return
	}

	require.NoError(t, err)

	// Verify key was created
	key, err := keyProvider.LoadKey(t.Context())
	require.NoError(t, err)
	require.NotNil(t, key)

	// Verify the algorithm matches what we requested
	alg, _ := key.Algorithm()
	assert.NotEmpty(t, alg)
	if flags.Alg != "" {
		expectedAlg := flags.Alg
		if expectedAlg == "EdDSA" {
			// EdDSA keys should have the EdDSA algorithm
			assert.Equal(t, "EdDSA", alg.String())
		} else {
			assert.Equal(t, expectedAlg, alg.String())
		}
	}
}

func TestKeyRotateSessionKey(t *testing.T) {
	envConfig := &common.EnvConfigSchema{
		EncryptionKey: []byte("test-encryption-key-characters-long"),
	}

	db := testingutils.NewDatabaseForTest(t)

	instanceID, err := instanceid.Load(t.Context(), db)
	require.NoError(t, err)

	// Seed both keys so we can check that only the session key is replaced
	keyProvider, err := jwkutils.GetKeyProvider(db, envConfig, instanceID)
	require.NoError(t, err)
	signingKey, err := jwkutils.GenerateKey("ES256", "")
	require.NoError(t, err)
	err = keyProvider.SaveKey(t.Context(), signingKey)
	require.NoError(t, err)

	sessionKeyProvider, err := jwkutils.GetSessionKeyProvider(db, envConfig, instanceID)
	require.NoError(t, err)
	originalSessionKey, err := jwkutils.GenerateSessionKey()
	require.NoError(t, err)
	err = sessionKeyProvider.SaveKey(t.Context(), originalSessionKey)
	require.NoError(t, err)

	// Rotate the session key
	err = keyRotate(t.Context(), keyRotateFlags{SessionKey: true, Yes: true}, db, instanceID, envConfig)
	require.NoError(t, err)

	// The session key must have been replaced with a new HS256 key
	rotatedSessionKey, err := sessionKeyProvider.LoadKey(t.Context())
	require.NoError(t, err)
	require.NotNil(t, rotatedSessionKey)

	alg, ok := rotatedSessionKey.Algorithm()
	_ = assert.True(t, ok) &&
		assert.Equal(t, "HS256", alg.String())

	originalKeyID, _ := originalSessionKey.KeyID()
	rotatedKeyID, _ := rotatedSessionKey.KeyID()
	assert.NotEqual(t, originalKeyID, rotatedKeyID, "Session key should have been replaced")

	// The token signing key must be left untouched
	unchangedSigningKey, err := keyProvider.LoadKey(t.Context())
	require.NoError(t, err)
	require.NotNil(t, unchangedSigningKey)

	signingKeyID, _ := signingKey.KeyID()
	unchangedSigningKeyID, _ := unchangedSigningKey.KeyID()
	assert.Equal(t, signingKeyID, unchangedSigningKeyID, "Token signing key should not have been rotated")
}

func TestKeyRotateDoesNotChangeSessionKey(t *testing.T) {
	envConfig := &common.EnvConfigSchema{
		EncryptionKey: []byte("test-encryption-key-characters-long"),
	}

	db := testingutils.NewDatabaseForTest(t)

	instanceID, err := instanceid.Load(t.Context(), db)
	require.NoError(t, err)

	sessionKeyProvider, err := jwkutils.GetSessionKeyProvider(db, envConfig, instanceID)
	require.NoError(t, err)
	originalSessionKey, err := jwkutils.GenerateSessionKey()
	require.NoError(t, err)
	err = sessionKeyProvider.SaveKey(t.Context(), originalSessionKey)
	require.NoError(t, err)

	// Rotating the token signing key must leave existing sessions valid
	err = keyRotate(t.Context(), keyRotateFlags{Alg: "ES256", Yes: true}, db, instanceID, envConfig)
	require.NoError(t, err)

	sessionKey, err := sessionKeyProvider.LoadKey(t.Context())
	require.NoError(t, err)
	require.NotNil(t, sessionKey)

	originalKeyID, _ := originalSessionKey.KeyID()
	sessionKeyID, _ := sessionKey.KeyID()
	assert.Equal(t, originalKeyID, sessionKeyID, "Session key should not have been rotated")
}

func TestKeyRotateMultipleAlgorithms(t *testing.T) {
	algorithms := []struct {
		alg string
		crv string
	}{
		{"RS256", ""},
		{"RS384", ""},
		// Skip RSA-4096 key generation test as it can take a long time
		// {"RS512", ""},
		{"ES256", ""},
		{"ES384", ""},
		{"ES512", ""},
		{"EdDSA", "Ed25519"},
	}

	for _, algo := range algorithms {
		t.Run(algo.alg, func(t *testing.T) {
			// Test with database storage for all algorithms
			testKeyRotateWithDatabaseStorage(t, keyRotateFlags{
				Alg: algo.alg,
				Crv: algo.crv,
				Yes: true,
			}, false, "")
		})
	}
}
