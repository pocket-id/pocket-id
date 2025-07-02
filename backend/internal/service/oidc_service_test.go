package service

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/lestrrat-go/jwx/v3/jwa"
	"github.com/lestrrat-go/jwx/v3/jwk"
	"github.com/lestrrat-go/jwx/v3/jwt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/dto"
	testutils "github.com/pocket-id/pocket-id/backend/internal/utils/testing"
)

// generateTestECDSAKey creates an ECDSA key for testing
func generateTestECDSAKey(t *testing.T) (jwk.Key, []byte) {
	t.Helper()

	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	privateJwk, err := jwk.Import(privateKey)
	require.NoError(t, err)

	err = privateJwk.Set(jwk.KeyIDKey, "test-key-1")
	require.NoError(t, err)
	err = privateJwk.Set(jwk.AlgorithmKey, "ES256")
	require.NoError(t, err)
	err = privateJwk.Set("use", "sig")
	require.NoError(t, err)

	publicJwk, err := jwk.PublicKeyOf(privateJwk)
	require.NoError(t, err)

	// Create a JWK Set with the public key
	jwkSet := jwk.NewSet()
	err = jwkSet.AddKey(publicJwk)
	require.NoError(t, err)
	jwkSetJSON, err := json.Marshal(jwkSet)
	require.NoError(t, err)

	return privateJwk, jwkSetJSON
}

func TestOidcService_jwkSetForURL(t *testing.T) {
	// Generate a test key for JWKS
	_, jwkSetJSON1 := generateTestECDSAKey(t)
	_, jwkSetJSON2 := generateTestECDSAKey(t)

	// Create a mock HTTP client with responses for different URLs
	const (
		url1 = "https://example.com/.well-known/jwks.json"
		url2 = "https://other-issuer.com/jwks"
	)
	mockResponses := map[string]*http.Response{
		//nolint:bodyclose
		url1: testutils.NewMockResponse(http.StatusOK, string(jwkSetJSON1)),
		//nolint:bodyclose
		url2: testutils.NewMockResponse(http.StatusOK, string(jwkSetJSON2)),
	}
	httpClient := &http.Client{
		Transport: &testutils.MockRoundTripper{
			Responses: mockResponses,
		},
	}

	// Create the OidcService with our mock client
	s := &OidcService{
		httpClient: httpClient,
	}

	var err error
	s.jwkCache, err = s.getJWKCache(t.Context())
	require.NoError(t, err)

	t.Run("Fetches and caches JWK set", func(t *testing.T) {
		jwks, err := s.jwkSetForURL(t.Context(), url1)
		require.NoError(t, err)
		require.NotNil(t, jwks)

		// Verify the JWK set contains our key
		require.Equal(t, 1, jwks.Len())
	})

	t.Run("Fails with invalid URL", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
		defer cancel()
		_, err := s.jwkSetForURL(ctx, "https://bad-url.com")
		require.Error(t, err)
		require.ErrorIs(t, err, context.DeadlineExceeded)
	})

	t.Run("Safe for concurrent use", func(t *testing.T) {
		const concurrency = 20

		// Channel to collect errors
		errChan := make(chan error, concurrency)

		// Start concurrent requests
		for range concurrency {
			go func() {
				jwks, err := s.jwkSetForURL(t.Context(), url2)
				if err != nil {
					errChan <- err
					return
				}

				// Verify the JWK set is valid
				if jwks == nil || jwks.Len() != 1 {
					errChan <- assert.AnError
					return
				}

				errChan <- nil
			}()
		}

		// Check for errors
		for range concurrency {
			assert.NoError(t, <-errChan, "Concurrent JWK set fetching should not produce errors")
		}
	})
}

