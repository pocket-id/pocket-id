//go:build exclude_frontend

package frontend

import (
	"context"

	"github.com/gin-gonic/gin"
)

func RegisterFrontend(_ *gin.Engine, _ func(ctx context.Context) (bool, error)) error {
	return ErrFrontendNotIncluded
}
