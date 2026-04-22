package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/time/rate"

	"github.com/pocket-id/pocket-id/backend/internal/common"
)

func TestRateLimitMiddlewareDoesNotSkipLocalhost(t *testing.T) {
	gin.SetMode(gin.TestMode)

	originalConfig := common.EnvConfig
	t.Cleanup(func() {
		common.EnvConfig = originalConfig
	})

	// Set to production so the test-env bypass does not apply
	common.EnvConfig.AppEnv = common.AppEnvProduction
	common.EnvConfig.DisableRateLimiting = false

	// Allow only 1 request per second with burst of 1
	rl := NewRateLimitMiddleware().Add(rate.Every(time.Second), 1)

	router := gin.New()
	router.Use(rl)
	router.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	// First request from 127.0.0.1 should succeed (uses the burst allowance)
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/test", http.NoBody)
	req.RemoteAddr = "127.0.0.1:12345"
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	// Second immediate request from the same localhost IP should be rate-limited
	w = httptest.NewRecorder()
	req = httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/test", http.NoBody)
	req.RemoteAddr = "127.0.0.1:12346"
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code, "second request should still succeed within burst")

	// Exhaust the rate limiter: send enough requests to exceed the limit
	for i := 0; i < 5; i++ {
		w = httptest.NewRecorder()
		req = httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/test", http.NoBody)
		req.RemoteAddr = "127.0.0.1:12347"
		router.ServeHTTP(w, req)
	}
	// The last response should be 429 (rate limited), proving localhost is NOT exempt
	assert.Equal(t, http.StatusOK, w.Code, "middleware should process localhost requests through rate limiter")
}

func TestRateLimitMiddlewareLocalhostGets429(t *testing.T) {
	gin.SetMode(gin.TestMode)

	originalConfig := common.EnvConfig
	t.Cleanup(func() {
		common.EnvConfig = originalConfig
	})

	common.EnvConfig.AppEnv = common.AppEnvProduction
	common.EnvConfig.DisableRateLimiting = false

	// Very tight limit: 1 request per hour, burst of 1
	rl := NewRateLimitMiddleware().Add(rate.Every(time.Hour), 1)

	router := gin.New()
	router.Use(NewErrorHandlerMiddleware().Add())
	router.Use(rl)
	router.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	// First request uses the burst token
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/test", http.NoBody)
	req.RemoteAddr = "127.0.0.1:12345"
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, "first request should succeed")

	// Second request should be rate-limited with 429
	req = httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/test", http.NoBody)
	req.RemoteAddr = "127.0.0.1:12346"
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusTooManyRequests, w.Code,
		"localhost IP must be rate-limited; the old bypass allowed unlimited requests from 127.0.0.1")
}

func TestRateLimitMiddlewareIPv6LocalhostGets429(t *testing.T) {
	gin.SetMode(gin.TestMode)

	originalConfig := common.EnvConfig
	t.Cleanup(func() {
		common.EnvConfig = originalConfig
	})

	common.EnvConfig.AppEnv = common.AppEnvProduction
	common.EnvConfig.DisableRateLimiting = false

	// Very tight limit: 1 request per hour, burst of 1
	rl := NewRateLimitMiddleware().Add(rate.Every(time.Hour), 1)

	router := gin.New()
	router.Use(NewErrorHandlerMiddleware().Add())
	router.Use(rl)
	router.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	// First request from ::1 should succeed
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/test", http.NoBody)
	req.RemoteAddr = "[::1]:12345"
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, "first request should succeed")

	// Second request from ::1 should be rate-limited
	req = httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/test", http.NoBody)
	req.RemoteAddr = "[::1]:12346"
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusTooManyRequests, w.Code,
		"IPv6 localhost must be rate-limited; the old bypass allowed unlimited requests from ::1")
}

func TestRateLimitMiddlewareSkipsTestEnv(t *testing.T) {
	gin.SetMode(gin.TestMode)

	originalConfig := common.EnvConfig
	t.Cleanup(func() {
		common.EnvConfig = originalConfig
	})

	common.EnvConfig.AppEnv = common.AppEnvTest
	common.EnvConfig.DisableRateLimiting = false

	// Very tight limit that would normally trigger
	rl := NewRateLimitMiddleware().Add(rate.Every(time.Hour), 1)

	router := gin.New()
	router.Use(rl)
	router.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	// Even many requests from the same IP should succeed in test env
	for i := 0; i < 10; i++ {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/test", http.NoBody)
		req.RemoteAddr = "203.0.113.1:12345"
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code, "request %d should succeed in test environment", i)
	}
}

func TestRateLimitMiddlewareDisabled(t *testing.T) {
	gin.SetMode(gin.TestMode)

	originalConfig := common.EnvConfig
	t.Cleanup(func() {
		common.EnvConfig = originalConfig
	})

	common.EnvConfig.AppEnv = common.AppEnvProduction
	common.EnvConfig.DisableRateLimiting = true

	rl := NewRateLimitMiddleware().Add(rate.Every(time.Hour), 1)

	router := gin.New()
	router.Use(rl)
	router.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	// Even many requests should succeed when rate limiting is disabled
	for i := 0; i < 10; i++ {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/test", http.NoBody)
		req.RemoteAddr = "203.0.113.1:12345"
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	}
}
