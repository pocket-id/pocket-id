package oidc

import (
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/lestrrat-go/jwx/v3/jwa"
	"github.com/ory/fosite"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	testutils "github.com/pocket-id/pocket-id/backend/internal/utils/testing"
	"github.com/stretchr/testify/require"
)

type testTokenSigner struct {
	key *rsa.PrivateKey
}

func (s testTokenSigner) GetPrivateKey() any {
	return s.key
}

func (s testTokenSigner) GetKeyAlg() (jwa.KeyAlgorithm, error) {
	return jwa.RS256(), nil
}

func (s testTokenSigner) GetKeyID() (string, bool) {
	return "test-key-id", true
}

func TestProviderIssuesJWTAccessTokens(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	signerKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	provider, err := newProvider(NewStore(db), nil, testTokenSigner{key: signerKey}, Config{ //nolint:gosec // static test-only provider secret
		BaseURL:      "https://issuer.example.com",
		TokenBaseURL: "https://issuer.example.com",
		Secret:       []byte("test-secret"),
	})
	require.NoError(t, err)

	session := NewEmptySession()
	session.Subject = "test-user"
	session.SetExpiresAt(fosite.AccessToken, time.Now().UTC().Add(time.Hour))

	request := fosite.NewAccessRequest(session)
	request.ID = "test-request"
	request.Client = Client{OidcClient: model.OidcClient{Base: model.Base{ID: "test-client"}}}
	request.GrantTypes = fosite.Arguments{string(fosite.GrantTypeClientCredentials)}
	request.RequestedScope = fosite.Arguments{"openid"}
	request.GrantedScope = fosite.Arguments{"openid"}
	request.RequestedAudience = fosite.Arguments{"test-client"}
	request.GrantedAudience = fosite.Arguments{"test-client"}

	response, err := provider.NewAccessResponse(t.Context(), request)
	require.NoError(t, err)
	require.Len(t, strings.Split(response.GetAccessToken(), "."), 3)

	// The issued JWT must carry a `kid` header matching the signing key so RPs can
	// select the verification key from the published JWKS (esp. after key rotation).
	header := decodeJWTPart(t, response.GetAccessToken(), 0)
	require.Equal(t, "test-key-id", header["kid"])
}

func TestProviderAcceptsWildcardRedirectURI(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	signerKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	require.NoError(t, db.Create(&model.OidcClient{
		Base:         model.Base{ID: "test-client"},
		Name:         "Test Client",
		CallbackURLs: model.UrlList{"https://*.example.com/callback"},
	}).Error)

	provider, err := newProvider(NewStore(db), nil, testTokenSigner{key: signerKey}, Config{ //nolint:gosec // static test-only provider secret
		BaseURL:      "https://issuer.example.com",
		TokenBaseURL: "https://issuer.example.com",
		Secret:       []byte("test-secret"),
	})
	require.NoError(t, err)

	const requestedRedirectURI = "https://tenant.example.com/callback"
	req := httptest.NewRequestWithContext(
		t.Context(),
		http.MethodGet,
		"/api/oidc/authorize?client_id=test-client&response_type=code&scope=openid&state=state-with-enough-entropy&redirect_uri="+requestedRedirectURI,
		nil,
	)

	ar, err := provider.NewAuthorizeRequest(req.Context(), req)
	require.NoError(t, err)
	require.Equal(t, requestedRedirectURI, ar.GetRedirectURI().String())
	require.True(t, ar.IsRedirectURIValid())
	require.NotContains(t, ar.GetClient().GetRedirectURIs(), requestedRedirectURI)
}

func TestProviderAcceptsPushedAuthorizationWildcardRedirectURI(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	signerKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	require.NoError(t, db.Create(&model.OidcClient{
		Base:         model.Base{ID: "test-client"},
		Name:         "Test Client",
		CallbackURLs: model.UrlList{"https://*.example.com/callback"},
		IsPublic:     true,
	}).Error)

	provider, err := newProvider(NewStore(db), nil, testTokenSigner{key: signerKey}, Config{ //nolint:gosec // static test-only provider secret
		BaseURL:      "https://issuer.example.com",
		TokenBaseURL: "https://issuer.example.com",
		Secret:       []byte("test-secret"),
	})
	require.NoError(t, err)

	const requestedRedirectURI = "https://tenant.example.com/callback"
	req := httptest.NewRequestWithContext(
		t.Context(),
		http.MethodPost,
		"/api/oidc/par?client_id=test-client&response_type=code&scope=openid&state=state-with-enough-entropy&redirect_uri="+requestedRedirectURI,
		nil,
	)

	ar, err := provider.NewPushedAuthorizeRequest(req.Context(), req)
	require.NoError(t, err)
	require.Equal(t, requestedRedirectURI, ar.GetRedirectURI().String())
	require.True(t, ar.IsRedirectURIValid())
	require.NotContains(t, ar.GetClient().GetRedirectURIs(), requestedRedirectURI)
}

