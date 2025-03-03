package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/pocket-id/pocket-id/backend/internal/common"
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

		// First try to validate as API key
		user, err := m.apiKeyService.ValidateApiKey(token)
		if err == nil {
			// API key is valid, set user context
			c.Set("userID", user.ID)
			c.Set("userIsAdmin", user.IsAdmin)
			c.Next()
			return
		}

		// If not an API key, try JWT
		claims, err := m.jwtService.VerifyAccessToken(token)
		if err != nil {
			// Neither API key nor JWT worked
			c.Error(&common.NotSignedInError{})
			c.Abort()
			return
		}

		// JWT worked
		c.Set("userID", claims.Subject)
		c.Set("userIsAdmin", claims.IsAdmin)
		c.Next()
	}
}
