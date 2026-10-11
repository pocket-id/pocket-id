package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func newCrossOriginProtectionTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(NewErrorHandlerMiddleware().Add())
	router.Use(NewCrossOriginProtectionMiddleware("https://id.example.com").Add())

	ok := func(c *gin.Context) { c.Status(http.StatusOK) }
	router.GET("/api/oidc/device/verify", ok)
	router.POST("/api/oidc/device/verify", ok)
	router.DELETE("/api/users/:id", ok)
	router.POST("/authorize", ok)
	router.POST("/api/oidc/token", ok)
	router.POST("/api/oidc/userinfo", ok)
	router.POST("/api/oidc/introspect", ok)
	router.POST("/api/oidc/end-session", ok)
	router.POST("/api/oidc/par", ok)
	router.POST("/api/oidc/device/authorize", ok)

	return router
}

func TestCrossOriginProtectionMiddleware(t *testing.T) {
	router := newCrossOriginProtectionTestRouter()

	type testCase struct {
		name    string
		method  string
		path    string
		headers map[string]string
		want    int
	}

	tests := []testCase{
		// Browser requests from a sibling origin on the same site carry SameSite=Lax cookies, so they must be rejected
		{"same-site sibling origin", http.MethodPost, "/api/oidc/device/verify?code=ABCD", map[string]string{"Sec-Fetch-Site": "same-site", "Origin": "https://app.example.com"}, http.StatusForbidden},
		{"cross-site origin", http.MethodPost, "/api/oidc/device/verify?code=ABCD", map[string]string{"Sec-Fetch-Site": "cross-site", "Origin": "https://evil.test"}, http.StatusForbidden},
		{"cross-site delete", http.MethodDelete, "/api/users/1", map[string]string{"Sec-Fetch-Site": "cross-site"}, http.StatusForbidden},
		{"old browser with foreign origin", http.MethodPost, "/api/oidc/device/verify", map[string]string{"Origin": "https://app.example.com"}, http.StatusForbidden},

		// Requests from the SPA itself and direct navigations are allowed
		{"same origin", http.MethodPost, "/api/oidc/device/verify", map[string]string{"Sec-Fetch-Site": "same-origin", "Origin": "https://id.example.com"}, http.StatusOK},
		{"user-initiated navigation", http.MethodPost, "/api/oidc/device/verify", map[string]string{"Sec-Fetch-Site": "none"}, http.StatusOK},
		{"old browser with matching host", http.MethodPost, "/api/oidc/device/verify", map[string]string{"Origin": "http://example.com"}, http.StatusOK},
		{"old browser behind host-rewriting proxy", http.MethodPost, "/api/oidc/device/verify", map[string]string{"Origin": "https://id.example.com"}, http.StatusOK},

		// Non-browser clients such as API key scripts send neither header
		{"non-browser client", http.MethodPost, "/api/oidc/device/verify", nil, http.StatusOK},

		// Safe methods are never checked
		{"cross-site get", http.MethodGet, "/api/oidc/device/verify", map[string]string{"Sec-Fetch-Site": "cross-site"}, http.StatusOK},
	}

	// Protocol endpoints are called by relying parties from their own origin
	for _, path := range []string{"/authorize", "/api/oidc/token", "/api/oidc/userinfo", "/api/oidc/introspect", "/api/oidc/end-session", "/api/oidc/par", "/api/oidc/device/authorize"} {
		tests = append(tests, testCase{"cross-site protocol endpoint " + path, http.MethodPost, path, map[string]string{"Sec-Fetch-Site": "cross-site", "Origin": "https://rp.test"}, http.StatusOK})
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequestWithContext(t.Context(), tt.method, tt.path, http.NoBody)
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			require.Equal(t, tt.want, w.Code, w.Body.String())
		})
	}
}
