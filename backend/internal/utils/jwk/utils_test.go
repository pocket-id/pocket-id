package jwk

import (
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"testing"

	"github.com/lestrrat-go/jwx/v3/jwa"
	"github.com/lestrrat-go/jwx/v3/jwk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateKey(t *testing.T) {
	tests := []struct {
		name        string
		alg         string
		crv         string
		expectError bool
		expectedAlg jwa.SignatureAlgorithm
	}{
		{
			name:        "RS256",
			alg:         jwa.RS256().String(),
			crv:         "",
			expectError: false,
			expectedAlg: jwa.RS256(),
		},
		{
			name:        "RS384",
			alg:         jwa.RS384().String(),
			crv:         "",
			expectError: false,
			expectedAlg: jwa.RS384(),
		},
		// Skip the RS512 test as generating a RSA-4096 key can take some time
		/* {
			name:        "RS512",
			alg:         jwa.RS512().String(),
			crv:         "",
			expectError: false,
			expectedAlg: jwa.RS512(),
		}, */
		{
			name:        "ES256",
			alg:         jwa.ES256().String(),
			crv:         jwa.P256().String(),
			expectError: false,
			expectedAlg: jwa.ES256(),
		},
		{
			name:        "ES384",
			alg:         jwa.ES384().String(),
			crv:         jwa.P384().String(),
			expectError: false,
			expectedAlg: jwa.ES384(),
		},
		{
			name:        "ES512",
			alg:         jwa.ES512().String(),
			crv:         jwa.P521().String(),
			expectError: false,
			expectedAlg: jwa.ES512(),
		},
		{
			name:        "EdDSA with Ed25519",
			alg:         jwa.EdDSA().String(),
			crv:         jwa.Ed25519().String(),
			expectError: false,
			expectedAlg: jwa.EdDSA(),
		},
		{
			name:        "EdDSA with unsupported curve",
			alg:         jwa.EdDSA().String(),
			crv:         "unsupported",
			expectError: true,
		},
		{
			name:        "Unsupported algorithm",
			alg:         "UNSUPPORTED",
			crv:         "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key, err := GenerateKey(tt.alg, tt.crv)

			if tt.expectError {
				require.Error(t, err)
				assert.Nil(t, key)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, key)

			// Verify the algorithm is set correctly
			alg, ok := key.Algorithm()
			require.True(t, ok, "algorithm should be set in the key")
			assert.Equal(t, tt.expectedAlg.String(), alg.String())

			// Verify other required fields are set
			kid, ok := key.KeyID()
			assert.True(t, ok, "key ID should be set")
			assert.NotEmpty(t, kid, "key ID should not be empty")

			usage, ok := key.KeyUsage()
			assert.True(t, ok, "key usage should be set")
			assert.Equal(t, KeyUsageSigning, usage)

			var crv any
			_ = key.Get("crv", &crv)

			// Verify key type matches expected algorithm
			switch tt.expectedAlg {
			case jwa.RS256(), jwa.RS384(), jwa.RS512():
				assert.Equal(t, jwa.RSA(), key.KeyType())
				assert.Nil(t, crv)
			case jwa.ES256(), jwa.ES384(), jwa.ES512():
				assert.Equal(t, jwa.EC(), key.KeyType())
				eca, ok := crv.(jwa.EllipticCurveAlgorithm)
				_ = assert.NotNil(t, crv) &&
					assert.True(t, ok) &&
					assert.Equal(t, tt.crv, eca.String())
			case jwa.EdDSA():
				assert.Equal(t, jwa.OKP(), key.KeyType())
				eca, ok := crv.(jwa.EllipticCurveAlgorithm)
				_ = assert.NotNil(t, crv) &&
					assert.True(t, ok) &&
					assert.Equal(t, tt.crv, eca.String())
			}
		})
	}
}

