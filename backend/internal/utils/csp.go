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

func BuildCSP(nonce string) string {
	return buildCSP(nonce, nil, nil)
}

// BuildFormPostCSP builds the Content-Security-Policy for an OIDC response_mode=form_post page
func BuildFormPostCSP(nonce, redirectURI, scriptHash string) string {
	return buildCSP(nonce, []string{redirectURI}, []string{scriptHash})
}

// BuildAPIDocsCSP allows the pinned Scalar bundle and the assets it creates
func BuildAPIDocsCSP(nonce string) string {
	scriptSrc := "script-src 'self' https://cdn.jsdelivr.net"
	if nonce != "" {
		scriptSrc += " 'nonce-" + nonce + "'"
	}

	return "default-src 'self'; " +
		"base-uri 'self'; " +
		"object-src 'none'; " +
		"frame-ancestors 'none'; " +
		"form-action 'self'; " +
		"img-src * blob: data:; " +
		"font-src 'self' https://cdn.jsdelivr.net data:; " +
		"style-src 'self' 'unsafe-inline' https://cdn.jsdelivr.net; " +
		"worker-src blob:; " +
		"connect-src 'self'; " +
		scriptSrc
}

func buildCSP(nonce string, formActionExtra, scriptSrcExtra []string) string {
	formAction := "'self'"
	scriptSrc := "script-src 'self'"
	if nonce != "" {
		scriptSrc += " 'nonce-" + nonce + "'"
	}

	for _, extra := range scriptSrcExtra {
		if extra != "" {
			scriptSrc += " " + extra
		}
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
