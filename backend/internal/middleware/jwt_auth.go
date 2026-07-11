package middleware

import (
	"errors"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/service"
	"github.com/pocket-id/pocket-id/backend/internal/utils/cookie"
)

type JwtAuthMiddleware struct {
	userService     *service.UserService
	jwtService      *service.JwtService
	auditLogService *service.AuditLogService
}

func NewJwtAuthMiddleware(jwtService *service.JwtService, userService *service.UserService, auditLogService *service.AuditLogService) *JwtAuthMiddleware {
	return &JwtAuthMiddleware{
		jwtService:      jwtService,
		userService:     userService,
		auditLogService: auditLogService,
	}
}

func (m *JwtAuthMiddleware) Add(adminRequired bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, isAdmin, authenticationMethod, authenticationTime, err := m.Verify(c, adminRequired)
		if err != nil {
			c.Abort()
			_ = c.Error(err)
			return
		}

		c.Set("userID", userID)
		c.Set("userIsAdmin", isAdmin)
		c.Set("authenticationMethod", authenticationMethod)
		c.Set("authenticationTime", authenticationTime)
		c.Next()
	}
}

func (m *JwtAuthMiddleware) Verify(c *gin.Context, adminRequired bool) (subject string, isAdmin bool, authenticationMethod string, authenticationTime time.Time, err error) {
	var userID string
	// With a deferred call, check if the error indicates a sign in failure that needs to be logged
	// Important: in this method, do not created "err" variables with a narrower scope (e.g. avoid `if err := ...; err {` constructs)
	defer func() {
		if err != nil && !errors.Is(err, &common.NotSignedInError{}) {
			m.auditLogService.CreateSignInFailure(c, c.ClientIP(), c.Request.UserAgent(), userID)
		}
	}()

	// Extract the token from the cookie
	accessToken, err := c.Cookie(cookie.AccessTokenCookieName)
	if err != nil {
		// Try to extract the token from the Authorization header if it's not in the cookie
		var ok bool
		_, accessToken, ok = strings.Cut(c.GetHeader("Authorization"), " ")
		if !ok || accessToken == "" {
			return "", false, "", time.Time{}, &common.NotSignedInError{}
		}
	}

	token, err := m.jwtService.VerifyAccessToken(accessToken)
	if err != nil {
		return "", false, "", time.Time{}, &common.NotSignedInError{}
	}
	authenticationMethod, err = m.jwtService.GetAuthenticationMethod(token)
	if err != nil {
		return "", false, "", time.Time{}, &common.NotSignedInError{}
	}
	authenticationTime, _ = token.IssuedAt()

	subject, ok := token.Subject()
	if !ok {
		_ = c.Error(&common.TokenInvalidError{})
		return "", false, "", time.Time{}, &common.TokenInvalidError{}
	}

	user, err := m.userService.GetUser(c, subject)
	if err != nil {
		return "", false, "", time.Time{}, &common.NotSignedInError{}
	}

	// Assign the user ID to the userID variable that can be used for logs
	userID = user.ID
	if user.Disabled {
		return "", false, "", time.Time{}, &common.UserDisabledError{}
	}

	if adminRequired && !user.IsAdmin {
		return "", false, "", time.Time{}, &common.MissingPermissionError{}
	}

	return subject, user.IsAdmin, authenticationMethod, authenticationTime, nil
}
