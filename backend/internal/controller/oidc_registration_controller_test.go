//go:build unit

package controller

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/ory/fosite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pocket-id/pocket-id/backend/internal/appconfig"
	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/service"
	testutils "github.com/pocket-id/pocket-id/backend/internal/utils/testing"
)

func newTestRegistrationRouter(t *testing.T) *gin.Engine {
	t.Helper()
	return newTestRegistrationRouterWithAllowlist(t, `["https://app.example.com/**"]`)
}

func newTestRegistrationRouterWithAllowlist(t *testing.T, allowlist string) *gin.Engine {
	t.Helper()

	gin.SetMode(gin.TestMode)

	db := testutils.NewDatabaseForTest(t)

	cfg := appconfig.NewTestConfig(nil)
	cfg.DynamicClientRedirectUriAllowlist = appconfig.AppConfigValue(allowlist)
	appConfigService := appconfig.NewTestAppConfigService(cfg)

	svc, err := service.NewOidcService(db, nil, appConfigService, nil, nil, nil, nil, nil)
	require.NoError(t, err)

	router := gin.New()
	group := router.Group("/api")
	// Rate limiting is exercised by the middleware's own tests; these use a no-op
	// so the handler behavior under test is not throttled.
	noRateLimit := func(c *gin.Context) { c.Next() }
	NewOidcRegistrationController(group, svc, noRateLimit, noRateLimit)
	return router
}

