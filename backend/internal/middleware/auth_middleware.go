package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/pocket-id/pocket-id/backend/internal/service"
)

// AuthMiddleware is a wrapper middleware that delegates to either API key or JWT authentication
type AuthMiddleware struct {
	apiKeyMiddleware *ApiKeyAuthMiddleware
	jwtMiddleware    *JwtAuthMiddleware
	options          AuthOptions
}

type AuthOptions struct {
	AdminRequired   bool
	SuccessOptional bool
}

func NewAuthMiddleware(
	apiKeyService *service.ApiKeyService,
	jwtService *service.JwtService,
) *AuthMiddleware {
	return &AuthMiddleware{
		apiKeyMiddleware: NewApiKeyAuthMiddleware(apiKeyService, jwtService),
		jwtMiddleware:    NewJwtAuthMiddleware(jwtService),
		options: AuthOptions{
			AdminRequired:   true,
			SuccessOptional: false,
		},
	}
}

// WithAdminNotRequired allows the middleware to continue with the request even if the user is not an admin
func (m *AuthMiddleware) WithAdminNotRequired() *AuthMiddleware {
	// Create a new instance to avoid modifying the original
	clone := &AuthMiddleware{
		apiKeyMiddleware: m.apiKeyMiddleware,
		jwtMiddleware:    m.jwtMiddleware,
		options:          m.options,
	}
	clone.options.AdminRequired = false
	return clone
}

// WithSuccessOptional allows the middleware to continue with the request even if authentication fails
func (m *AuthMiddleware) WithSuccessOptional() *AuthMiddleware {
	// Create a new instance to avoid modifying the original
	clone := &AuthMiddleware{
		apiKeyMiddleware: m.apiKeyMiddleware,
		jwtMiddleware:    m.jwtMiddleware,
		options:          m.options,
	}
	clone.options.SuccessOptional = true
	return clone
}

func (m *AuthMiddleware) Add() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, isAdmin, isDisabled, err := m.jwtMiddleware.Verify(c, m.options.AdminRequired)
		if err == nil {
			if isDisabled {
				c.AbortWithStatusJSON(403, gin.H{"error": "User account is disabled"})
				return
			}
			c.Set("userID", userID)
			c.Set("userIsAdmin", isAdmin)
			if c.IsAborted() {
				return
			}
			c.Next()
			return
		}

		// JWT auth failed, try API key auth
		userID, isAdmin, isDisabled, err = m.apiKeyMiddleware.Verify(c, m.options.AdminRequired)
		if err == nil {
			if isDisabled {
				c.AbortWithStatusJSON(403, gin.H{"error": "User account is disabled"})
				return
			}
			c.Set("userID", userID)
			c.Set("userIsAdmin", isAdmin)
			if c.IsAborted() {
				return
			}
			c.Next()
			return
		}

		if m.options.SuccessOptional {
			c.Next()
			return
		}

		// Both JWT and API key auth failed
		c.Abort()
		_ = c.Error(err)
	}
}
