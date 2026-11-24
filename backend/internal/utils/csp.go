package utils

import (
	"crypto/rand"
	"encoding/base64"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/pocket-id/pocket-id/backend/internal/common"
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
	return buildCSP(nonce, []string{formActionSource(redirectURI)}, []string{scriptHash})
}

// formActionSource returns the origin of the redirect URI as a CSP source expression, or an empty string if it has none
// The path and query are left out because the client chooses them and they may contain the ";", "," and whitespace that end a directive or a policy
func formActionSource(redirectURI string) string {
	u, err := url.Parse(redirectURI)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return ""
	}
	origin := (&url.URL{Scheme: u.Scheme, Host: u.Host}).String()

	// net/url accepts these delimiters in a host too
	if strings.ContainsAny(origin, ";, \t\n\f\r") {
		return ""
	}
	return origin
}

func buildCSP(nonce string, formActionExtra, scriptSrcExtra []string) string {
	formAction := "'self'"
	var scriptSrc strings.Builder
	scriptSrc.WriteString("script-src 'self'")
	if nonce != "" {
		scriptSrc.WriteString(" 'nonce-" + nonce + "'")
	}

	for _, extra := range scriptSrcExtra {
		if extra != "" {
			scriptSrc.WriteString(" " + extra)
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

	// Only the origins the admin allowed may embed Pocket ID, because framing exposes it to clickjacking
	frameAncestors := "'none'"
	if len(common.EnvConfig.AllowedFrameAncestors) > 0 {
		frameAncestors = strings.Join(common.EnvConfig.AllowedFrameAncestors, " ")
	}

	return "default-src 'self'; " +
		"base-uri 'self'; " +
		"object-src 'none'; " +
		"frame-ancestors " + frameAncestors + "; " +
		"form-action " + formAction + "; " +
		"img-src * blob:;" +
		"font-src 'self'; " +
		"style-src 'self' 'unsafe-inline'; " +
		scriptSrc.String()
}

// GenerateCSPNonce returns a random base64 nonce for use in a CSP header.
func GenerateCSPNonce() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "" // if generation fails, return empty; policy will omit nonce
	}
	return base64.RawURLEncoding.EncodeToString(b)
}
