package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
)

// CspMiddleware sets a Content Security Policy header and, when possible,
// includes a per-request nonce for inline scripts.
type CspMiddleware struct{}

func NewCspMiddleware() *CspMiddleware { return &CspMiddleware{} }

// GetCSPNonce returns the CSP nonce generated for this request, if any.
func GetCSPNonce(c *gin.Context) string {
	return utils.GetCSPNonce(c)
}

func (m *CspMiddleware) Add() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Generate a random base64 nonce for this request
		nonce := utils.GenerateCSPNonce()
		utils.SetCSPNonce(c, nonce)
		c.Writer.Header().Set("Content-Security-Policy", utils.BuildCSP(nonce))

		c.Next()
	}
}
