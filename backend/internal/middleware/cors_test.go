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
