package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/pocket-id/pocket-id/backend/internal/common"
)

type CorsMiddleware struct{}

func NewCorsMiddleware() *CorsMiddleware {
	return &CorsMiddleware{}
}

func (m *CorsMiddleware) Add() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Allow all origins for the token endpoint
		switch c.FullPath() {
		case "/api/oidc/token", "/api/oidc/introspect":
			c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		default:
			c.Writer.Header().Set("Access-Control-Allow-Origin", common.EnvConfig.AppURL)
		}

		c.Writer.Header().Set("Access-Control-Allow-Headers", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT")

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}
