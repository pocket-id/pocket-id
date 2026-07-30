package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/pocket-id/pocket-id/backend/internal/apikey"
	"github.com/pocket-id/pocket-id/backend/internal/apperror"
	"github.com/pocket-id/pocket-id/backend/internal/service"
)

type ApiKeyAuthMiddleware struct {
	apiKeyModule *apikey.Module
	jwtService   *service.JwtService
}

func NewApiKeyAuthMiddleware(apiKeyModule *apikey.Module, jwtService *service.JwtService) *ApiKeyAuthMiddleware {
	return &ApiKeyAuthMiddleware{
		apiKeyModule: apiKeyModule,
		jwtService:   jwtService,
	}
}

func (m *ApiKeyAuthMiddleware) Add(adminRequired bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, isAdmin, err := m.Verify(c, adminRequired)
		if err != nil {
			c.Abort()
			_ = c.Error(err)
			return
		}

		c.Set("userID", userID)
		c.Set("userIsAdmin", isAdmin)
		c.Next()
	}
}

func (m *ApiKeyAuthMiddleware) Verify(c *gin.Context, adminRequired bool) (userID string, isAdmin bool, err error) {
	apiKey := c.GetHeader("X-API-Key")

	user, err := m.apiKeyModule.ValidateApiKey(c.Request.Context(), apiKey)
	if err != nil {
		return "", false, apperror.NotSignedIn()
	}

	if user.Disabled {
		return "", false, apperror.UserDisabled()
	}

	if adminRequired && !user.IsAdmin {
		return "", false, apperror.MissingPermission()
	}

	return user.ID, user.IsAdmin, nil
}
