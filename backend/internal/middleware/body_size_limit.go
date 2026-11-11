package middleware

import (
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
)

// unlimitedBodyKey holds the request body from before the global limit, so a route that accepts uploads can allow more
const unlimitedBodyKey = "unlimitedRequestBody"

type BodySizeLimitMiddleware struct{}

func NewBodySizeLimitMiddleware() *BodySizeLimitMiddleware {
	return &BodySizeLimitMiddleware{}
}

// Add caps every request body, so no endpoint can be made to parse, buffer or store unbounded input
func (m *BodySizeLimitMiddleware) Add(maxSize int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set(unlimitedBodyKey, c.Request.Body)
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxSize)
		c.Next()
	}
}

// unlimitedBody returns the request body without the global limit
func unlimitedBody(c *gin.Context) io.ReadCloser {
	if body, ok := c.Get(unlimitedBodyKey); ok {
		return body.(io.ReadCloser)
	}
	return c.Request.Body
}
