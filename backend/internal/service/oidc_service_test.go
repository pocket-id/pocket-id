package service

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
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
	"github.com/pocket-id/pocket-id/backend/internal/storage"
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
	common.EnvConfig.EncryptionKey = []byte("0123456789abcdef0123456789abcdef")

	// Create two JWKs for testing
	privateJWK, jwkSetJSON := generateTestECDSAKey(t)
	require.NoError(t, err)
	privateJWKDefaults, jwkSetJSONDefaults := generateTestECDSAKey(t)
	require.NoError(t, err)

	// Create a mock config and JwtService to test complete a token creation process
	mockConfig := NewTestAppConfigService(&model.AppConfig{
		SessionDuration: model.AppConfigVariable{Value: "60"}, // 60 minutes
	})
	mockJwtService, err := NewJwtService(t.Context(), db, mockConfig)
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

func TestOidcService_updateClientLogoType(t *testing.T) {
	// Create a test database
	db := testutils.NewDatabaseForTest(t)

	// Create database storage
	dbStorage, err := storage.NewDatabaseStorage(db)
	require.NoError(t, err)

	// Init the OidcService
	s := &OidcService{
		db:          db,
		fileStorage: dbStorage,
	}

	// Create a test client
	client := model.OidcClient{
		Name:         "Test Client",
		CallbackURLs: model.UrlList{"https://example.com/callback"},
	}
	err = db.Create(&client).Error
	require.NoError(t, err)

	// Helper function to check if a file exists in storage
	fileExists := func(t *testing.T, path string) bool {
		t.Helper()
		_, _, err := dbStorage.Open(t.Context(), path)
		return err == nil
	}

	// Helper function to create a dummy file in storage
	createDummyFile := func(t *testing.T, path string) {
		t.Helper()
		err := dbStorage.Save(t.Context(), path, strings.NewReader("dummy content"))
		require.NoError(t, err)
	}

	t.Run("Updates light logo type for client without previous logo", func(t *testing.T) {
		// Update the logo type
		err := s.updateClientLogoType(t.Context(), client.ID, "png", true)
		require.NoError(t, err)

		// Verify the client was updated
		var updatedClient model.OidcClient
		err = db.First(&updatedClient, "id = ?", client.ID).Error
		require.NoError(t, err)
		require.NotNil(t, updatedClient.ImageType)
		assert.Equal(t, "png", *updatedClient.ImageType)
	})

	t.Run("Updates dark logo type for client without previous dark logo", func(t *testing.T) {
		// Update the dark logo type
		err := s.updateClientLogoType(t.Context(), client.ID, "jpg", false)
		require.NoError(t, err)

		// Verify the client was updated
		var updatedClient model.OidcClient
		err = db.First(&updatedClient, "id = ?", client.ID).Error
		require.NoError(t, err)
		require.NotNil(t, updatedClient.DarkImageType)
		assert.Equal(t, "jpg", *updatedClient.DarkImageType)
	})

	t.Run("Updates light logo type and deletes old file when type changes", func(t *testing.T) {
		// Create the old PNG file in storage
		oldPath := "oidc-client-images/" + client.ID + ".png"
		createDummyFile(t, oldPath)
		require.True(t, fileExists(t, oldPath), "Old file should exist before update")

		// Client currently has a PNG logo, update to WEBP
		err := s.updateClientLogoType(t.Context(), client.ID, "webp", true)
		require.NoError(t, err)

		// Verify the client was updated
		var updatedClient model.OidcClient
		err = db.First(&updatedClient, "id = ?", client.ID).Error
		require.NoError(t, err)
		require.NotNil(t, updatedClient.ImageType)
		assert.Equal(t, "webp", *updatedClient.ImageType)

		// Old PNG file should be deleted
		assert.False(t, fileExists(t, oldPath), "Old PNG file should have been deleted")
	})

	t.Run("Updates dark logo type and deletes old file when type changes", func(t *testing.T) {
		// Create the old JPG dark file in storage
		oldPath := "oidc-client-images/" + client.ID + "-dark.jpg"
		createDummyFile(t, oldPath)
		require.True(t, fileExists(t, oldPath), "Old dark file should exist before update")

		// Client currently has a JPG dark logo, update to WEBP
		err := s.updateClientLogoType(t.Context(), client.ID, "webp", false)
		require.NoError(t, err)

		// Verify the client was updated
		var updatedClient model.OidcClient
		err = db.First(&updatedClient, "id = ?", client.ID).Error
		require.NoError(t, err)
		require.NotNil(t, updatedClient.DarkImageType)
		assert.Equal(t, "webp", *updatedClient.DarkImageType)

		// Old JPG dark file should be deleted
		assert.False(t, fileExists(t, oldPath), "Old JPG dark file should have been deleted")
	})

	t.Run("Does not delete file when type remains the same", func(t *testing.T) {
		// Create the WEBP file in storage
		webpPath := "oidc-client-images/" + client.ID + ".webp"
		createDummyFile(t, webpPath)
		require.True(t, fileExists(t, webpPath), "WEBP file should exist before update")

		// Update to the same type (WEBP)
		err := s.updateClientLogoType(t.Context(), client.ID, "webp", true)
		require.NoError(t, err)

		// Verify the client still has WEBP
		var updatedClient model.OidcClient
		err = db.First(&updatedClient, "id = ?", client.ID).Error
		require.NoError(t, err)
		require.NotNil(t, updatedClient.ImageType)
		assert.Equal(t, "webp", *updatedClient.ImageType)

		// WEBP file should still exist since type didn't change
		assert.True(t, fileExists(t, webpPath), "WEBP file should still exist")
	})

	t.Run("Returns error for non-existent client", func(t *testing.T) {
		err := s.updateClientLogoType(t.Context(), "non-existent-client-id", "png", true)
		require.Error(t, err)
		require.ErrorContains(t, err, "failed to look up client")
	})
}

