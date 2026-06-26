package middleware

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestClientIDParamMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	const cimdURL = "https://claude.ai/oauth/claude-code-client-metadata"
	encoded := "~" + base64.RawURLEncoding.EncodeToString([]byte(cimdURL))

	tests := []struct {
		name  string
		param string
		want  string
	}{
		{"plain client ID unchanged", "my-client_id.1", "my-client_id.1"},
		{"uuid unchanged", "550e8400-e29b-41d4-a716-446655440000", "550e8400-e29b-41d4-a716-446655440000"},
		{"encoded CIMD URL decoded", encoded, cimdURL},
		{"invalid base64 left as-is", "~!!!", "~!!!"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.Use(NewClientIDParamMiddleware().Add())

			var got string
			router.GET("/oidc/clients/:id/meta", func(c *gin.Context) {
				got = c.Param("id")
				c.Status(http.StatusOK)
			})

			req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/oidc/clients/"+tt.param+"/meta", http.NoBody)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			require.Equal(t, http.StatusOK, w.Code)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestClientIDParamMiddlewareIgnoresNonClientParams(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(NewClientIDParamMiddleware().Add())

	var got string
	// "~"-prefixed value on a non-client param key must pass through untouched.
	router.GET("/users/:userId", func(c *gin.Context) {
		got = c.Param("userId")
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/users/~abc", http.NoBody)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, "~abc", got)
}
