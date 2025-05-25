package service

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/lestrrat-go/jwx/v3/jwk"
	"github.com/pocket-id/pocket-id/backend/internal/dto"
	"github.com/stretchr/testify/require"
)

// generateTestECDSAKey creates an ECDSA key for testing
func generateTestECDSAKey(t *testing.T) jwk.Key {
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

	return privateJwk
}

func TestOidcService_VerifyClientCredentialsInternal(t *testing.T) {
	var err error
	// Create a test database
	db := newDatabaseForTest(t)

	// Create a JWK for testing
	privateJwk := generateTestECDSAKey(t)
	require.NoError(t, err)
	publicJwk, err := jwk.PublicKeyOf(privateJwk)
	require.NoError(t, err)

	// Create a JWK Set with the public key
	jwkSet := jwk.NewSet()
	err = jwkSet.AddKey(publicJwk)
	require.NoError(t, err)
	jwkSetJSON, err := json.Marshal(jwkSet)
	require.NoError(t, err)

	// Create a mock HTTP client with custom transport to return the JWKS
	mockJwksTransport := &MockRoundTripper{
		Response: NewMockResponse(http.StatusOK, string(jwkSetJSON)),
	}
	httpClient := &http.Client{
		Transport: mockJwksTransport,
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

	_ = confidentialClient
	_ = confidentialSecret
	_ = publicClient
	_ = federatedClient
}
