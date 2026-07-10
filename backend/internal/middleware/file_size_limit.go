package middleware

import (
	"fmt"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humagin"
	"github.com/gin-gonic/gin"
	"github.com/pocket-id/pocket-id/backend/internal/common"
)

type FileSizeLimitMiddleware struct{}

func NewFileSizeLimitMiddleware() *FileSizeLimitMiddleware {
	return &FileSizeLimitMiddleware{}
}

// Huma returns a multipart size-limit middleware that preserves the existing error message
func (m *FileSizeLimitMiddleware) Huma(api huma.API, maxSize int64) func(huma.Context, func(huma.Context)) {
	return func(ctx huma.Context, next func(huma.Context)) {
		ginCtx := humagin.Unwrap(ctx)
		ginCtx.Request.Body = http.MaxBytesReader(ginCtx.Writer, ginCtx.Request.Body, maxSize)
		if err := ginCtx.Request.ParseMultipartForm(maxSize); err != nil {
			fileError := &common.FileTooLargeError{MaxSize: formatFileSize(maxSize)}
			_ = huma.WriteErr(api, ctx, fileError.HttpStatusCode(), fileError.Error())
			return
		}
		next(ctx)
	}
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
