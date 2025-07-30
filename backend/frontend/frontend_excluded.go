//go:build exclude_frontend

package frontend

import (
	"github.com/gin-gonic/gin"

	"github.com/pocket-id/pocket-id/backend/internal/service"
)

func RegisterFrontend(router *gin.Engine, appConfigService *service.AppConfigService) error {
	return ErrFrontendNotIncluded
}
