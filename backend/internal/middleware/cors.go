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
		if c.FullPath() == "/api/oidc/token" {
			c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		} else if c.FullPath() == "/api/oidc/introspect" {
			c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		} else {
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
