package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
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

		if !isCorsPath(path) {
			c.Next()
			return
		}

		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Authorization")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST")

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
		"/oidc/end-session",
		"/api/oidc/introspect",
		"/.well-known/jwks.json",
		"/.well-known/openid-configuration":
		return true
	default:
		return false
	}
}
