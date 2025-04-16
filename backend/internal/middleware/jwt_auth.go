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
		userID, isAdmin, isDisabled, err := m.Verify(c, adminRequired)
		if err != nil {
			c.Abort()
			_ = c.Error(err)
			return
		}

		c.Set("userID", userID)
		c.Set("userIsAdmin", isAdmin)
		c.Set("userIsDisabled", isDisabled)
		c.Next()
	}
}

func (m *JwtAuthMiddleware) Verify(c *gin.Context, adminRequired bool) (subject string, isAdmin bool, isDisabled bool, err error) {
	// Extract the token from the cookie
	accessToken, err := c.Cookie(cookie.AccessTokenCookieName)
	if err != nil {
		// Try to extract the token from the Authorization header if it's not in the cookie
		var ok bool
		_, accessToken, ok = strings.Cut(c.GetHeader("Authorization"), " ")
		if !ok || accessToken == "" {
			return "", false, false, &common.NotSignedInError{}
		}
	}

	token, err := m.jwtService.VerifyAccessToken(accessToken)
	if err != nil {
		return "", false, false, &common.NotSignedInError{}
	}

	subject, ok := token.Subject()
	if !ok {
		_ = c.Error(&common.TokenInvalidError{})
		return
	}

	isAdmin, err = service.GetIsAdmin(token)
	if err != nil {
		return "", false, false, &common.TokenInvalidError{}
	}

	// Extract disabled claim
	isDisabled = false
	if token.Has("disabled") {
		_ = token.Get("disabled", &isDisabled)
	}

	if adminRequired && !isAdmin {
		return "", false, false, &common.MissingPermissionError{}
	}

	return subject, isAdmin, isDisabled, nil
}
