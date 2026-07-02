package oidc

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ory/fosite"
	"github.com/stretchr/testify/require"

	"github.com/pocket-id/pocket-id/backend/internal/model"
	testutils "github.com/pocket-id/pocket-id/backend/internal/utils/testing"
)

// TestUserInfoHandler covers the userinfo endpoint, which returns user PII based purely on
// a presented access token. It must reject a missing/garbage token and — importantly — a
// token that carries no resource owner (e.g. a client_credentials token), so machine
// tokens cannot be exchanged for a user's profile. A valid user access token returns the
// claims for exactly the scopes it was granted.
func TestUserInfoHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	const (
		baseURL  = "https://issuer.example.com"
		userID   = "user-1"
		clientID = "client-1"
	)

	db := testutils.NewDatabaseForTest(t)
	require.NoError(t, db.Create(&model.OidcClient{Base: model.Base{ID: clientID}, Name: "Test Client"}).Error)
	require.NoError(t, db.Create(&model.User{
		Base:          model.Base{ID: userID},
		Username:      "tim",
		FirstName:     "Tim",
		LastName:      "Cook",
		DisplayName:   "Tim Cook",
		Email:         stringPointer("tim@example.com"),
		EmailVerified: true,
	}).Error)

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	provider, err := newProvider(NewStore(db, nil), nil, testTokenSigner{key: key}, Config{
		BaseURL:      baseURL,
		TokenBaseURL: baseURL,
		Secret:       "test-secret",
	})
	require.NoError(t, err)

	handler := newUserInfoHandler(provider, newClaimsService(db, nil, baseURL, nil))

	issueAccessToken := func(t *testing.T, requestID, subject string, scopes ...string) string {
		t.Helper()
		session := NewEmptySession()
		session.Subject = subject
		session.SetExpiresAt(fosite.AccessToken, time.Now().UTC().Add(time.Hour))

		request := fosite.NewAccessRequest(session)
		request.ID = requestID
		request.Client = Client{OidcClient: model.OidcClient{Base: model.Base{ID: clientID}}}
		request.GrantTypes = fosite.Arguments{string(fosite.GrantTypeClientCredentials)}
		request.RequestedScope = fosite.Arguments(scopes)
		request.GrantedScope = fosite.Arguments(scopes)
		// A login token is audienced to the requesting client; userinfo gates on the openid scope, not the audience
		request.RequestedAudience = fosite.Arguments{clientID}
		request.GrantedAudience = fosite.Arguments{clientID}

		response, err := provider.NewAccessResponse(t.Context(), request)
		require.NoError(t, err)
		return response.GetAccessToken()
	}

	call := func(t *testing.T, token string) (*httptest.ResponseRecorder, *gin.Context) {
		t.Helper()
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/oidc/userinfo", nil)
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		rec := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(rec)
		c.Request = req
		handler.userInfo(c)
		return rec, c
	}

	t.Run("valid user access token returns the granted claims", func(t *testing.T) {
		token := issueAccessToken(t, "req-valid", userID, "openid", "email")
		rec, c := call(t, token)

		require.Empty(t, c.Errors)
		require.Equal(t, http.StatusOK, rec.Code)

		var claims map[string]any
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &claims))
		require.Equal(t, userID, claims["sub"])
		require.Equal(t, "tim@example.com", claims["email"])
		// profile was not granted, so profile claims must be absent
		require.NotContains(t, claims, "given_name")
	})

	t.Run("token without the openid scope is rejected", func(t *testing.T) {
		// A token issued purely for a custom API carries no openid scope and must not read profile claims
		session := NewEmptySession()
		session.Subject = userID
		session.SetExpiresAt(fosite.AccessToken, time.Now().UTC().Add(time.Hour))

		request := fosite.NewAccessRequest(session)
		request.ID = "req-no-openid"
		request.Client = Client{OidcClient: model.OidcClient{Base: model.Base{ID: clientID}}}
		request.GrantTypes = fosite.Arguments{string(fosite.GrantTypeClientCredentials)}
		request.RequestedScope = fosite.Arguments{"read:orders"}
		request.GrantedScope = fosite.Arguments{"read:orders"}
		request.RequestedAudience = fosite.Arguments{"https://api.example.com"}
		request.GrantedAudience = fosite.Arguments{"https://api.example.com"}

		response, err := provider.NewAccessResponse(t.Context(), request)
		require.NoError(t, err)

		rec, c := call(t, response.GetAccessToken())
		require.Empty(t, c.Errors)
		require.Equal(t, http.StatusForbidden, rec.Code)
	})

	t.Run("token whose user no longer exists is rejected with 401", func(t *testing.T) {
		// A valid token whose subject was deleted is an auth failure, not a 404
		token := issueAccessToken(t, "req-ghost", "ghost-user", "openid")
		rec, c := call(t, token)
		require.Empty(t, c.Errors)
		require.Equal(t, http.StatusUnauthorized, rec.Code)
		require.Contains(t, rec.Header().Get("WWW-Authenticate"), `Bearer error="request_unauthorized"`)
	})

	t.Run("missing access token is rejected", func(t *testing.T) {
		rec, c := call(t, "")
		require.Empty(t, c.Errors)
		require.Equal(t, http.StatusUnauthorized, rec.Code)
		require.Contains(t, rec.Header().Get("WWW-Authenticate"), `Bearer error="request_unauthorized"`)
	})

	t.Run("garbage access token is rejected", func(t *testing.T) {
		rec, c := call(t, "garbage.token.value")
		require.Empty(t, c.Errors)
		require.Equal(t, http.StatusUnauthorized, rec.Code)
		require.Contains(t, rec.Header().Get("WWW-Authenticate"), `Bearer error=`)
	})

	t.Run("token without a resource owner is rejected", func(t *testing.T) {
		// client_credentials-style token: valid, but no subject -> must not return PII.
		token := issueAccessToken(t, "req-no-subject", "", "openid")
		rec, c := call(t, token)
		require.Empty(t, c.Errors)
		require.Equal(t, http.StatusUnauthorized, rec.Code)
		require.Contains(t, rec.Header().Get("WWW-Authenticate"), `Bearer error="request_unauthorized"`)
	})
}
