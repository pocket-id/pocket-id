package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestCacheControlMiddlewareSetsDefault(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(NewCacheControlMiddleware().Add())

	router.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	require.Equal(t, "private, no-store", w.Header().Get("Cache-Control"))
}

func TestCacheControlMiddlewarePreservesExistingHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(NewCacheControlMiddleware().Add())

	router.GET("/custom", func(c *gin.Context) {
		c.Header("Cache-Control", "public, max-age=60")
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/custom", http.NoBody)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	require.Equal(t, "public, max-age=60", w.Header().Get("Cache-Control"))
}
