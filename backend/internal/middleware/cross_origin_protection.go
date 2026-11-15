package middleware

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/pocket-id/pocket-id/backend/internal/apperror"
	"github.com/pocket-id/pocket-id/backend/internal/authz"
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
		if isCrossOriginAllowedPath(c.FullPath()) || isCookielessBearerRequest(c) {
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
		"/api/oidc/device/authorize",
		// The MCP endpoint never accepts the session cookie, so there is nothing to forge, and browser MCP clients have to reach it anonymously to get its 401 challenge
		authz.MCPPath,
		authz.MCPEndpointPath:
		return true
	default:
		return false
	}
}

// isCookielessBearerRequest reports whether the request authenticates with an Authorization header and carries no cookie, as browser apps on other origins do when they call the API with an OAuth access token
// CSRF rides on ambient cookies, so a request without any cannot be forged, while a request with one is still checked whatever headers it adds
func isCookielessBearerRequest(c *gin.Context) bool {
	return c.GetHeader("Authorization") != "" && c.GetHeader("Cookie") == ""
}