func TestOidcService_verifyClientCredentialsInternal(t *testing.T) {
	const (
		federatedClientIssuer         = "https://external-idp.com"
		federatedClientAudience       = "https://pocket-id.com"
		federatedClientIssuerDefaults = "https://external-idp-defaults.com/"
	)

	var err error
	// Create a test database
	db := testutils.NewDatabaseForTest(t)

	// Create two JWKs for testing
	privateJWK, jwkSetJSON := generateTestECDSAKey(t)
	require.NoError(t, err)
	privateJWKDefaults, jwkSetJSONDefaults := generateTestECDSAKey(t)
	require.NoError(t, err)

	// Create a mock HTTP client with custom transport to return the JWKS
	httpClient := &http.Client{
		Transport: &testutils.MockRoundTripper{
			Responses: map[string]*http.Response{
				//nolint:bodyclose
				federatedClientIssuer + "/jwks.json": testutils.NewMockResponse(http.StatusOK, string(jwkSetJSON)),
				//nolint:bodyclose
				federatedClientIssuerDefaults + ".well-known/jwks.json": testutils.NewMockResponse(http.StatusOK, string(jwkSetJSONDefaults)),
			},
		},
	}

	// Init the OidcService
	s := &OidcService{
		db:         db,
		httpClient: httpClient,
	}
	s.jwkCache, err = s.getJWKCache(t.Context())
	require.NoError(t, err)

	// Create the test clients
	// 1. Confidential client
	confidentialClient, err := s.CreateClient(t.Context(), dto.OidcClientCreateDto{
		Name:         "Confidential Client",
		CallbackURLs: []string{"https://example.com/callback"},
	}, "test-user-id")
	require.NoError(t, err)

	// Create a client secret for the confidential client
	confidentialSecret, err := s.CreateClientSecret(t.Context(), confidentialClient.ID)
	require.NoError(t, err)

	// 2. Public client
	publicClient, err := s.CreateClient(t.Context(), dto.OidcClientCreateDto{
		Name:         "Public Client",
		CallbackURLs: []string{"https://example.com/callback"},
		IsPublic:     true,
	}, "test-user-id")
	require.NoError(t, err)

	// 3. Confidential client with federated identity
	federatedClient, err := s.CreateClient(t.Context(), dto.OidcClientCreateDto{
		Name:         "Federated Client",
		CallbackURLs: []string{"https://example.com/callback"},
	}, "test-user-id")
	require.NoError(t, err)

	federatedClient, err = s.UpdateClient(t.Context(), federatedClient.ID, dto.OidcClientCreateDto{
		Name:         federatedClient.Name,
		CallbackURLs: federatedClient.CallbackURLs,
		Credentials: dto.OidcClientCredentialsDto{
			FederatedIdentities: []dto.OidcClientFederatedIdentityDto{
				{
					Issuer:   federatedClientIssuer,
					Audience: federatedClientAudience,
					Subject:  federatedClient.ID,
					JWKS:     federatedClientIssuer + "/jwks.json",
				},
				{Issuer: federatedClientIssuerDefaults},
			},
		},
	})
	require.NoError(t, err)

	// Test cases for confidential client (using client secret)
	t.Run("Confidential client", func(t *testing.T) {
		t.Run("Succeeds with valid secret", func(t *testing.T) {
			// Test with valid client credentials
			client, err := s.verifyClientCredentialsInternal(t.Context(), s.db, ClientAuthCredentials{
				ClientID:     confidentialClient.ID,
				ClientSecret: confidentialSecret,
			}, true)
			require.NoError(t, err)
			require.NotNil(t, client)
			assert.Equal(t, confidentialClient.ID, client.ID)
		})

		t.Run("Fails with invalid secret", func(t *testing.T) {
			// Test with invalid client secret
			client, err := s.verifyClientCredentialsInternal(t.Context(), s.db, ClientAuthCredentials{
				ClientID:     confidentialClient.ID,
				ClientSecret: "invalid-secret",
			}, true)
			require.Error(t, err)
			require.ErrorIs(t, err, &common.OidcClientSecretInvalidError{})
			assert.Nil(t, client)
		})

		t.Run("Fails with missing secret", func(t *testing.T) {
			// Test with missing client secret
			client, err := s.verifyClientCredentialsInternal(t.Context(), s.db, ClientAuthCredentials{
				ClientID: confidentialClient.ID,
			}, true)
			require.Error(t, err)
			require.ErrorIs(t, err, &common.OidcMissingClientCredentialsError{})
			assert.Nil(t, client)
		})
	})

	// Test cases for public client
	t.Run("Public client", func(t *testing.T) {
		t.Run("Succeeds with no credentials", func(t *testing.T) {
			// Public clients don't require client secret
			client, err := s.verifyClientCredentialsInternal(t.Context(), s.db, ClientAuthCredentials{
				ClientID: publicClient.ID,
			}, true)
			require.NoError(t, err)
			require.NotNil(t, client)
			assert.Equal(t, publicClient.ID, client.ID)
		})

		t.Run("Fails with no credentials if allowPublicClientsWithoutAuth is false", func(t *testing.T) {
			// Public clients don't require client secret
			client, err := s.verifyClientCredentialsInternal(t.Context(), s.db, ClientAuthCredentials{
				ClientID: publicClient.ID,
			}, false)
			require.Error(t, err)
			require.ErrorIs(t, err, &common.OidcMissingClientCredentialsError{})
			assert.Nil(t, client)
		})
	})

	// Test cases for federated client using JWT assertion
	t.Run("Federated client", func(t *testing.T) {
		t.Run("Succeeds with valid JWT", func(t *testing.T) {
			// Create JWT for federated identity
			token, err := jwt.NewBuilder().
				Issuer(federatedClientIssuer).
				Audience([]string{federatedClientAudience}).
				Subject(federatedClient.ID).
				IssuedAt(time.Now()).
				Expiration(time.Now().Add(10 * time.Minute)).
				Build()
			require.NoError(t, err)
			signedToken, err := jwt.Sign(token, jwt.WithKey(jwa.ES256(), privateJWK))
			require.NoError(t, err)

			// Test with valid JWT assertion
			client, err := s.verifyClientCredentialsInternal(t.Context(), s.db, ClientAuthCredentials{
				ClientID:            federatedClient.ID,
				ClientAssertionType: ClientAssertionTypeJWTBearer,
				ClientAssertion:     string(signedToken),
			}, true)
			require.NoError(t, err)
			require.NotNil(t, client)
			assert.Equal(t, federatedClient.ID, client.ID)
		})

		t.Run("Fails with malformed JWT", func(t *testing.T) {
			// Test with invalid JWT assertion (just a random string)
			client, err := s.verifyClientCredentialsInternal(t.Context(), s.db, ClientAuthCredentials{
				ClientID:            federatedClient.ID,
				ClientAssertionType: ClientAssertionTypeJWTBearer,
				ClientAssertion:     "invalid.jwt.token",
			}, true)
			require.Error(t, err)
			require.ErrorIs(t, err, &common.OidcClientAssertionInvalidError{})
			assert.Nil(t, client)
		})

		testBadJWT := func(builderFn func(builder *jwt.Builder)) func(t *testing.T) {
			return func(t *testing.T) {
				// Populate all claims with valid values
				builder := jwt.NewBuilder().
					Issuer(federatedClientIssuer).
					Audience([]string{federatedClientAudience}).
					Subject(federatedClient.ID).
					IssuedAt(time.Now()).
					Expiration(time.Now().Add(10 * time.Minute))

				// Call builderFn to override the claims
				builderFn(builder)

				token, err := builder.Build()
				require.NoError(t, err)
				signedToken, err := jwt.Sign(token, jwt.WithKey(jwa.ES256(), privateJWK))
				require.NoError(t, err)

				// Test with invalid JWT assertion
				client, err := s.verifyClientCredentialsInternal(t.Context(), s.db, ClientAuthCredentials{
					ClientID:            federatedClient.ID,
					ClientAssertionType: ClientAssertionTypeJWTBearer,
					ClientAssertion:     string(signedToken),
				}, true)
				require.Error(t, err)
				require.ErrorIs(t, err, &common.OidcClientAssertionInvalidError{})
				require.Nil(t, client)
			}
		}

		t.Run("Fails with expired JWT", testBadJWT(func(builder *jwt.Builder) {
			builder.Expiration(time.Now().Add(-30 * time.Minute))
		}))

		t.Run("Fails with wrong issuer in JWT", testBadJWT(func(builder *jwt.Builder) {
			builder.Issuer("https://bad-issuer.com")
		}))

		t.Run("Fails with wrong audience in JWT", testBadJWT(func(builder *jwt.Builder) {
			builder.Audience([]string{"bad-audience"})
		}))

		t.Run("Fails with wrong subject in JWT", testBadJWT(func(builder *jwt.Builder) {
			builder.Subject("bad-subject")
		}))

		t.Run("Uses default values for audience and subject", func(t *testing.T) {
			// Create JWT for federated identity
			token, err := jwt.NewBuilder().
				Issuer(federatedClientIssuerDefaults).
				Audience([]string{common.EnvConfig.AppURL}).
				Subject(federatedClient.ID).
				IssuedAt(time.Now()).
				Expiration(time.Now().Add(10 * time.Minute)).
				Build()
			require.NoError(t, err)
			signedToken, err := jwt.Sign(token, jwt.WithKey(jwa.ES256(), privateJWKDefaults))
			require.NoError(t, err)

			// Test with valid JWT assertion
			client, err := s.verifyClientCredentialsInternal(t.Context(), s.db, ClientAuthCredentials{
				ClientID:            federatedClient.ID,
				ClientAssertionType: ClientAssertionTypeJWTBearer,
				ClientAssertion:     string(signedToken),
			}, true)
			require.NoError(t, err)
			require.NotNil(t, client)
			assert.Equal(t, federatedClient.ID, client.ID)
		})
	})
}
