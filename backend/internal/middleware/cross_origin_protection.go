package middleware

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/pocket-id/pocket-id/backend/internal/apperror"
)

// CrossOriginProtectionMiddleware rejects state-changing browser requests that come from another origin to prevent CSRF
type CrossOriginProtectionMiddleware struct {
	protection *http.CrossOriginProtection
}

func NewCrossOriginProtectionMiddleware(appURL string) *CrossOriginProtectionMiddleware {
	protection := http.NewCrossOriginProtection()

	// Trust APP_URL so browsers without Sec-Fetch-Site still pass when a reverse proxy rewrites the Host header
	if appURL != "" {
		if err := protection.AddTrustedOrigin(appURL); err != nil {
			slog.Warn("Failed to add APP_URL as a trusted origin for cross-origin protection", slog.Any("error", err))
		}
	}

	return &CrossOriginProtectionMiddleware{protection: protection}
}

func (m *CrossOriginProtectionMiddleware) Add() gin.HandlerFunc {
	return func(c *gin.Context) {
		if isCrossOriginAllowedPath(c.FullPath()) {
			c.Next()
			return
		}

		if err := m.protection.Check(c.Request); err != nil {
			_ = c.Error(apperror.CrossOriginRequestForbidden(err))
			c.Abort()
			return
		}

		c.Next()
	}
}

// isCrossOriginAllowedPath reports whether a route must accept requests from other origins
// These are OAuth/OIDC protocol endpoints that relying parties call from their own origin, either via CORS or a cross-site form post
func isCrossOriginAllowedPath(path string) bool {
	if isCorsPath(path) {
		return true
	}

	switch path {
	case "/authorize",
		"/api/oidc/par",
		"/api/oidc/device/authorize":
		return true
	default:
		return false
	}
}
