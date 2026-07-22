package middleware

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/gin-gonic/gin"
	"github.com/italypaleale/francis/builtin/ratelimit"
	"github.com/italypaleale/francis/host/local"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pocket-id/pocket-id/backend/internal/common"
	httpapi "github.com/pocket-id/pocket-id/backend/internal/utils/huma"
	testutils "github.com/pocket-id/pocket-id/backend/internal/utils/testing"
)

// startRateLimitServices registers one rate-limit actor per policy on an in-memory test actor host and returns the bound services keyed by policy name
func startRateLimitServices(t *testing.T, policies ...RateLimitPolicy) map[string]*ratelimit.RateLimitService {
	t.Helper()

	limiters := make(map[string]*ratelimit.RateLimit, len(policies))
	h := testutils.NewActorHostForTest(t, func(t *testing.T, h *local.Host) {
		for _, p := range policies {
			rl, err := ratelimit.New(p.Name, ratelimit.WithRate(p.Rate), ratelimit.WithPer(p.Per), ratelimit.WithBurst(p.Burst))
			require.NoError(t, err)
			limiters[p.Name] = rl

			err = h.RegisterBuiltInActor(rl)
			require.NoError(t, err)
		}
	})

	services := make(map[string]*ratelimit.RateLimitService, len(limiters))
	svc := h.Service()
	for name, rl := range limiters {
		services[name] = rl.Service(svc)
	}

	return services
}

// newRateLimitRouter builds a gin engine that runs the rate-limit middleware for the given policy on GET /test
// Trusted proxies are disabled so ClientIP resolves to the request's RemoteAddr, and the error handler turns the middleware's error into a 429 response
func newRateLimitRouter(t *testing.T, services map[string]*ratelimit.RateLimitService, policy string) *gin.Engine {
	t.Helper()

	r := gin.New()

	err := r.SetTrustedProxies(nil)
	require.NoError(t, err)

	r.Use(NewErrorHandlerMiddleware().Add())

	mw := NewRateLimitMiddleware(services)
	r.GET("/test", mw.Add(policy), func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	return r
}

// doRateLimitRequest sends a GET /test request as the given client IP and returns the response recorder
func doRateLimitRequest(ctx context.Context, r *gin.Engine, ip string) *httptest.ResponseRecorder {
	req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/test", nil)
	req.RemoteAddr = net.JoinHostPort(ip, "12345")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestRateLimitMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// The middleware reads the global config, so restore it after the test
	originalEnvConfig := common.EnvConfig
	t.Cleanup(func() {
		common.EnvConfig = originalEnvConfig
	})

	common.EnvConfig.AppEnv = common.AppEnvProduction
	common.EnvConfig.DisableRateLimiting = false

	// A slow refill (one token per hour) with a burst of 2 means the first two calls for a key are admitted and the third is rejected, with no refill during the test
	const policy = "test-limit"
	services := startRateLimitServices(t, RateLimitPolicy{Name: policy, Rate: 1, Per: time.Hour, Burst: 2})

	t.Run("rejects requests over the limit and sets Retry-After", func(t *testing.T) {
		r := newRateLimitRouter(t, services, policy)
		const ip = "203.0.113.1"

		require.Equal(t, http.StatusOK, doRateLimitRequest(t.Context(), r, ip).Code)
		require.Equal(t, http.StatusOK, doRateLimitRequest(t.Context(), r, ip).Code)

		w := doRateLimitRequest(t.Context(), r, ip)
		require.Equal(t, http.StatusTooManyRequests, w.Code)
		assert.NotEmpty(t, w.Header().Get("Retry-After"), "a throttled response should advertise Retry-After")
	})

	t.Run("limits each client IP independently", func(t *testing.T) {
		r := newRateLimitRouter(t, services, policy)

		// Exhaust the budget for the first IP
		require.Equal(t, http.StatusOK, doRateLimitRequest(t.Context(), r, "203.0.113.2").Code)
		require.Equal(t, http.StatusOK, doRateLimitRequest(t.Context(), r, "203.0.113.2").Code)
		require.Equal(t, http.StatusTooManyRequests, doRateLimitRequest(t.Context(), r, "203.0.113.2").Code)

		// A different IP still has its full budget
		require.Equal(t, http.StatusOK, doRateLimitRequest(t.Context(), r, "203.0.113.3").Code)
	})

	t.Run("allows all requests when rate limiting is disabled", func(t *testing.T) {
		common.EnvConfig.DisableRateLimiting = true
		t.Cleanup(func() { common.EnvConfig.DisableRateLimiting = false })

		r := newRateLimitRouter(t, services, policy)
		for range 5 {
			require.Equal(t, http.StatusOK, doRateLimitRequest(t.Context(), r, "203.0.113.4").Code)
		}
	})

	t.Run("skips rate limiting for loopback addresses", func(t *testing.T) {
		r := newRateLimitRouter(t, services, policy)
		for _, ip := range []string{"127.0.0.1", "::1"} {
			for range 5 {
				require.Equal(t, http.StatusOK, doRateLimitRequest(t.Context(), r, ip).Code)
			}
		}
	})

	t.Run("skips rate limiting in the test environment", func(t *testing.T) {
		common.EnvConfig.AppEnv = common.AppEnvTest
		t.Cleanup(func() { common.EnvConfig.AppEnv = common.AppEnvProduction })

		r := newRateLimitRouter(t, services, policy)
		for range 5 {
			require.Equal(t, http.StatusOK, doRateLimitRequest(t.Context(), r, "203.0.113.5").Code)
		}
	})

	t.Run("fails with 500 when the policy is not registered", func(t *testing.T) {
		// An unknown policy has no bound service, which is a configuration error surfaced as a 500
		r := newRateLimitRouter(t, services, "does-not-exist")
		require.Equal(t, http.StatusInternalServerError, doRateLimitRequest(t.Context(), r, "203.0.113.6").Code)
	})
}

func TestHumaRateLimitMiddleware(t *testing.T) {
	originalEnvConfig := common.EnvConfig
	t.Cleanup(func() { common.EnvConfig = originalEnvConfig })
	common.EnvConfig.AppEnv = common.AppEnvProduction
	common.EnvConfig.DisableRateLimiting = false

	const policy = "huma-test-limit"
	services := startRateLimitServices(t, RateLimitPolicy{Name: policy, Rate: 1, Per: time.Hour, Burst: 1})
	router := gin.New()
	require.NoError(t, router.SetTrustedProxies(nil))
	api := httpapi.New(router, router.Group("/"))
	operation := huma.Operation{OperationID: "huma-rate-limit", Method: http.MethodGet, Path: "/api/rate-limit"}
	operation.Middlewares = append(operation.Middlewares, NewRateLimitMiddleware(services).Huma(api, policy))
	httpapi.Register(api, operation, func(context.Context, *struct{}) (*struct{}, error) { return &struct{}{}, nil })

	request := func() *httptest.ResponseRecorder {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/rate-limit", nil)
		req.RemoteAddr = net.JoinHostPort("203.0.113.10", "12345")
		response := httptest.NewRecorder()
		router.ServeHTTP(response, req)
		return response
	}

	require.Equal(t, http.StatusNoContent, request().Code)
	response := request()
	require.Equal(t, http.StatusTooManyRequests, response.Code)
	require.NotEmpty(t, response.Header().Get("Retry-After"))
	require.JSONEq(t, `{"error":"Too many requests"}`, response.Body.String())
}
