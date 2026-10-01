package middleware

import (
	"github.com/gin-gonic/gin"

	"github.com/pocket-id/pocket-id/backend/internal/iplocation"
)

// CloudflareLocationMiddleware captures location headers for services that only receive the request context
// Register it only when the deployment explicitly trusts Cloudflare's location headers
func CloudflareLocationMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := iplocation.WithCloudflareLocation(c.Request.Context(), c.ClientIP(), c.GetHeader("CF-IPCountry"), c.GetHeader("CF-IPCity"))
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}
