package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/pocket-id/pocket-id/backend/internal/authz"
)

type CorsMiddleware struct{}

func NewCorsMiddleware() *CorsMiddleware {
	return &CorsMiddleware{}
}

func (m *CorsMiddleware) Add() gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.FullPath()
		if path == "" {
			// The router doesn't map preflight requests, so we need to use the raw URL path
			path = c.Request.URL.Path
		}

		switch {
		case isCorsPath(path):
			c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
			c.Writer.Header().Set("Access-Control-Allow-Headers", "Authorization")
			c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST")
		case path == authz.MCPPath || path == authz.MCPEndpointPath || (strings.HasPrefix(path, "/api/") && (c.Request.Method == http.MethodOptions || c.GetHeader("Authorization") != "")):
			// Browser apps on other origins call the API with OAuth access tokens
			// Credentials are never allowed, so the browser refuses to send or expose a cross-origin request that relies on the session cookie
			// The MCP endpoint and the URL that redirects to it also answer anonymous requests, so browser MCP clients can read the 401 challenge that starts their OAuth flow
			c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
			c.Writer.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, Mcp-Protocol-Version")
			c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE")
			c.Writer.Header().Set("Access-Control-Expose-Headers", "WWW-Authenticate")
		default:
			c.Next()
			return
		}

		// Preflight request
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

func isCorsPath(path string) bool {
	switch path {
	case "/api/oidc/token",
		"/api/oidc/userinfo",
		"/api/oidc/end-session",
		"/api/oidc/introspect",
		"/.well-known/jwks.json",
		"/.well-known/openid-configuration",
		"/.well-known/oauth-authorization-server",
		authz.MCPResourceMetadataPath:
		return true
	default:
		return false
	}
}
