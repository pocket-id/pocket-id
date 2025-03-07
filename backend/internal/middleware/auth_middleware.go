package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/pocket-id/pocket-id/backend/internal/service"
)

// AuthMiddleware is a wrapper middleware that delegates to either API key or JWT authentication
type AuthMiddleware struct {
	apiKeyMiddleware *ApiKeyAuthMiddleware
	jwtMiddleware    *JwtAuthMiddleware
}

func NewAuthMiddleware(
	apiKeyService *service.ApiKeyService,
	jwtService *service.JwtService,
) *AuthMiddleware {
	return &AuthMiddleware{
		apiKeyMiddleware: NewApiKeyAuthMiddleware(apiKeyService, jwtService),
		jwtMiddleware:    NewJwtAuthMiddleware(jwtService),
	}
}

func (m *AuthMiddleware) Add(adminRequired bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		// First try JWT auth
		userID, isAdmin, err := m.jwtMiddleware.Verify(c, adminRequired)
		if err == nil {
			// JWT auth succeeded, continue with the request
			c.Set("userID", userID)
			c.Set("userIsAdmin", isAdmin)
			c.Next()
			return
		}

		// JWT auth failed, try API key auth
		userID, isAdmin, err = m.apiKeyMiddleware.Verify(c, adminRequired)
		if err == nil {
			// API key auth succeeded, continue with the request
			c.Set("userID", userID)
			c.Set("userIsAdmin", isAdmin)
			c.Next()
			return
		}

		// Both JWT and API key auth failed
		c.Abort()
		c.Error(err)
	}
}
