package middleware

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type IsHeadRequestCtxKey struct{}

type headWriter struct {
	gin.ResponseWriter
	size int
}

func (w *headWriter) Write(b []byte) (int, error) {
	w.size += len(b)
	return w.size, nil
}

func HeadMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Only process if it's a HEAD request
		if c.Request.Context().Value(IsHeadRequestCtxKey{}) != true {
			c.Next()
			return
		}

		// Replace the ResponseWriter with our headWriter to swallow the body
		hw := &headWriter{ResponseWriter: c.Writer}
		c.Writer = hw

		c.Next()

		c.Writer.Header().Set("Content-Length", strconv.Itoa(hw.size))
		c.Request.Method = http.MethodHead

	}
}
