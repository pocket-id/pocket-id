package middleware

import (
	"crypto/rand"
	"encoding/base64"

	"github.com/gin-gonic/gin"
)

// CspMiddleware sets a Content Security Policy header and, when possible,
// includes a per-request nonce for inline scripts.
type CspMiddleware struct{}

func NewCspMiddleware() *CspMiddleware { return &CspMiddleware{} }

// GetCSPNonce returns the CSP nonce generated for this request, if any.
func GetCSPNonce(c *gin.Context) string {
	if v, ok := c.Get("csp_nonce"); ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func (m *CspMiddleware) Add() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Generate a random base64 nonce for this request
		nonce := generateNonce()
		c.Set("csp_nonce", nonce)

		csp := "default-src 'self'; " +
			"base-uri 'self'; " +
			"object-src 'none'; " +
			"frame-ancestors 'none'; " +
			"form-action 'self'; " +
			"img-src 'self' data: blob:; " +
			"font-src 'self'; " +
			"style-src 'self' 'unsafe-inline'; " +
			"script-src 'self' 'nonce-" + nonce + "'; " +
			"connect-src 'self' https://api.github.com"

		c.Writer.Header().Set("Content-Security-Policy", csp)
		c.Next()
	}
}

func generateNonce() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "" // if generation fails, return empty; policy will omit nonce
	}
	return base64.RawURLEncoding.EncodeToString(b)
}
