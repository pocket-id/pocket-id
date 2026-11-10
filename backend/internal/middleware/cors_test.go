package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestCorsMiddlewareAllowsDiscoveryDocuments(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(NewCorsMiddleware().Add())

	paths := []string{"/.well-known/openid-configuration", "/.well-known/oauth-authorization-server"}
	for _, path := range paths {
		router.GET(path, func(c *gin.Context) {
			c.Status(http.StatusOK)
		})
	}

	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, path, http.NoBody)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			require.Equal(t, http.StatusOK, w.Code)
			require.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))

			req = httptest.NewRequestWithContext(t.Context(), http.MethodOptions, path, http.NoBody)
			w = httptest.NewRecorder()
			router.ServeHTTP(w, req)

			require.Equal(t, http.StatusNoContent, w.Code)
			require.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
		})
	}
}

func TestCorsMiddlewareAllowsBearerRequestsToTheAPI(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(NewCorsMiddleware().Add())
	router.DELETE("/api/users/:id", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	// The preflight carries no Authorization header, only the request headers it asks to send
	req := httptest.NewRequestWithContext(t.Context(), http.MethodOptions, "/api/users/1", http.NoBody)
	req.Header.Set("Access-Control-Request-Headers", "authorization")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusNoContent, w.Code)
	require.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
	require.Contains(t, w.Header().Get("Access-Control-Allow-Methods"), http.MethodDelete)
	require.Empty(t, w.Header().Get("Access-Control-Allow-Credentials"))

	req = httptest.NewRequestWithContext(t.Context(), http.MethodDelete, "/api/users/1", http.NoBody)
	req.Header.Set("Authorization", "Bearer token")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))

	// A cookie-authenticated request from the SPA itself gets no CORS headers
	req = httptest.NewRequestWithContext(t.Context(), http.MethodDelete, "/api/users/1", http.NoBody)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Empty(t, w.Header().Get("Access-Control-Allow-Origin"))
}

func TestCorsMiddlewareLetsBrowserMCPClientsReadTheChallenge(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(NewCorsMiddleware().Add())
	router.POST("/mcp", func(c *gin.Context) {
		c.Status(http.StatusUnauthorized)
	})

	// The first request of an MCP client carries no token yet
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/mcp", http.NoBody)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
	require.Equal(t, "WWW-Authenticate", w.Header().Get("Access-Control-Expose-Headers"))
	require.Contains(t, w.Header().Get("Access-Control-Allow-Headers"), "Mcp-Protocol-Version")
}
