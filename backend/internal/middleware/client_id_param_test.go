package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

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
