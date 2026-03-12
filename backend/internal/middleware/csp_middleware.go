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

// SetAllowedFormAction sets the allowed form-action for the CSP header.
// This is used for OAuth2 response_mode=form_post to allow form submissions to the client's redirect URI.
func SetAllowedFormAction(c *gin.Context, uri string) {
	c.Set("csp_allowed_form_action", uri)
}

func (m *CspMiddleware) Add() gin.HandlerFunc {
	return func(c *gin.Context) {
		nonce := generateNonce()
		c.Set("csp_nonce", nonce)

		// Let the handler run first, then set CSP header with the final context values
		c.Next()

		// Determine if there is an EXTRA target beyond 'self'
		// This is set by handlers (e.g., OIDC authorize) after validating the redirect URI
		var extraAction string
		if v, ok := c.Get("csp_allowed_form_action"); ok {
			extraAction, _ = v.(string)
		}

		// 'self' is kept in the string; extraAction is just appended
		csp := "default-src 'self'; " +
			"base-uri 'self'; " +
			"object-src 'none'; " +
			"frame-ancestors 'none'; " +
			"form-action 'self' " + extraAction + "; " +
			"img-src * blob:;" +
			"font-src 'self'; " +
			"style-src 'self' 'unsafe-inline'; " +
			"script-src 'self' 'nonce-" + nonce + "'"

		c.Writer.Header().Set("Content-Security-Policy", csp)
	}
}

func generateNonce() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "" // if generation fails, return empty; policy will omit nonce
	}
	return base64.RawURLEncoding.EncodeToString(b)
}