func TestOidcService_downloadAndSaveLogoFromURL(t *testing.T) {
	// Create a test database
	db := testutils.NewDatabaseForTest(t)

	// Create database storage
	dbStorage, err := storage.NewDatabaseStorage(db)
	require.NoError(t, err)

	// Create a test client
	client := model.OidcClient{
		Name:         "Test Client",
		CallbackURLs: model.UrlList{"https://example.com/callback"},
	}
	err = db.Create(&client).Error
	require.NoError(t, err)

	// Helper function to check if a file exists in storage
	fileExists := func(t *testing.T, path string) bool {
		t.Helper()
		_, _, err := dbStorage.Open(t.Context(), path)
		return err == nil
	}

	// Helper function to get file content from storage
	getFileContent := func(t *testing.T, path string) []byte {
		t.Helper()
		reader, _, err := dbStorage.Open(t.Context(), path)
		require.NoError(t, err)
		defer reader.Close()
		content, err := io.ReadAll(reader)
		require.NoError(t, err)
		return content
	}

	t.Run("Successfully downloads and saves PNG logo from URL", func(t *testing.T) {
		// Create mock PNG content
		pngContent := []byte("fake-png-content")

		// Create a mock HTTP response with headers
		//nolint:bodyclose
		pngResponse := testutils.NewMockResponse(http.StatusOK, string(pngContent))
		pngResponse.Header.Set("Content-Type", "image/png")

		// Create a mock HTTP client with responses
		mockResponses := map[string]*http.Response{
			//nolint:bodyclose
			"https://example.com/logo.png": pngResponse,
		}
		httpClient := &http.Client{
			Transport: &testutils.MockRoundTripper{
				Responses: mockResponses,
			},
		}

		// Init the OidcService with mock HTTP client
		s := &OidcService{
			db:          db,
			fileStorage: dbStorage,
			httpClient:  httpClient,
		}

		// Download and save the logo
		err := s.downloadAndSaveLogoFromURL(t.Context(), client.ID, "https://example.com/logo.png", true)
		require.NoError(t, err)

		// Verify the file was saved
		logoPath := "oidc-client-images/" + client.ID + ".png"
		require.True(t, fileExists(t, logoPath), "Logo file should exist in storage")

		// Verify the content
		savedContent := getFileContent(t, logoPath)
		assert.Equal(t, pngContent, savedContent)

		// Verify the client was updated
		var updatedClient model.OidcClient
		err = db.First(&updatedClient, "id = ?", client.ID).Error
		require.NoError(t, err)
		require.NotNil(t, updatedClient.ImageType)
		assert.Equal(t, "png", *updatedClient.ImageType)
	})

	t.Run("Successfully downloads and saves dark logo", func(t *testing.T) {
		// Create mock WEBP content
		webpContent := []byte("fake-webp-content")

		//nolint:bodyclose
		webpResponse := testutils.NewMockResponse(http.StatusOK, string(webpContent))
		webpResponse.Header.Set("Content-Type", "image/webp")

		mockResponses := map[string]*http.Response{
			//nolint:bodyclose
			"https://example.com/dark-logo.webp": webpResponse,
		}
		httpClient := &http.Client{
			Transport: &testutils.MockRoundTripper{
				Responses: mockResponses,
			},
		}

		s := &OidcService{
			db:          db,
			fileStorage: dbStorage,
			httpClient:  httpClient,
		}

		// Download and save the dark logo
		err := s.downloadAndSaveLogoFromURL(t.Context(), client.ID, "https://example.com/dark-logo.webp", false)
		require.NoError(t, err)

		// Verify the dark logo file was saved
		darkLogoPath := "oidc-client-images/" + client.ID + "-dark.webp"
		require.True(t, fileExists(t, darkLogoPath), "Dark logo file should exist in storage")

		// Verify the content
		savedContent := getFileContent(t, darkLogoPath)
		assert.Equal(t, webpContent, savedContent)

		// Verify the client was updated
		var updatedClient model.OidcClient
		err = db.First(&updatedClient, "id = ?", client.ID).Error
		require.NoError(t, err)
		require.NotNil(t, updatedClient.DarkImageType)
		assert.Equal(t, "webp", *updatedClient.DarkImageType)
	})

	t.Run("Detects extension from URL path", func(t *testing.T) {
		svgContent := []byte("<svg></svg>")

		mockResponses := map[string]*http.Response{
			//nolint:bodyclose
			"https://example.com/icon.svg": testutils.NewMockResponse(http.StatusOK, string(svgContent)),
		}
		httpClient := &http.Client{
			Transport: &testutils.MockRoundTripper{
				Responses: mockResponses,
			},
		}

		s := &OidcService{
			db:          db,
			fileStorage: dbStorage,
			httpClient:  httpClient,
		}

		err := s.downloadAndSaveLogoFromURL(t.Context(), client.ID, "https://example.com/icon.svg", true)
		require.NoError(t, err)

		// Verify SVG file was saved
		logoPath := "oidc-client-images/" + client.ID + ".svg"
		require.True(t, fileExists(t, logoPath), "SVG logo should exist")
	})

	t.Run("Detects extension from Content-Type when path has no extension", func(t *testing.T) {
		jpgContent := []byte("fake-jpg-content")

		//nolint:bodyclose
		jpgResponse := testutils.NewMockResponse(http.StatusOK, string(jpgContent))
		jpgResponse.Header.Set("Content-Type", "image/jpeg")

		mockResponses := map[string]*http.Response{
			//nolint:bodyclose
			"https://example.com/logo": jpgResponse,
		}
		httpClient := &http.Client{
			Transport: &testutils.MockRoundTripper{
				Responses: mockResponses,
			},
		}

		s := &OidcService{
			db:          db,
			fileStorage: dbStorage,
			httpClient:  httpClient,
		}

		err := s.downloadAndSaveLogoFromURL(t.Context(), client.ID, "https://example.com/logo", true)
		require.NoError(t, err)

		// Verify JPG file was saved (jpeg extension is normalized to jpg)
		logoPath := "oidc-client-images/" + client.ID + ".jpg"
		require.True(t, fileExists(t, logoPath), "JPG logo should exist")
	})

	t.Run("Returns error for invalid URL", func(t *testing.T) {
		s := &OidcService{
			db:          db,
			fileStorage: dbStorage,
			httpClient:  &http.Client{},
		}

		err := s.downloadAndSaveLogoFromURL(t.Context(), client.ID, "://invalid-url", true)
		require.Error(t, err)
	})

	t.Run("Returns error for non-200 status code", func(t *testing.T) {
		mockResponses := map[string]*http.Response{
			//nolint:bodyclose
			"https://example.com/not-found.png": testutils.NewMockResponse(http.StatusNotFound, "Not Found"),
		}
		httpClient := &http.Client{
			Transport: &testutils.MockRoundTripper{
				Responses: mockResponses,
			},
		}

		s := &OidcService{
			db:          db,
			fileStorage: dbStorage,
			httpClient:  httpClient,
		}

		err := s.downloadAndSaveLogoFromURL(t.Context(), client.ID, "https://example.com/not-found.png", true)
		require.Error(t, err)
		require.ErrorContains(t, err, "failed to fetch logo")
	})

	t.Run("Returns error for too large content", func(t *testing.T) {
		// Create content larger than 2MB (maxLogoSize)
		largeContent := strings.Repeat("x", 2<<20+100) // 2.1MB

		//nolint:bodyclose
		largeResponse := testutils.NewMockResponse(http.StatusOK, largeContent)
		largeResponse.Header.Set("Content-Type", "image/png")
		largeResponse.Header.Set("Content-Length", strconv.Itoa(len(largeContent)))

		mockResponses := map[string]*http.Response{
			//nolint:bodyclose
			"https://example.com/large.png": largeResponse,
		}
		httpClient := &http.Client{
			Transport: &testutils.MockRoundTripper{
				Responses: mockResponses,
			},
		}

		s := &OidcService{
			db:          db,
			fileStorage: dbStorage,
			httpClient:  httpClient,
		}

		err := s.downloadAndSaveLogoFromURL(t.Context(), client.ID, "https://example.com/large.png", true)
		require.Error(t, err)
		require.ErrorIs(t, err, errLogoTooLarge)
	})

	t.Run("Returns error for unsupported file type", func(t *testing.T) {
		//nolint:bodyclose
		textResponse := testutils.NewMockResponse(http.StatusOK, "text content")
		textResponse.Header.Set("Content-Type", "text/plain")

		mockResponses := map[string]*http.Response{
			//nolint:bodyclose
			"https://example.com/file.txt": textResponse,
		}
		httpClient := &http.Client{
			Transport: &testutils.MockRoundTripper{
				Responses: mockResponses,
			},
		}

		s := &OidcService{
			db:          db,
			fileStorage: dbStorage,
			httpClient:  httpClient,
		}

		err := s.downloadAndSaveLogoFromURL(t.Context(), client.ID, "https://example.com/file.txt", true)
		require.Error(t, err)
		var fileTypeErr *common.FileTypeNotSupportedError
		require.ErrorAs(t, err, &fileTypeErr)
	})

	t.Run("Returns error for non-existent client", func(t *testing.T) {
		//nolint:bodyclose
		pngResponse := testutils.NewMockResponse(http.StatusOK, "content")
		pngResponse.Header.Set("Content-Type", "image/png")

		mockResponses := map[string]*http.Response{
			//nolint:bodyclose
			"https://example.com/logo.png": pngResponse,
		}
		httpClient := &http.Client{
			Transport: &testutils.MockRoundTripper{
				Responses: mockResponses,
			},
		}

		s := &OidcService{
			db:          db,
			fileStorage: dbStorage,
			httpClient:  httpClient,
		}

		err := s.downloadAndSaveLogoFromURL(t.Context(), "non-existent-client-id", "https://example.com/logo.png", true)
		require.Error(t, err)
		require.ErrorContains(t, err, "failed to look up client")
	})
}
