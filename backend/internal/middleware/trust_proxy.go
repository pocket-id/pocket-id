package middleware

import (
	"net"
	"strings"

	"github.com/gin-gonic/gin"
)

const ClientIPKey = "client_ip" // Key for storing real client IP in context

type TrustProxyMiddleware struct{}

func NewTrustProxyMiddleware() *TrustProxyMiddleware {
	return &TrustProxyMiddleware{}
}

func (m *TrustProxyMiddleware) Add() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Always try to extract client IP from headers
		var clientIP string

		if xff := c.Request.Header.Get("X-Forwarded-For"); xff != "" {
			// X-Forwarded-For can contain multiple IPs, take the first one which is the client
			ips := strings.Split(xff, ",")
			if len(ips) > 0 {
				clientIP = strings.TrimSpace(ips[0])
				if net.ParseIP(clientIP) != nil {
					// Valid IP found in X-Forwarded-For
					c.Set(ClientIPKey, clientIP)
					// Override Gin's ClientIP() result
					c.Request.Header.Set("X-Real-IP", clientIP)
				}
			}
		} else if xrip := c.Request.Header.Get("X-Real-IP"); xrip != "" {
			if net.ParseIP(xrip) != nil {
				// Valid IP found in X-Real-IP
				c.Set(ClientIPKey, xrip)
			}
		}

		c.Next()
	}
}
