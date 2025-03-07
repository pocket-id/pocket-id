package middleware

import (
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

func (m *ApiKeyAuthMiddleware) Add(adminRequired bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, isAdmin, err := m.Verify(c, adminRequired)
		if err != nil {
			c.Abort()
			c.Error(err)
			return
		}

		c.Set("userID", userID)
		c.Set("userIsAdmin", isAdmin)
		c.Next()
	}
}

func (m *ApiKeyAuthMiddleware) Verify(c *gin.Context, adminRequired bool) (userID string, isAdmin bool, err error) {
	apiKey := c.GetHeader("X-API-KEY")

	user, err := m.apiKeyService.ValidateApiKey(apiKey)
	if err != nil {
		return "", false, &common.NotSignedInError{}
	}

	// Check if the user is an admin
	if adminRequired && !user.IsAdmin {
		return "", false, &common.MissingPermissionError{}
	}

	return user.ID, user.IsAdmin, nil
}
