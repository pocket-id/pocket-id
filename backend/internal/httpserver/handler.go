package httpserver

import (
	"context"
	"errors"

	"github.com/gin-gonic/gin"
)

// HandlerFunc is an application HTTP handler that returns failures to the shared error middleware
type HandlerFunc func(*gin.Context) error

// Handle adapts an error-returning application handler to Gin
func Handle(handler HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := handler(c); err != nil {
			if errors.Is(err, context.Canceled) {
				c.Abort()
				return
			}

			_ = c.Error(err)
			c.Abort()
		}
	}
}
