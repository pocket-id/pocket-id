//go:build exclude_frontend

package frontend

import (
	"context"

	"github.com/gin-gonic/gin"
)

func RegisterFrontend(router *gin.Engine, animationsDisabled func(ctx context.Context) bool) error {
	return ErrFrontendNotIncluded
}
