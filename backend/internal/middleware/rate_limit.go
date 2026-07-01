package middleware

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/italypaleale/francis/builtin/ratelimit"

	"github.com/pocket-id/pocket-id/backend/internal/common"
)

// rateLimitWaitTimeout bounds how long a request waits for the rate-limit actor to admit it
// The Francis rate limiter blocks until the key's leaky bucket admits the call: in-budget bursts are admitted instantly, while an over-budget call would otherwise sleep for at least the smallest throttle interval (1s)
// Capping the wait well below that interval makes over-the-limit requests hit the deadline and get rejected with a 429, preserving the previous non-blocking behavior
const rateLimitWaitTimeout = 250 * time.Millisecond

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
	// Slack is the burst allowance, i.e. how many unspent calls may accumulate
	Slack int
}

// RateLimitPolicies returns the configuration for every rate-limit policy
func RateLimitPolicies() []RateLimitPolicy {
	return []RateLimitPolicy{
		{Name: RateLimitAPI, Rate: 1, Per: time.Second, Slack: 100},
		{Name: RateLimitSignup, Rate: 1, Per: time.Minute, Slack: 10},
		{Name: RateLimitWebauthnLogin, Rate: 1, Per: 10 * time.Second, Slack: 5},
		{Name: RateLimitWebauthnReauthenticate, Rate: 1, Per: 10 * time.Second, Slack: 5},
		{Name: RateLimitOneTimeAccessToken, Rate: 1, Per: 10 * time.Second, Slack: 5},
		{Name: RateLimitOneTimeAccessEmail, Rate: 1, Per: 10 * time.Minute, Slack: 3},
		{Name: RateLimitSendEmailVerification, Rate: 1, Per: 10 * time.Minute, Slack: 3},
		{Name: RateLimitVerifyEmail, Rate: 1, Per: 10 * time.Second, Slack: 5},
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

	svc := m.services[policy]
	if svc == nil {
		return func(c *gin.Context) {
			c.AbortWithStatus(500)
		}
	}

	return func(c *gin.Context) {
		ip := c.ClientIP()

		// Skip rate limiting for localhost and test environment
		// If the client ip is localhost the request comes from the frontend
		if ip == "" || ip == "127.0.0.1" || ip == "::1" || common.EnvConfig.AppEnv.IsTest() {
			c.Next()
			return
		}

		// Bound the wait so an over-the-limit request hits the deadline and is rejected instead of blocking until the next slot
		ctx, cancel := context.WithTimeout(c.Request.Context(), rateLimitWaitTimeout)
		defer cancel()

		// Take admits this key against the policy's limiter, blocking until admitted or the context expires
		err := svc.Take(ctx, ip)
		if err != nil {
			_ = c.Error(&common.TooManyRequestsError{})
			c.Abort()
			return
		}

		c.Next()
	}
}
