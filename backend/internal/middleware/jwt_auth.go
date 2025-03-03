package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/service"
	"github.com/pocket-id/pocket-id/backend/internal/utils/cookie"
)

type JwtAuthMiddleware struct {
	jwtService            *service.JwtService
	ignoreUnauthenticated bool
}

func NewJwtAuthMiddleware(jwtService *service.JwtService, ignoreUnauthenticated bool) *JwtAuthMiddleware {
	return &JwtAuthMiddleware{jwtService: jwtService, ignoreUnauthenticated: ignoreUnauthenticated}
}

func (m *JwtAuthMiddleware) Add(adminRequired bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		// If user is already authenticated (by API key middleware), skip JWT check
		if userID, exists := c.Get("userID"); exists && userID != "" {
			// User already authenticated via API key
			if adminRequired {
				// Check if admin access is required
				userIsAdmin, _ := c.Get("userIsAdmin")
				if userIsAdmin == true {
					c.Next()
					return
				}
				c.Error(&common.MissingPermissionError{})
				c.Abort()
				return
			}
			c.Next()
			return
		}

		// Extract the token from the cookie or the Authorization header
		token, err := c.Cookie(cookie.AccessTokenCookieName)
		if err != nil {
			authorizationHeaderSplitted := strings.Split(c.GetHeader("Authorization"), " ")
			if len(authorizationHeaderSplitted) == 2 {
				token = authorizationHeaderSplitted[1]
			} else if m.ignoreUnauthenticated {
				c.Next()
				return
			} else {
				c.Error(&common.NotSignedInError{})
				c.Abort()
				return
			}
		}

		claims, err := m.jwtService.VerifyAccessToken(token)
		if err != nil && m.ignoreUnauthenticated {
			c.Next()
			return
		} else if err != nil {
			c.Error(&common.NotSignedInError{})
			c.Abort()
			return
		}

		// Check if the user is an admin
		if adminRequired && !claims.IsAdmin {
			c.Error(&common.MissingPermissionError{})
			c.Abort()
			return
		}

		c.Set("userID", claims.Subject)
		c.Set("userIsAdmin", claims.IsAdmin)
		c.Next()
	}
}
