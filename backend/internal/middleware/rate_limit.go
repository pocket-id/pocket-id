package middleware

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/italypaleale/francis/builtin/ratelimit"

	"github.com/pocket-id/pocket-id/backend/internal/apperror"
	"github.com/pocket-id/pocket-id/backend/internal/common"
)

// Rate-limit policy names
// Each constant names a limiter registered on the actor host and is the value passed to Add to select that limiter
const (
	RateLimitAPI                     = "api"
	RateLimitSignup                  = "signup"
	RateLimitWebauthnLogin           = "webauthn-login"
	RateLimitWebauthnReauthenticate  = "webauthn-reauthenticate"
	RateLimitOneTimeAccessToken      = "one-time-access-token"
	RateLimitOneTimeAccessEmail      = "one-time-access-email"
	RateLimitDeviceLoginCreate       = "device-login-create"
	RateLimitDeviceLoginExchange     = "device-login-exchange"
	RateLimitDeviceLoginVerification = "device-login-verification"
	RateLimitSendEmailVerification   = "send-email-verification"
	RateLimitVerifyEmail             = "verify-email"
	RateLimitClientRegistration      = "client-registration"
	RateLimitClientConfiguration     = "client-configuration"
	RateLimitInternal                = "internal"
)

// RateLimitPolicy is the configuration for a single rate-limit actor
// Each policy maps to one Francis rate-limit actor type and requests are keyed by client IP, so every IP is limited independently and per-route limits stay isolated from each other
type RateLimitPolicy struct {
	// Name must be unique across policies and must not contain '/'
	Name string
	// Rate is the number of calls admitted per Per window
	Rate int
	// Per is the window the rate applies over
	Per time.Duration
	// Burst is the token bucket's capacity, i.e. how many calls may be admitted instantly before throttling kicks in
	Burst int
}

// RateLimitPolicies returns the configuration for every rate-limit policy
// The slice is built on each call so the policies are not retained at the package level, and the actor host registers one limiter per entry
func RateLimitPolicies() []RateLimitPolicy {
	return []RateLimitPolicy{
		{Name: RateLimitAPI, Rate: 100, Per: time.Second, Burst: 300},
		{Name: RateLimitSignup, Rate: 2, Per: time.Minute, Burst: 10},
		{Name: RateLimitWebauthnLogin, Rate: 1, Per: 5 * time.Second, Burst: 10},
		{Name: RateLimitWebauthnReauthenticate, Rate: 1, Per: 10 * time.Second, Burst: 5},
		{Name: RateLimitOneTimeAccessToken, Rate: 1, Per: 10 * time.Second, Burst: 5},
		{Name: RateLimitOneTimeAccessEmail, Rate: 2, Per: 10 * time.Minute, Burst: 5},
		{Name: RateLimitDeviceLoginCreate, Rate: 1, Per: 10 * time.Second, Burst: 5},
		{Name: RateLimitDeviceLoginExchange, Rate: 1, Per: 2 * time.Second, Burst: 10},
		{Name: RateLimitDeviceLoginVerification, Rate: 1, Per: 10 * time.Second, Burst: 5},
		{Name: RateLimitSendEmailVerification, Rate: 2, Per: 10 * time.Minute, Burst: 1},
		{Name: RateLimitVerifyEmail, Rate: 1, Per: 10 * time.Second, Burst: 5},
		// Dynamic Client Registration is unauthenticated when enabled, and every call
		// creates a durable row, so it is limited far more tightly than the generic API
		{Name: RateLimitClientRegistration, Rate: 1, Per: time.Minute, Burst: 5},
		// The RFC 7592 configuration endpoints authenticate with a bearer registration
		// access token, so they are throttled to blunt token brute-forcing
		{Name: RateLimitClientConfiguration, Rate: 1, Per: 2 * time.Second, Burst: 10},
		{Name: RateLimitInternal, Rate: 20, Per: time.Second, Burst: 20},
	}
}

type RateLimitMiddleware struct {
	services map[string]*ratelimit.RateLimitService
}

func NewRateLimitMiddleware(services map[string]*ratelimit.RateLimitService) *RateLimitMiddleware {
	return &RateLimitMiddleware{
		services: services,
	}
}

func (m *RateLimitMiddleware) Add(policy string) gin.HandlerFunc {
	if common.EnvConfig.DisableRateLimiting {
		return func(c *gin.Context) {
			c.Next()
		}
	}

	// A missing service means the policy was never registered on the actor host, which is a development-time error
	svc := m.services[policy]
	if svc == nil {
		return func(c *gin.Context) {
			_ = c.Error(apperror.Internal(fmt.Errorf("rate limiter service is not configured for policy %q", policy)))
			c.Abort()
		}
	}

	return func(c *gin.Context) {
		ip := c.ClientIP()

		// Skip rate limiting in test environments
		if common.EnvConfig.AppEnv == common.AppEnvTest {
			c.Next()
			return
		}

		// Allow is a non-blocking token-bucket check keyed by client IP: it consumes a slot and reports whether the call is admitted right now
		allowed, retryAfter, err := svc.Allow(c.Request.Context(), ip)
		if err != nil {
			// Fail open so a limiter error does not turn away otherwise-valid traffic
			if !errors.Is(err, context.Canceled) {
				// A cancelled context just means the client went away, so it is not worth logging
				slog.WarnContext(c.Request.Context(), "Rate limiter unavailable, allowing request", slog.String("policy", policy), slog.Any("error", err))
			}
			c.Next()
			return
		}

		if !allowed {
			// Advertise when the caller may retry, mapping the limiter's delay onto a structured application error
			_ = c.Error(apperror.TooManyRequests().WithRetryAfter(retryAfter))
			c.Abort()
			return
		}

		c.Next()
	}
}
