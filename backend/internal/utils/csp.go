package utils

import (
	"crypto/rand"
	"encoding/base64"
	"strings"

	"github.com/gin-gonic/gin"
)

const cspNonceContextKey = "csp_nonce"

// GetCSPNonce returns the CSP nonce generated for this request, if any.
func GetCSPNonce(c *gin.Context) string {
	if v, ok := c.Get(cspNonceContextKey); ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// SetCSPNonce stores a per-request CSP nonce so handlers can reference it in
// Content-Security-Policy headers they emit themselves.
func SetCSPNonce(c *gin.Context, nonce string) {
	c.Set(cspNonceContextKey, nonce)
}

func BuildCSP(nonce string, formActionExtra ...string) string {
	formAction := "'self'"
	scriptSrc := "script-src 'self'"
	if nonce != "" {
		scriptSrc += " 'nonce-" + nonce + "'"
	}

	if len(formActionExtra) > 0 {
		b := strings.Builder{}

		for _, extra := range formActionExtra {
			if extra != "" {
				b.WriteByte(' ')
				b.WriteString(extra)
			}
		}

		formAction += b.String()
	}

	return "default-src 'self'; " +
		"base-uri 'self'; " +
		"object-src 'none'; " +
		"frame-ancestors 'none'; " +
		"form-action " + formAction + "; " +
		"img-src * blob:;" +
		"font-src 'self'; " +
		"style-src 'self' 'unsafe-inline'; " +
		scriptSrc
}

// GenerateCSPNonce returns a random base64 nonce for use in a CSP header.
func GenerateCSPNonce() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "" // if generation fails, return empty; policy will omit nonce
	}
	return base64.RawURLEncoding.EncodeToString(b)
}
