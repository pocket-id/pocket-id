package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/pocket-id/pocket-id/backend/internal/service"
)

type ApiKeyAuthMiddleware struct {
	apiKeyService *service.ApiKeyService
	jwtService    *service.JwtService
}

func NewApiKeyAuthMiddleware(apiKeyService *service.ApiKeyService, jwtService *service.JwtService) *ApiKeyAuthMiddleware {
	return &ApiKeyAuthMiddleware{
		apiKeyService: apiKeyService,
		jwtService:    jwtService,
	}
}

func (m *ApiKeyAuthMiddleware) Add() gin.HandlerFunc {
	return func(c *gin.Context) {
		auth := c.GetHeader("Authorization")

		// If no Authorization header, just continue to the next middleware
		if auth == "" || !strings.HasPrefix(auth, "Bearer ") {
			c.Next()
			return
		}

		token := strings.TrimPrefix(auth, "Bearer ")

		// Check if token looks like a JWT (has 3 parts separated by dots)
		// If it does, let the JWT middleware handle it
		if strings.Count(token, ".") == 2 {
			c.Next()
			return
		}

		// Not a JWT format, so try to validate as API key
		user, err := m.apiKeyService.ValidateApiKey(token)
		if err != nil {
			// Not a valid API key, let the request continue to JWT auth
			// which will handle the error if needed
			c.Next()
			return
		}

		// API key is valid, set user context
		c.Set("userID", user.ID)
		c.Set("userIsAdmin", user.IsAdmin)
		c.Next()
	}
}
