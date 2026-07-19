package middleware

import (
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/service"
	"github.com/pocket-id/pocket-id/backend/internal/utils/cookie"
)

type JwtAuthMiddleware struct {
	userService *service.UserService
	jwtService  *service.JwtService
}

func NewJwtAuthMiddleware(jwtService *service.JwtService, userService *service.UserService) *JwtAuthMiddleware {
	return &JwtAuthMiddleware{jwtService: jwtService, userService: userService}
}

func (m *JwtAuthMiddleware) Verify(c *gin.Context, adminRequired bool, includeIsolatedToken bool) (subject string, isAdmin bool, authenticationMethod string, authenticationTime time.Time, permittedClients string, err error) {
	// Extract the token from the cookie
	accessToken, err := c.Cookie(cookie.AccessTokenCookieName)
	if err != nil {
		// Try to extract the token from the Authorization header if it's not in the cookie
		var ok bool
		_, accessToken, ok = strings.Cut(c.GetHeader("Authorization"), " ")
		if !ok || accessToken == "" {
			return "", false, "", time.Time{}, "", &common.NotSignedInError{}
		}
	}

	token, err := m.jwtService.VerifyAccessTokenWithIsolated(accessToken, includeIsolatedToken)
	if err != nil {
		return "", false, "", time.Time{}, "", &common.NotSignedInError{}
	}

	permittedClients, err = m.jwtService.GetPermittedClients(token)
	if err != nil {
		return "", false, "", time.Time{}, "", &common.NotSignedInError{}
	}

	authenticationMethod, err = m.jwtService.GetAuthenticationMethod(token)
	if err != nil {
		return "", false, "", time.Time{}, "", &common.NotSignedInError{}
	}
	authenticationTime, _ = token.IssuedAt()

	subject, ok := token.Subject()
	if !ok {
		_ = c.Error(&common.TokenInvalidError{})
		return "", false, "", time.Time{}, "", &common.TokenInvalidError{}
	}

	user, err := m.userService.GetUser(c, subject)
	if err != nil {
		return "", false, "", time.Time{}, "", &common.NotSignedInError{}
	}

	if user.Disabled {
		return "", false, "", time.Time{}, "", &common.UserDisabledError{}
	}

	if adminRequired && !user.IsAdmin {
		return "", false, "", time.Time{}, "", &common.MissingPermissionError{}
	}

	return subject, user.IsAdmin, authenticationMethod, authenticationTime, permittedClients, nil
}
