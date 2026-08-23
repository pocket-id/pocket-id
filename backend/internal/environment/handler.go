package environment

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/pocket-id/pocket-id/backend/internal/common"
	_ "github.com/pocket-id/pocket-id/backend/internal/dto"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
)

type handler struct {
	service *Service
}

func newHandler(service *Service) *handler {
	return &handler{service: service}
}

// getLatestVersion godoc
// @Summary Get latest available version of Pocket ID
// @Tags Version
// @Produce json
// @Success 200 {object} map[string]string "Latest version information"
// @Failure default {object} dto.ErrorDto "Error"
// @Router /api/version/latest [get]
func (h *handler) getLatestVersion(c *gin.Context) error {
	tag, err := h.service.GetLatestVersion(c.Request.Context())
	if err != nil {
		return err
	}

	utils.SetCacheControlHeader(c, 5*time.Minute, 15*time.Minute)

	c.JSON(http.StatusOK, gin.H{
		"latestVersion": tag,
	})
	return nil
}

// getCurrentVersion godoc
// @Summary Get current deployed version of Pocket ID
// @Tags Version
// @Produce json
// @Success 200 {object} map[string]string "Current version information"
// @Failure default {object} dto.ErrorDto "Error"
// @Router /api/version/current [get]
func (h *handler) getCurrentVersion(c *gin.Context) error {
	c.JSON(http.StatusOK, gin.H{
		"currentVersion": common.Version,
	})
	return nil
}

// getSqliteStorageWarning godoc
// @Summary Get whether the SQLite storage warning should be shown
// @Description Reports whether Pocket ID found its SQLite database on a networked filesystem, which is unsupported and can lead to database corruption
// @Tags Storage
// @Produce json
// @Success 200 {object} map[string]bool "SQLite storage warning state"
// @Failure default {object} dto.ErrorDto "Error"
// @Router /api/storage/sqlite-warning [get]
func (h *handler) getSqliteStorageWarning(c *gin.Context) error {
	c.JSON(http.StatusOK, gin.H{
		"showWarning": h.service.ShowSQLiteStorageWarning(),
	})
	return nil
}
