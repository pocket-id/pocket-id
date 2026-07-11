package middleware

import (
	"context"
	"errors"
	"log/slog"
	"math"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/italypaleale/francis/builtin/ratelimit"

	"github.com/pocket-id/pocket-id/backend/internal/common"
)

// Rate-limit policy names
// Each constant names a limiter registered on the actor host and is the value passed to Add to select that limiter
const (
	RateLimitAPI                    = "api"
	RateLimitSignup                 = "signup"
	RateLimitWebauthnLogin          = "webauthn-login"
	RateLimitWebauthnReauthenticate = "webauthn-reauthenticate"
	RateLimitOneTimeAccessToken     = "one-time-access-token"
	RateLimitOneTimeAccessEmail     = "one-time-access-email"
	RateLimitSendEmailVerification  = "send-email-verification"
	RateLimitVerifyEmail            = "verify-email"
	RateLimitInternal               = "internal"
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
		{Name: RateLimitSendEmailVerification, Rate: 2, Per: 10 * time.Minute, Burst: 1},
		{Name: RateLimitVerifyEmail, Rate: 1, Per: 10 * time.Second, Burst: 5},
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

	// A missing service means the policy was never registered on the actor host, which is a development-time errror
	svc := m.services[policy]
	if svc == nil {
		return func(c *gin.Context) {
			c.AbortWithStatus(http.StatusInternalServerError)
		}
	}

	return func(c *gin.Context) {
		ip := c.ClientIP()

		// Skip rate limiting for localhost and test environment
		// If the client ip is localhost the request comes from the frontend
		if common.EnvConfig.AppEnv == common.AppEnvTest || net.ParseIP(ip).IsLoopback() {
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
			// Advertise when the caller may retry, mapping the limiter's delay onto a Retry-After header
			if retryAfter > 0 {
				c.Header("Retry-After", strconv.Itoa(int(math.Ceil(retryAfter.Seconds()))))
			}
			_ = c.Error(&common.TooManyRequestsError{})
			c.Abort()
			return
		}

		c.Next()
	}
}
