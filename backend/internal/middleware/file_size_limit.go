package middleware

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/pocket-id/pocket-id/backend/internal/common"
)

type FileSizeLimitMiddleware struct{}

func NewFileSizeLimitMiddleware() *FileSizeLimitMiddleware {
	return &FileSizeLimitMiddleware{}
}

func (m *FileSizeLimitMiddleware) Add(maxSize int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxSize)
		if err := c.Request.ParseMultipartForm(maxSize); err != nil {
			err = &common.FileTooLargeError{MaxSize: formatFileSize(maxSize)}
			_ = c.Error(err)
			c.Abort()
			return
		}
		c.Next()
	}
}

// formatFileSize formats a file size in bytes to a human-readable string
func formatFileSize(size int64) string {
	const (
		KB = 1 << (10 * 1)
		MB = 1 << (10 * 2)
		GB = 1 << (10 * 3)
	)

	switch {
	case size >= GB:
		return fmt.Sprintf("%.2f GB", float64(size)/GB)
	case size >= MB:
		return fmt.Sprintf("%.2f MB", float64(size)/MB)
	case size >= KB:
		return fmt.Sprintf("%.2f KB", float64(size)/KB)
	default:
		return fmt.Sprintf("%d bytes", size)
	}
}
