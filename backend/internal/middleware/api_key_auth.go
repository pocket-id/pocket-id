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

		if auth == "" || !strings.HasPrefix(auth, "Bearer ") {
			c.Next()
			return
		}

		token := strings.TrimPrefix(auth, "Bearer ")

		// First try to validate as API key
		if user, err := m.apiKeyService.ValidateApiKey(token); err == nil {
			c.Set("userID", user.ID)
			c.Set("userIsAdmin", user.IsAdmin)
			c.Next()
			return
		}

		// Otherwise, try standard JWT token validation
		claims, err := m.jwtService.VerifyAccessToken(token)
		if err != nil {
			c.Error(&common.NotSignedInError{})
			c.Abort()
			return
		}

		c.Set("userID", claims.Subject)
		c.Set("userIsAdmin", claims.IsAdmin)
		c.Next()
	}
}
