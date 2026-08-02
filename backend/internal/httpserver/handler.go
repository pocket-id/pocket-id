package httpserver

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"
)

// HandlerFunc is an application HTTP handler that returns failures to the shared error middleware
type HandlerFunc func(*gin.Context) error

// Handle adapts an error-returning application handler to Gin
func Handle(handler HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			recovered := recover()
			if recovered == nil {
				return
			}

			// Preserve net/http's intentional request-abort behavior without logging it as an application panic
			if err, ok := recovered.(error); ok && errors.Is(err, http.ErrAbortHandler) {
				panic(recovered)
			}

			// Keep recovered errors out of the unwrap chain so every panic is reported as an internal failure
			err := fmt.Errorf("panic in HTTP handler (%T): %v\n%s", recovered, recovered, debug.Stack())
			_ = c.Error(err)
			c.Abort()
		}()

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
