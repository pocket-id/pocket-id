package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/service"
	"github.com/pocket-id/pocket-id/backend/internal/utils/cookie"
)

type JwtAuthMiddleware struct {
	jwtService *service.JwtService
}

func NewJwtAuthMiddleware(jwtService *service.JwtService) *JwtAuthMiddleware {
	return &JwtAuthMiddleware{jwtService: jwtService}
}

func (m *JwtAuthMiddleware) Add(adminRequired bool) gin.HandlerFunc {
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

func (m *JwtAuthMiddleware) Verify(c *gin.Context, adminRequired bool) (userID string, isAdmin bool, err error) {
	// Extract the token from the cookie
	token, err := c.Cookie(cookie.AccessTokenCookieName)
	if err != nil {
		// Try to extract the token from the Authorization header if it's not in the cookie
		authorizationHeaderSplit := strings.Split(c.GetHeader("Authorization"), " ")
		if len(authorizationHeaderSplit) != 2 {
			return "", false, &common.NotSignedInError{}
		}
		token = authorizationHeaderSplit[1]
	}

	claims, err := m.jwtService.VerifyAccessToken(token)
	if err != nil {
		return "", false, &common.NotSignedInError{}
	}

	// Check if the user is an admin
	if adminRequired && !claims.IsAdmin {
		return "", false, &common.MissingPermissionError{}
	}

	return claims.Subject, claims.IsAdmin, nil
}
