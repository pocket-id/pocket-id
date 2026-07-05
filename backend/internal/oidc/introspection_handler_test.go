package oidc

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lestrrat-go/jwx/v3/jwa"
	"github.com/lestrrat-go/jwx/v3/jwk"
	"github.com/lestrrat-go/jwx/v3/jwt"
	"github.com/ory/fosite"
	"github.com/stretchr/testify/require"

	"github.com/pocket-id/pocket-id/backend/internal/model"
	jwkutils "github.com/pocket-id/pocket-id/backend/internal/utils/jwk"
	testutils "github.com/pocket-id/pocket-id/backend/internal/utils/testing"
)

// TestIntrospectionHandlerBindsTokenToCallerClient verifies that an RFC 6750 bearer-auth
// caller (which omits client_id) can only introspect its own client's tokens. A client
// must not be able to introspect another client's token — doing so would leak the token's
// validity and the user PII it carries.
func TestIntrospectionHandlerBindsTokenToCallerClient(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db := testutils.NewDatabaseForTest(t)
	require.NoError(t, db.Create(&model.OidcClient{Base: model.Base{ID: "client-a"}, Name: "Client A"}).Error)
	require.NoError(t, db.Create(&model.OidcClient{Base: model.Base{ID: "client-b"}, Name: "Client B"}).Error)

	signerKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	provider, err := newProvider(NewStore(db, nil), nil, testTokenSigner{key: signerKey}, Config{ //nolint:gosec // static test-only provider secret
		BaseURL:      "https://issuer.example.com",
		TokenBaseURL: "https://issuer.example.com",
		Secret:       []byte("test-secret"),
	})
	require.NoError(t, err)

	issueAccessToken := func(t *testing.T, requestID, clientID, subject string) string {
		t.Helper()
		session := NewEmptySession()
		session.Subject = subject
		session.SetExpiresAt(fosite.AccessToken, time.Now().UTC().Add(time.Hour))

		request := fosite.NewAccessRequest(session)
		request.ID = requestID
		request.Client = Client{OidcClient: model.OidcClient{Base: model.Base{ID: clientID}}}
		request.GrantTypes = fosite.Arguments{string(fosite.GrantTypeClientCredentials)}
		request.RequestedScope = fosite.Arguments{"openid"}
		request.GrantedScope = fosite.Arguments{"openid"}
		request.RequestedAudience = fosite.Arguments{clientID}
		request.GrantedAudience = fosite.Arguments{clientID}

		response, err := provider.NewAccessResponse(t.Context(), request)
		require.NoError(t, err)
		return response.GetAccessToken()
	}

	clientAToken := issueAccessToken(t, "req-a", "client-a", "user-a")
	clientBToken := issueAccessToken(t, "req-b", "client-b", "user-b")
	clientBOtherToken := issueAccessToken(t, "req-b-2", "client-b", "user-b")

	handler := newIntrospectionHandler(provider, nil, "https://issuer.example.com")

	introspect := func(t *testing.T, bearer, token string) map[string]any {
		t.Helper()
		body := url.Values{"token": {token}}.Encode()
		req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/oidc/introspect", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Authorization", "Bearer "+bearer)

		rec := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(rec)
		c.Request = req

		handler.introspectToken(c)

		var out map[string]any
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &out))
		return out
	}

	// A client using its own token as bearer must not learn about another client's token.
	cross := introspect(t, clientBToken, clientAToken)
	require.Equal(t, false, cross["active"])

	// A client may introspect another of its own tokens.
	same := introspect(t, clientBToken, clientBOtherToken)
	require.Equal(t, true, same["active"])
}

func TestIntrospectionHandlerAllowsReusedFederatedClientAssertion(t *testing.T) {
	gin.SetMode(gin.TestMode)

	const (
		baseURL  = "https://issuer.example.com"
		issuer   = "https://idp.example.com"
		clientID = "federated-client"
		subject  = "external-workload"
		audience = "api://pocket-id"
		jwksURL  = "https://idp.example.com/jwks.json"
	)

	signingKey, err := jwkutils.GenerateKey(jwa.RS256().String(), "")
	require.NoError(t, err)
	signingAlg, ok := signingKey.Algorithm()
	require.True(t, ok)

	publicKey, err := signingKey.PublicKey()
	require.NoError(t, err)
	jwks := jwk.NewSet()
	require.NoError(t, jwks.AddKey(publicKey))

	db := testutils.NewDatabaseForTest(t)
	replayProtection := false
	require.NoError(t, db.Create(&model.OidcClient{
		Base: model.Base{ID: clientID},
		Name: "Federated Client",
		Credentials: model.OidcClientCredentials{
			FederatedIdentities: []model.OidcClientFederatedIdentity{{
				Issuer:           issuer,
				Subject:          subject,
				Audience:         audience,
				JWKS:             jwksURL,
				ReplayProtection: replayProtection,
			}},
		},
	}).Error)

	store := NewStore(db, nil)
	authenticator, err := newFederatedClientAuthenticator(t.Context(), store, newJWKSetHTTPClient(t, jwks), baseURL)
	require.NoError(t, err)

	signerKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	provider, err := newProvider(store, authenticator, testTokenSigner{key: signerKey}, Config{ //nolint:gosec // static test-only provider secret
		BaseURL:      baseURL,
		TokenBaseURL: baseURL,
		Secret:       []byte("test-secret"),
	})
	require.NoError(t, err)

	session := NewEmptySession()
	session.Subject = "user-a"
	session.SetExpiresAt(fosite.AccessToken, time.Now().UTC().Add(time.Hour))

	accessRequest := fosite.NewAccessRequest(session)
	accessRequest.ID = "replay-test-request"
	accessRequest.Client = Client{OidcClient: model.OidcClient{Base: model.Base{ID: clientID}}}
	accessRequest.GrantTypes = fosite.Arguments{string(fosite.GrantTypeClientCredentials)}
	accessRequest.RequestedScope = fosite.Arguments{"openid"}
	accessRequest.GrantedScope = fosite.Arguments{"openid"}
	accessRequest.RequestedAudience = fosite.Arguments{clientID}
	accessRequest.GrantedAudience = fosite.Arguments{clientID}

	accessResponse, err := provider.NewAccessResponse(t.Context(), accessRequest)
	require.NoError(t, err)

	assertionToken, err := jwt.NewBuilder().
		Issuer(issuer).
		Subject(subject).
		Audience([]string{audience}).
		IssuedAt(time.Now()).
		Expiration(time.Now().Add(5 * time.Minute)).
		JwtID("cached-provider-token").
		Build()
	require.NoError(t, err)
	signedAssertion, err := jwt.Sign(assertionToken, jwt.WithKey(signingAlg, signingKey))
	require.NoError(t, err)

	handler := newIntrospectionHandler(provider, authenticator, baseURL)
	introspect := func(t *testing.T) (int, map[string]any) {
		t.Helper()
		body := url.Values{
			"client_id": {clientID},
			"token":     {accessResponse.GetAccessToken()},
		}.Encode()
		req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/oidc/introspect", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Authorization", "Bearer "+string(signedAssertion))

		rec := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(rec)
		c.Request = req
		handler.introspectToken(c)

		var out map[string]any
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &out))
		return rec.Code, out
	}

	status, body := introspect(t)
	require.Equal(t, http.StatusOK, status)
	require.Equal(t, true, body["active"])

	status, body = introspect(t)
	require.Equal(t, http.StatusOK, status)
	require.Equal(t, true, body["active"])
}