func TestRegistrationEndpoint(t *testing.T) {
	original := common.EnvConfig
	t.Cleanup(func() { common.EnvConfig = original })
	common.EnvConfig.UiConfigDisabled = true

	router := newTestRegistrationRouter(t)

	post := func(t *testing.T, path, body string) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		return w
	}

	t.Run("registers a client when DCR is enabled", func(t *testing.T) {
		body := `{"redirect_uris":["https://app.example.com/cb"],"client_name":"C","token_endpoint_auth_method":"client_secret_basic"}`
		resp := post(t, "/api/oidc/register", body)
		require.Equal(t, http.StatusCreated, resp.Code)
		var doc fosite.ClientRegistrationResponse
		require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &doc))
		assert.NotEmpty(t, doc.ClientID)
		assert.NotEmpty(t, doc.ClientSecret)
		assert.NotEmpty(t, doc.RegistrationAccessToken)
		assert.Contains(t, doc.RegistrationClientURI, doc.ClientID)

		// Synthesized DCR response metadata: derived from the stored model,
		// not persisted verbatim from the request.
		assert.Equal(t, "client_secret_basic", doc.TokenEndpointAuthMethod)
		assert.ElementsMatch(t, []string{"authorization_code", "refresh_token"}, doc.GrantTypes)
		assert.Contains(t, doc.ResponseTypes, "code")
		assert.Equal(t, []string{"https://app.example.com/cb"}, doc.RedirectURIs)
	})

	t.Run("registers a public client with token_endpoint_auth_method none", func(t *testing.T) {
		body := `{"redirect_uris":["https://app.example.com/cb"],"client_name":"Public","token_endpoint_auth_method":"none"}`
		resp := post(t, "/api/oidc/register", body)
		require.Equal(t, http.StatusCreated, resp.Code)
		var doc fosite.ClientRegistrationResponse
		require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &doc))
		assert.Equal(t, "none", doc.TokenEndpointAuthMethod)
		assert.ElementsMatch(t, []string{"authorization_code", "refresh_token"}, doc.GrantTypes)
		assert.Contains(t, doc.ResponseTypes, "code")
	})

	t.Run("logo_uri pointing at a private/loopback address does not fail registration", func(t *testing.T) {
		body := `{"redirect_uris":["https://app.example.com/cb"],"client_name":"C","token_endpoint_auth_method":"client_secret_basic","logo_uri":"http://127.0.0.1/logo.png"}`
		resp := post(t, "/api/oidc/register", body)
		require.Equal(t, http.StatusCreated, resp.Code)
		var doc fosite.ClientRegistrationResponse
		require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &doc))
		assert.NotEmpty(t, doc.ClientID)
	})

	// An empty allowlist is the disabled-by-default state: there is no separate
	// feature flag, so registration must be refused until an administrator opts in.
	t.Run("rejects every registration when the allowlist is empty", func(t *testing.T) {
		emptyRouter := newTestRegistrationRouterWithAllowlist(t, `[]`)
		req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/oidc/register",
			strings.NewReader(`{"redirect_uris":["https://app.example.com/cb"]}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		emptyRouter.ServeHTTP(w, req)
		require.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "invalid_redirect_uri")
	})

	t.Run("rejects a redirect URI outside the allowlist", func(t *testing.T) {
		resp := post(t, "/api/oidc/register", `{"redirect_uris":["https://evil.example.com/cb"]}`)
		require.Equal(t, http.StatusBadRequest, resp.Code)
	})
}

// TestRegistrationClientConfigurationEndpoint exercises the RFC 7592 client
// configuration endpoints (GET/PUT/DELETE /api/oidc/register/:id) at the HTTP
// layer, including the status-code regressions from the final review: a PUT
// with an out-of-allowlist redirect URI must return 400 (not 401), and a PUT
// that moves a public client to a confidential auth method must return a
// non-empty client_secret.
func TestRegistrationClientConfigurationEndpoint(t *testing.T) {
	original := common.EnvConfig
	t.Cleanup(func() { common.EnvConfig = original })
	common.EnvConfig.UiConfigDisabled = true

	router := newTestRegistrationRouter(t)

	do := func(t *testing.T, method, path, token, body string) *httptest.ResponseRecorder {
		t.Helper()
		var reader *strings.Reader
		if body != "" {
			reader = strings.NewReader(body)
		} else {
			reader = strings.NewReader("")
		}
		req := httptest.NewRequestWithContext(t.Context(), method, path, reader)
		req.Header.Set("Content-Type", "application/json")
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		return w
	}

	register := func(t *testing.T, body string) fosite.ClientRegistrationResponse {
		t.Helper()
		req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/oidc/register", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		require.Equal(t, http.StatusCreated, w.Code)
		var doc fosite.ClientRegistrationResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &doc))
		return doc
	}

	// Register a confidential client for GET/PUT/DELETE happy-path assertions.
	confidential := register(t, `{"redirect_uris":["https://app.example.com/cb"],"client_name":"C","token_endpoint_auth_method":"client_secret_basic"}`)
	path := "/api/oidc/register/" + confidential.ClientID

	// A successful read or update rotates the registration access token (RFC 7592
	// section 2.1 and appendix A.1), so the subtests below thread the newest token
	// forward rather than reusing the one issued at registration.
	currentToken := confidential.RegistrationAccessToken

	t.Run("GET with a valid bearer token returns the client", func(t *testing.T) {
		resp := do(t, http.MethodGet, path, currentToken, "")
		require.Equal(t, http.StatusOK, resp.Code)
		var doc fosite.ClientRegistrationResponse
		require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &doc))
		assert.Equal(t, confidential.ClientID, doc.ClientID)

		// RFC 7592 section 3 requires the response to carry a registration access
		// token; it is rotated on read, so it differs from the previous one.
		require.NotEmpty(t, doc.RegistrationAccessToken)
		assert.NotEqual(t, currentToken, doc.RegistrationAccessToken)
		assert.NotEmpty(t, doc.RegistrationClientURI)
		currentToken = doc.RegistrationAccessToken

		// Synthesized DCR response metadata, consistent with the register response.
		assert.Equal(t, "client_secret_basic", doc.TokenEndpointAuthMethod)
		assert.ElementsMatch(t, []string{"authorization_code", "refresh_token"}, doc.GrantTypes)
		assert.Contains(t, doc.ResponseTypes, "code")
		assert.Equal(t, []string{"https://app.example.com/cb"}, doc.RedirectURIs)
	})

	t.Run("GET with a wrong bearer token returns 401", func(t *testing.T) {
		resp := do(t, http.MethodGet, path, "wrong-token", "")
		require.Equal(t, http.StatusUnauthorized, resp.Code)
	})

	t.Run("GET with no bearer token returns 401", func(t *testing.T) {
		resp := do(t, http.MethodGet, path, "", "")
		require.Equal(t, http.StatusUnauthorized, resp.Code)
	})

	t.Run("PUT with a redirect URI outside the allowlist returns 400", func(t *testing.T) {
		resp := do(t, http.MethodPut, path, currentToken,
			`{"redirect_uris":["https://evil.example.com/cb"],"client_name":"C","token_endpoint_auth_method":"client_secret_basic"}`)
		require.Equal(t, http.StatusBadRequest, resp.Code)
		var body map[string]any
		require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &body))
		assert.Equal(t, "invalid_redirect_uri", body["error"])
	})

	t.Run("PUT switching a public client to client_secret_basic returns a new client_secret", func(t *testing.T) {
		public := register(t, `{"redirect_uris":["https://app.example.com/cb"],"client_name":"Public","token_endpoint_auth_method":"none"}`)
		require.Empty(t, public.ClientSecret)
		assert.Equal(t, "none", public.TokenEndpointAuthMethod)

		publicPath := "/api/oidc/register/" + public.ClientID
		resp := do(t, http.MethodPut, publicPath, public.RegistrationAccessToken,
			`{"redirect_uris":["https://app.example.com/cb"],"client_name":"Public","token_endpoint_auth_method":"client_secret_basic"}`)
		require.Equal(t, http.StatusOK, resp.Code)
		var doc fosite.ClientRegistrationResponse
		require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &doc))
		assert.NotEmpty(t, doc.ClientSecret)

		// The update response must also carry a registration access token, rotated
		// away from the one issued at registration.
		require.NotEmpty(t, doc.RegistrationAccessToken)
		assert.NotEqual(t, public.RegistrationAccessToken, doc.RegistrationAccessToken)

		// Synthesized DCR response metadata reflects the post-update state
		// (now confidential), consistent with register/GET.
		assert.Equal(t, "client_secret_basic", doc.TokenEndpointAuthMethod)
		assert.ElementsMatch(t, []string{"authorization_code", "refresh_token"}, doc.GrantTypes)
		assert.Contains(t, doc.ResponseTypes, "code")
		assert.Equal(t, []string{"https://app.example.com/cb"}, doc.RedirectURIs)
	})

	t.Run("DELETE with a valid token removes the client, subsequent GET returns 401", func(t *testing.T) {
		resp := do(t, http.MethodDelete, path, currentToken, "")
		require.Equal(t, http.StatusNoContent, resp.Code)

		getResp := do(t, http.MethodGet, path, currentToken, "")
		require.Equal(t, http.StatusUnauthorized, getResp.Code)
	})
}

// TestRegistrationResponseIncludesSecretExpiry pins the RFC 7591 section 3.2.1 rule that
// client_secret_expires_at is present whenever a client_secret is issued.
//
// 0 means "never expires", which is exactly the value a naive `omitempty` on an int
// silently drops, so this asserts on the decoded JSON rather than on a Go struct field.
// A public client is issued no secret, so the member must be absent there.
func TestRegistrationResponseIncludesSecretExpiry(t *testing.T) {
	original := common.EnvConfig
	t.Cleanup(func() { common.EnvConfig = original })
	common.EnvConfig.UiConfigDisabled = true

	router := newTestRegistrationRouter(t)

	register := func(t *testing.T, body string) map[string]any {
		t.Helper()
		req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/oidc/register", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
		var out map[string]any
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &out))
		return out
	}

	t.Run("a confidential client is told its secret never expires", func(t *testing.T) {
		out := register(t, `{"redirect_uris":["https://app.example.com/cb"],"token_endpoint_auth_method":"client_secret_basic"}`)
		require.NotEmpty(t, out["client_secret"])
		require.Contains(t, out, "client_secret_expires_at",
			"RFC 7591 requires client_secret_expires_at whenever a secret is issued")
		// Compared against the decoded JSON number, which must be exactly 0: that is
		// the RFC 7591 encoding of "this secret never expires".
		assert.Zero(t, out["client_secret_expires_at"])
	})

	t.Run("a public client gets no secret and no expiry", func(t *testing.T) {
		out := register(t, `{"redirect_uris":["https://app.example.com/cb"],"token_endpoint_auth_method":"none"}`)
		assert.NotContains(t, out, "client_secret")
		assert.NotContains(t, out, "client_secret_expires_at")
	})
}

// TestRegistrationAccessTokenSchemeIsCaseInsensitive pins RFC 7235 section 2.1: the
// Authorization auth-scheme is case-insensitive, so a client sending "bearer" must be
// accepted exactly like one sending "Bearer". It also pins RFC 6750 section 3, which
// requires a 401 from a bearer-protected resource to carry a WWW-Authenticate challenge.
func TestRegistrationAccessTokenSchemeIsCaseInsensitive(t *testing.T) {
	original := common.EnvConfig
	t.Cleanup(func() { common.EnvConfig = original })
	common.EnvConfig.UiConfigDisabled = true

	router := newTestRegistrationRouter(t)

	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/oidc/register",
		strings.NewReader(`{"redirect_uris":["https://app.example.com/cb"]}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())

	var registered map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &registered))
	clientID, _ := registered["client_id"].(string)
	token, _ := registered["registration_access_token"].(string)
	require.NotEmpty(t, clientID)
	require.NotEmpty(t, token)

	// A successful read rotates the registration access token, so each iteration
	// authenticates with the token returned by the previous one.
	for _, scheme := range []string{"Bearer", "bearer", "BEARER", "BeArEr"} {
		t.Run("scheme "+scheme, func(t *testing.T) {
			req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/oidc/register/"+clientID, nil)
			req.Header.Set("Authorization", scheme+" "+token)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)
			require.Equal(t, http.StatusOK, rec.Code, "the auth-scheme is case-insensitive")

			var doc map[string]any
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &doc))
			next, _ := doc["registration_access_token"].(string)
			require.NotEmpty(t, next)
			token = next
		})
	}

	t.Run("a rejected token gets a WWW-Authenticate challenge", func(t *testing.T) {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/oidc/register/"+clientID, nil)
		req.Header.Set("Authorization", "Bearer not-the-right-token")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		require.Equal(t, http.StatusUnauthorized, rec.Code)
		assert.Contains(t, rec.Header().Get("WWW-Authenticate"), "Bearer",
			"RFC 6750 requires a challenge on a 401 from a bearer-protected resource")
	})
}
