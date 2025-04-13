package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	"github.com/pocket-id/pocket-id/backend/internal/service"
	"gorm.io/gorm"
)

// AuthMiddleware is a wrapper middleware that delegates to either API key or JWT authentication
type AuthMiddleware struct {
	apiKeyMiddleware *ApiKeyAuthMiddleware
	jwtMiddleware    *JwtAuthMiddleware
	options          AuthOptions
	db               *gorm.DB
}

type AuthOptions struct {
	AdminRequired   bool
	SuccessOptional bool
}

func NewAuthMiddleware(
	apiKeyService *service.ApiKeyService,
	jwtService *service.JwtService,
	db *gorm.DB,
) *AuthMiddleware {
	return &AuthMiddleware{
		apiKeyMiddleware: NewApiKeyAuthMiddleware(apiKeyService, jwtService),
		jwtMiddleware:    NewJwtAuthMiddleware(jwtService),
		options: AuthOptions{
			AdminRequired:   true,
			SuccessOptional: false,
		},
		db: db,
	}
}

// WithAdminNotRequired allows the middleware to continue with the request even if the user is not an admin
func (m *AuthMiddleware) WithAdminNotRequired() *AuthMiddleware {
	// Create a new instance to avoid modifying the original
	clone := &AuthMiddleware{
		apiKeyMiddleware: m.apiKeyMiddleware,
		jwtMiddleware:    m.jwtMiddleware,
		options:          m.options,
		db:               m.db,
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
		db:               m.db,
	}
	clone.options.SuccessOptional = true
	return clone
}

func (m *AuthMiddleware) Add() gin.HandlerFunc {
	return func(c *gin.Context) {
		// First try JWT auth
		userID, isAdmin, err := m.jwtMiddleware.Verify(c, m.options.AdminRequired)
		if err == nil {
			// JWT auth succeeded, continue with the request
			c.Set("userID", userID)
			c.Set("userIsAdmin", isAdmin)
			m.validateToken(c)
			if c.IsAborted() {
				return
			}
			c.Next()
			return
		}

		// JWT auth failed, try API key auth
		userID, isAdmin, err = m.apiKeyMiddleware.Verify(c, m.options.AdminRequired)
		if err == nil {
			// API key auth succeeded, continue with the request
			c.Set("userID", userID)
			c.Set("userIsAdmin", isAdmin)
			m.validateToken(c)
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

// validateToken checks if the user is disabled
func (am *AuthMiddleware) validateToken(c *gin.Context) {
	// After validating the token and extracting the user ID
	userID := c.GetString("userID")

	// Check if the user is disabled
	var user model.User
	err := am.db.
		WithContext(c.Request.Context()).
		Select("id, username, disabled").
		Where("id = ?", userID).
		First(&user).
		Error

	if err == nil && user.Disabled {
		c.AbortWithStatus(http.StatusForbidden)
		_ = c.Error(&common.UserDisabledError{})
		return
	}

	// Continue with your existing middleware flow
}
