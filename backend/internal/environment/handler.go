package environment

import (
	"time"

	"github.com/gin-gonic/gin"

	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
)

type latestVersionDto struct {
	LatestVersion string `json:"latestVersion"`
}

type currentVersionDto struct {
	CurrentVersion string `json:"currentVersion"`
}

type sqliteStorageWarningDto struct {
	ShowWarning bool `json:"showWarning"`
}

type handler struct {
	service *Service
}

func newHandler(service *Service) *handler {
	return &handler{service: service}
}

func (h *handler) getLatestVersion(c *gin.Context) (latestVersionDto, error) {
	tag, err := h.service.GetLatestVersion(c.Request.Context())
	if err != nil {
		return latestVersionDto{}, err
	}

	utils.SetCacheControlHeader(c, 5*time.Minute, 15*time.Minute)

	return latestVersionDto{LatestVersion: tag}, nil
}

func (h *handler) getCurrentVersion(c *gin.Context) (currentVersionDto, error) {
	return currentVersionDto{CurrentVersion: common.Version}, nil
}

func (h *handler) getSqliteStorageWarning(c *gin.Context) (sqliteStorageWarningDto, error) {
	return sqliteStorageWarningDto{ShowWarning: h.service.ShowSQLiteStorageWarning()}, nil
}
