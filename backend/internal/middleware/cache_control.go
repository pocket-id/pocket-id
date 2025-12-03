package middleware

import "github.com/gin-gonic/gin"

// CacheControlMiddleware sets a safe default Cache-Control header on responses
// that do not already specify one. This prevents proxies from caching
// authenticated responses that might contain private data.
type CacheControlMiddleware struct {
	headerValue string
}

func NewCacheControlMiddleware() *CacheControlMiddleware {
	return &CacheControlMiddleware{
		headerValue: "private, no-store",
	}
}

func (m *CacheControlMiddleware) Add() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Writer.Header().Get("Cache-Control") == "" {
			c.Header("Cache-Control", m.headerValue)
		}

		c.Next()
	}
}