func TestProviderRejectsUnmatchedWildcardRedirectURI(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	signerKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	require.NoError(t, db.Create(&model.OidcClient{
		Base:         model.Base{ID: "test-client"},
		Name:         "Test Client",
		CallbackURLs: model.UrlList{"https://*.example.com/callback"},
	}).Error)

	provider, err := newProvider(NewStore(db), nil, testTokenSigner{key: signerKey}, Config{ //nolint:gosec // static test-only provider secret
		BaseURL:      "https://issuer.example.com",
		TokenBaseURL: "https://issuer.example.com",
		Secret:       []byte("test-secret"),
	})
	require.NoError(t, err)

	const requestedRedirectURI = "https://evil.example.net/callback"
	req := httptest.NewRequestWithContext(
		t.Context(),
		http.MethodGet,
		"/api/oidc/authorize?client_id=test-client&response_type=code&scope=openid&state=state-with-enough-entropy&redirect_uri="+requestedRedirectURI,
		nil,
	)

	_, err = provider.NewAuthorizeRequest(req.Context(), req)
	require.ErrorIs(t, err, fosite.ErrInvalidRequest)
}

// decodeJWTPart base64url-decodes the header (index 0) or claims (index 1) segment of a
// JWT without verifying the signature, for assertions in tests.
func decodeJWTPart(t *testing.T, token string, index int) map[string]any {
	t.Helper()
	parts := strings.Split(token, ".")
	require.Len(t, parts, 3)
	raw, err := base64.RawURLEncoding.DecodeString(parts[index])
	require.NoError(t, err)
	var out map[string]any
	require.NoError(t, json.Unmarshal(raw, &out))
	return out
}

type algTestSigner struct {
	key any
	alg jwa.KeyAlgorithm
}

func (s algTestSigner) GetPrivateKey() any                   { return s.key }
func (s algTestSigner) GetKeyAlg() (jwa.KeyAlgorithm, error) { return s.alg, nil }
func (s algTestSigner) GetKeyID() (string, bool)             { return "test-key-id", true }

// TestProviderIssuesAndValidatesTokensForSupportedAlgorithms guards against the
// regression where fosite's DefaultSigner derived the JWT algorithm from the Go key type
// alone: it broke EdDSA/ES384/ES512 token issuance entirely and silently downgraded
// RS384/RS512 to RS256. The provider must both ISSUE (NewAccessResponse) and VALIDATE
// (IntrospectToken -> Signer.Decode, the path used by introspection and userinfo) tokens
// for every signing algorithm Pocket ID supports.
func TestProviderIssuesAndValidatesTokensForSupportedAlgorithms(t *testing.T) {
	cases := []struct {
		name string
		alg  jwa.KeyAlgorithm
		gen  func(t *testing.T) any
	}{
		{"RS256", jwa.RS256(), generateRSATestKey},
		{"RS384", jwa.RS384(), generateRSATestKey},
		{"RS512", jwa.RS512(), generateRSATestKey},
		{"ES256", jwa.ES256(), func(t *testing.T) any { return generateECTestKey(t, elliptic.P256()) }},
		{"ES384", jwa.ES384(), func(t *testing.T) any { return generateECTestKey(t, elliptic.P384()) }},
		{"ES512", jwa.ES512(), func(t *testing.T) any { return generateECTestKey(t, elliptic.P521()) }},
		{"EdDSA", jwa.EdDSA(), generateEd25519TestKey},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			db := testutils.NewDatabaseForTest(t)
			require.NoError(t, db.Create(&model.OidcClient{Base: model.Base{ID: "test-client"}, Name: "Test Client"}).Error)

			provider, err := newProvider(NewStore(db), nil, algTestSigner{key: tc.gen(t), alg: tc.alg}, Config{ //nolint:gosec // static test-only provider secret
				BaseURL:      "https://issuer.example.com",
				TokenBaseURL: "https://issuer.example.com",
				Secret:       []byte("test-secret"),
			})
			require.NoError(t, err)

			session := NewEmptySession()
			session.Subject = "test-user"
			session.SetExpiresAt(fosite.AccessToken, time.Now().UTC().Add(time.Hour))

			request := fosite.NewAccessRequest(session)
			request.ID = "test-request-" + tc.name
			request.Client = Client{OidcClient: model.OidcClient{Base: model.Base{ID: "test-client"}}}
			request.GrantTypes = fosite.Arguments{string(fosite.GrantTypeClientCredentials)}
			request.RequestedScope = fosite.Arguments{"openid"}
			request.GrantedScope = fosite.Arguments{"openid"}
			request.RequestedAudience = fosite.Arguments{"test-client"}
			request.GrantedAudience = fosite.Arguments{"test-client"}

			response, err := provider.NewAccessResponse(t.Context(), request)
			require.NoError(t, err)

			accessToken := response.GetAccessToken()
			require.Len(t, strings.Split(accessToken, "."), 3)
			header := decodeJWTPart(t, accessToken, 0)
			require.Equal(t, tc.alg.String(), header["alg"])

			tokenUse, introspected, err := provider.IntrospectToken(t.Context(), accessToken, fosite.AccessToken, NewEmptySession())
			require.NoError(t, err)
			require.Equal(t, fosite.AccessToken, tokenUse)
			require.Equal(t, "test-client", introspected.GetClient().GetID())
		})
	}
}

func generateRSATestKey(t *testing.T) any {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	return key
}

func generateECTestKey(t *testing.T, curve elliptic.Curve) any {
	t.Helper()
	key, err := ecdsa.GenerateKey(curve, rand.Reader)
	require.NoError(t, err)
	return key
}

func generateEd25519TestKey(t *testing.T) any {
	t.Helper()
	_, key, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	return key
}
