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
	// Map to store the rate limiters per IP
	var clients = make(map[string]*client)
	var mu sync.Mutex

	// Start the cleanup routine
	go cleanupClients(&mu, clients)

	return func(c *gin.Context) {
		ip := c.ClientIP()

		// Skip rate limiting for localhost and test environment
		// If the client ip is localhost the request comes from the frontend
		if ip == "" || ip == "127.0.0.1" || ip == "::1" || common.EnvConfig.AppEnv == "test" {
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
