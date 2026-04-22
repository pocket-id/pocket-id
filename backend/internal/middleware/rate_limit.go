package middleware

import (
	"sync"
	"time"

	"github.com/pocket-id/pocket-id/backend/internal/common"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

type RateLimitMiddleware struct{}

func NewRateLimitMiddleware() *RateLimitMiddleware {
	return &RateLimitMiddleware{}
}

func (m *RateLimitMiddleware) Add(limit rate.Limit, burst int) gin.HandlerFunc {
	if common.EnvConfig.DisableRateLimiting {
		return func(c *gin.Context) {
			c.Next()
		}
	}

	// Map to store the rate limiters per IP
	var clients = make(map[string]*client)
	var mu sync.Mutex

	// Start the cleanup routine
	go cleanupClients(&mu, clients)

	return func(c *gin.Context) {
		ip := c.ClientIP()

		// Skip rate limiting only in the test environment.
		// Localhost IPs are NOT exempted: with proper trusted-proxy
		// configuration, Gin resolves the real client IP from forwarded
		// headers, so a localhost ClientIP() should not appear in
		// production. Exempting localhost would allow any client behind a
		// misconfigured proxy to bypass rate limiting via X-Forwarded-For.
		if ip == "" || common.EnvConfig.AppEnv.IsTest() {
			c.Next()
			return
		}

		limiter := getLimiter(ip, limit, burst, &mu, clients)
		if !limiter.Allow() {
			_ = c.Error(&common.TooManyRequestsError{})
			c.Abort()
			return
		}

		c.Next()
	}
}

type client struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// Cleanup routine to remove stale clients that haven't been seen for a while
func cleanupClients(mu *sync.Mutex, clients map[string]*client) {
	for {
		time.Sleep(time.Minute)
		mu.Lock()
		for ip, client := range clients {
			if time.Since(client.lastSeen) > 3*time.Minute {
				delete(clients, ip)
			}
		}
		mu.Unlock()
	}
}

// getLimiter retrieves the rate limiter for a given IP address, creating one if it doesn't exist
func getLimiter(ip string, limit rate.Limit, burst int, mu *sync.Mutex, clients map[string]*client) *rate.Limiter {
	mu.Lock()
	defer mu.Unlock()

	if client, exists := clients[ip]; exists {
		client.lastSeen = time.Now()
		return client.limiter
	}

	limiter := rate.NewLimiter(limit, burst)
	clients[ip] = &client{limiter: limiter, lastSeen: time.Now()}
	return limiter
}
