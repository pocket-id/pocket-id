package controller

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/middleware"
	"github.com/pocket-id/pocket-id/backend/internal/service"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
)

// NewVersionController registers version-related routes.
func NewVersionController(group *gin.RouterGroup, authMiddleware *middleware.AuthMiddleware, versionService *service.VersionService) {
	vc := &VersionController{versionService: versionService}
	group.GET("/version/latest", vc.getLatestVersionHandler)
	group.GET("/version/current", authMiddleware.WithAdminNotRequired().Add(), vc.getCurrentVersionHandler)
}

type VersionController struct {
	versionService *service.VersionService
}

// getLatestVersionHandler godoc
// @Summary Get latest available version of Pocket ID
// @Tags Version
// @Produce json
// @Success 200 {object} map[string]string "Latest version information"
// @Router /api/version/latest [get]
func (vc *VersionController) getLatestVersionHandler(c *gin.Context) {
	tag, err := vc.versionService.GetLatestVersion(c.Request.Context())
	if err != nil {
		_ = c.Error(err)
		return
	}

	utils.SetCacheControlHeader(c, 5*time.Minute, 15*time.Minute)

	c.JSON(http.StatusOK, gin.H{
		"latestVersion": tag,
	})
}

// getCurrentVersionHandler godoc
// @Summary Get current deployed version of Pocket ID
// @Tags Version
// @Produce json
// @Success 200 {object} map[string]string "Current version information"
// @Router /api/version/current [get]
func (vc *VersionController) getCurrentVersionHandler(c *gin.Context) {

	c.JSON(http.StatusOK, gin.H{
		"currentVersion": common.Version,
	})
}