func TestEnsureAlgInKey(t *testing.T) {
	// Generate an RSA-2048 key
	rsaKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	t.Run("does not change alg already set", func(t *testing.T) {
		// Import the RSA key
		key, err := jwk.Import(rsaKey)
		require.NoError(t, err)

		// Pre-set the algorithm
		_ = key.Set(jwk.AlgorithmKey, jwa.RS256())

		// Call EnsureAlgInKey with a different algorithm
		EnsureAlgInKey(key, jwa.RS384().String(), "")

		// Verify the algorithm wasn't changed
		alg, ok := key.Algorithm()
		require.True(t, ok)
		assert.Equal(t, jwa.RS256().String(), alg.String())
	})

	t.Run("set algorithm to explicitly-provided value", func(t *testing.T) {
		tests := []struct {
			name        string
			keyGen      func() (any, error)
			alg         string
			crv         string
			expectedAlg jwa.SignatureAlgorithm
			expectedCrv string
		}{
			{
				name: "RSA key with RS384",
				keyGen: func() (any, error) {
					return rsaKey, nil
				},
				alg:         jwa.RS384().String(),
				crv:         "",
				expectedAlg: jwa.RS384(),
				expectedCrv: "",
			},
			{
				name: "ECDSA key with ES384",
				keyGen: func() (any, error) {
					return ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
				},
				alg:         jwa.ES384().String(),
				crv:         jwa.P384().String(),
				expectedAlg: jwa.ES384(),
				expectedCrv: jwa.P384().String(),
			},
			{
				name: "Ed25519 key with EdDSA",
				keyGen: func() (any, error) {
					_, priv, err := ed25519.GenerateKey(rand.Reader)
					return priv, err
				},
				alg:         jwa.EdDSA().String(),
				crv:         jwa.Ed25519().String(),
				expectedAlg: jwa.EdDSA(),
				expectedCrv: jwa.Ed25519().String(),
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				rawKey, err := tt.keyGen()
				require.NoError(t, err)

				key, err := jwk.Import(rawKey)
				require.NoError(t, err)

				// Ensure no algorithm is set initially
				_, ok := key.Algorithm()
				assert.False(t, ok)

				// Call EnsureAlgInKey
				EnsureAlgInKey(key, tt.alg, tt.crv)

				// Verify the algorithm was set correctly
				alg, ok := key.Algorithm()
				require.True(t, ok)
				assert.Equal(t, tt.expectedAlg.String(), alg.String())

				// Verify curve if expected
				if tt.expectedCrv != "" {
					var crv any
					_ = key.Get("crv", &crv)
					require.NotNil(t, crv)
					eca, ok := crv.(jwa.EllipticCurveAlgorithm)
					require.True(t, ok)
					assert.Equal(t, tt.expectedCrv, eca.String())
				}
			})
		}
	})

	t.Run("set default algorithms if not present", func(t *testing.T) {
		tests := []struct {
			name        string
			keyGen      func() (any, error)
			expectedAlg jwa.SignatureAlgorithm
			expectedCrv string
		}{
			{
				name: "RSA key defaults to RS256",
				keyGen: func() (any, error) {
					return rsaKey, nil
				},
				expectedAlg: jwa.RS256(),
				expectedCrv: "",
			},
			{
				name: "ECDSA key defaults to ES256 with P256",
				keyGen: func() (any, error) {
					return ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
				},
				expectedAlg: jwa.ES256(),
				expectedCrv: jwa.P256().String(),
			},
			{
				name: "Ed25519 key defaults to EdDSA with Ed25519",
				keyGen: func() (any, error) {
					_, priv, err := ed25519.GenerateKey(rand.Reader)
					return priv, err
				},
				expectedAlg: jwa.EdDSA(),
				expectedCrv: jwa.Ed25519().String(),
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				rawKey, err := tt.keyGen()
				require.NoError(t, err)

				key, err := jwk.Import(rawKey)
				require.NoError(t, err)

				// Ensure no algorithm is set initially
				_, ok := key.Algorithm()
				assert.False(t, ok)

				// Call EnsureAlgInKey with empty parameters
				EnsureAlgInKey(key, "", "")

				// Verify the default algorithm was set
				alg, ok := key.Algorithm()
				require.True(t, ok)
				assert.Equal(t, tt.expectedAlg.String(), alg.String())

				// Verify curve if expected
				if tt.expectedCrv != "" {
					var crv any
					_ = key.Get("crv", &crv)
					require.NotNil(t, crv)
					eca, ok := crv.(jwa.EllipticCurveAlgorithm)
					require.True(t, ok)
					assert.Equal(t, tt.expectedCrv, eca.String())
				}
			})
		}
	})

	t.Run("invalid curve should not set curve parameter", func(t *testing.T) {
		rsaKey, err := rsa.GenerateKey(rand.Reader, 2048)
		require.NoError(t, err)

		key, err := jwk.Import(rsaKey)
		require.NoError(t, err)

		// Call EnsureAlgInKey with invalid curve
		EnsureAlgInKey(key, jwa.RS256().String(), "invalid-curve")

		// Verify algorithm was set but curve was not
		alg, ok := key.Algorithm()
		require.True(t, ok)
		assert.Equal(t, jwa.RS256().String(), alg.String())

		var crv any
		_ = key.Get("crv", &crv)
		assert.Nil(t, crv)
	})
}
