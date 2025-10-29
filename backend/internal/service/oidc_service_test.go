package service

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
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
	"github.com/pocket-id/pocket-id/backend/internal/model"
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
		federatedClientIssuerOther    = "https://external-idp-2.com"
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

	// Create a mock config and JwtService to test complete a token creation process
	mockConfig := NewTestAppConfigService(&model.AppConfig{
		SessionDuration: model.AppConfigVariable{Value: "60"}, // 60 minutes
	})
	mockJwtService, err := NewJwtService(db, mockConfig)
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
		db:               db,
		jwtService:       mockJwtService,
		appConfigService: mockConfig,
		httpClient:       httpClient,
	}
	s.jwkCache, err = s.getJWKCache(t.Context())
	require.NoError(t, err)

	// Create the test clients
	// 1. Confidential client
	confidentialClient, err := s.CreateClient(t.Context(), dto.OidcClientCreateDto{
		OidcClientUpdateDto: dto.OidcClientUpdateDto{
			Name:         "Confidential Client",
			CallbackURLs: []string{"https://example.com/callback"},
		},
	}, "test-user-id")
	require.NoError(t, err)

	// Create a client secret for the confidential client
	confidentialSecret, err := s.CreateClientSecret(t.Context(), confidentialClient.ID)
	require.NoError(t, err)

	// 2. Public client
	publicClient, err := s.CreateClient(t.Context(), dto.OidcClientCreateDto{
		OidcClientUpdateDto: dto.OidcClientUpdateDto{
			Name:         "Public Client",
			CallbackURLs: []string{"https://example.com/callback"},
			IsPublic:     true,
		},
	}, "test-user-id")
	require.NoError(t, err)

	// 3. Confidential client with federated identity
	federatedClient, err := s.CreateClient(t.Context(), dto.OidcClientCreateDto{
		OidcClientUpdateDto: dto.OidcClientUpdateDto{
			Name:         "Federated Client",
			CallbackURLs: []string{"https://example.com/callback"},
		},
	}, "test-user-id")
	require.NoError(t, err)

	differingFederatedID := "some-issuer-specific-sub"
	federatedClient, err = s.UpdateClient(t.Context(), federatedClient.ID, dto.OidcClientUpdateDto{
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
				{
					Issuer:   federatedClientIssuerOther,
					Audience: federatedClientAudience,
					Subject:  differingFederatedID,
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

		t.Run("Succeeds with valid JWT with sub different from client ID and no client ID", func(t *testing.T) {
			// Create JWT for federated identity
			token, err := jwt.NewBuilder().
				Issuer(federatedClientIssuerOther).
				Audience([]string{federatedClientAudience}).
				Subject(differingFederatedID).
				IssuedAt(time.Now()).
				Expiration(time.Now().Add(10 * time.Minute)).
				Build()
			require.NoError(t, err)
			signedToken, err := jwt.Sign(token, jwt.WithKey(jwa.ES256(), privateJWK))
			require.NoError(t, err)

			// Test with valid JWT assertion
			client, err := s.verifyClientCredentialsInternal(t.Context(), s.db, ClientAuthCredentials{
				ClientAssertionType: ClientAssertionTypeJWTBearer,
				ClientAssertion:     string(signedToken),
			}, true)
			require.NoError(t, err)
			require.NotNil(t, client)
			assert.Equal(t, federatedClient.ID, client.ID)
		})

		t.Run("Fails with client ID mismatch", func(t *testing.T) {
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

			// Test with valid JWT assertion but mismatching client ID
			client, err := s.verifyClientCredentialsInternal(t.Context(), s.db, ClientAuthCredentials{
				ClientID:            "something-else",
				ClientAssertionType: ClientAssertionTypeJWTBearer,
				ClientAssertion:     string(signedToken),
			}, true)
			require.Error(t, err)
			require.ErrorIs(t, err, &common.OidcClientAssertionInvalidError{})
			assert.Nil(t, client)
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

	t.Run("Complete token creation flow", func(t *testing.T) {
		t.Run("Client Credentials flow", func(t *testing.T) {
			t.Run("Succeeds with valid secret", func(t *testing.T) {
				// Generate a token
				input := dto.OidcCreateTokensDto{
					ClientID:     confidentialClient.ID,
					ClientSecret: confidentialSecret,
				}
				token, err := s.createTokenFromClientCredentials(t.Context(), input)
				require.NoError(t, err)
				require.NotNil(t, token)

				// Verify the token
				claims, err := s.jwtService.VerifyOAuthAccessToken(token.AccessToken)
				require.NoError(t, err, "Failed to verify generated token")

				// Check the claims
				subject, ok := claims.Subject()
				_ = assert.True(t, ok, "User ID not found in token") &&
					assert.Equal(t, "client-"+confidentialClient.ID, subject, "Token subject should match confidential client ID with prefix")
				audience, ok := claims.Audience()
				_ = assert.True(t, ok, "Audience not found in token") &&
					assert.Equal(t, []string{confidentialClient.ID}, audience, "Audience should contain confidential client ID")
			})

			t.Run("Fails with invalid secret", func(t *testing.T) {
				input := dto.OidcCreateTokensDto{
					ClientID:     confidentialClient.ID,
					ClientSecret: "invalid-secret",
				}
				_, err := s.createTokenFromClientCredentials(t.Context(), input)
				require.Error(t, err)
				require.ErrorIs(t, err, &common.OidcClientSecretInvalidError{})
			})

			t.Run("Fails without client secret for public clients", func(t *testing.T) {
				input := dto.OidcCreateTokensDto{
					ClientID: publicClient.ID,
				}
				_, err := s.createTokenFromClientCredentials(t.Context(), input)
				require.Error(t, err)
				require.ErrorIs(t, err, &common.OidcMissingClientCredentialsError{})
			})

			t.Run("Succeeds with valid assertion", func(t *testing.T) {
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

				// Generate a token
				input := dto.OidcCreateTokensDto{
					ClientAssertion:     string(signedToken),
					ClientAssertionType: ClientAssertionTypeJWTBearer,
				}
				createdToken, err := s.createTokenFromClientCredentials(t.Context(), input)
				require.NoError(t, err)
				require.NotNil(t, token)

				// Verify the token
				claims, err := s.jwtService.VerifyOAuthAccessToken(createdToken.AccessToken)
				require.NoError(t, err, "Failed to verify generated token")

				// Check the claims
				subject, ok := claims.Subject()
				_ = assert.True(t, ok, "User ID not found in token") &&
					assert.Equal(t, "client-"+federatedClient.ID, subject, "Token subject should match federated client ID with prefix")
				audience, ok := claims.Audience()
				_ = assert.True(t, ok, "Audience not found in token") &&
					assert.Equal(t, []string{federatedClient.ID}, audience, "Audience should contain the federated client ID")
			})

			t.Run("Fails with invalid assertion", func(t *testing.T) {
				input := dto.OidcCreateTokensDto{
					ClientAssertion:     "invalid.jwt.token",
					ClientAssertionType: ClientAssertionTypeJWTBearer,
				}
				_, err := s.createTokenFromClientCredentials(t.Context(), input)
				require.Error(t, err)
				require.ErrorIs(t, err, &common.OidcClientAssertionInvalidError{})
			})

			t.Run("Succeeds with custom resource", func(t *testing.T) {
				// Generate a token
				input := dto.OidcCreateTokensDto{
					ClientID:     confidentialClient.ID,
					ClientSecret: confidentialSecret,
					Resource:     "https://example.com/",
				}
				token, err := s.createTokenFromClientCredentials(t.Context(), input)
				require.NoError(t, err)
				require.NotNil(t, token)

				// Verify the token
				claims, err := s.jwtService.VerifyOAuthAccessToken(token.AccessToken)
				require.NoError(t, err, "Failed to verify generated token")

				// Check the claims
				subject, ok := claims.Subject()
				_ = assert.True(t, ok, "User ID not found in token") &&
					assert.Equal(t, "client-"+confidentialClient.ID, subject, "Token subject should match confidential client ID with prefix")
				audience, ok := claims.Audience()
				_ = assert.True(t, ok, "Audience not found in token") &&
					assert.Equal(t, []string{input.Resource}, audience, "Audience should contain the resource provided in request")
			})
		})
	})
}

func TestValidateCodeVerifier_Plain(t *testing.T) {
	require.False(t, validateCodeVerifier("", "", false))
	require.False(t, validateCodeVerifier("", "", true))

	t.Run("plain", func(t *testing.T) {
		require.False(t, validateCodeVerifier("", "challenge", false))
		require.False(t, validateCodeVerifier("verifier", "", false))
		require.True(t, validateCodeVerifier("plainVerifier", "plainVerifier", false))
		require.False(t, validateCodeVerifier("plainVerifier", "otherVerifier", false))
	})

	t.Run("SHA 256", func(t *testing.T) {
		codeVerifier := "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
		hash := sha256.Sum256([]byte(codeVerifier))
		codeChallenge := base64.RawURLEncoding.EncodeToString(hash[:])

		require.True(t, validateCodeVerifier(codeVerifier, codeChallenge, true))
		require.False(t, validateCodeVerifier("wrongVerifier", codeChallenge, true))
		require.False(t, validateCodeVerifier(codeVerifier, "!", true))

		// Invalid base64
		require.False(t, validateCodeVerifier("NOT!VALID", codeChallenge, true))
	})
}
