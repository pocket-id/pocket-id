package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/pocket-id/pocket-id/backend/internal/common"
	_ "github.com/pocket-id/pocket-id/backend/internal/dto"
	"github.com/pocket-id/pocket-id/backend/internal/httpserver"
	"github.com/pocket-id/pocket-id/backend/internal/middleware"
)

// NewStorageController registers storage-related routes.
func NewStorageController(group *gin.RouterGroup, authMiddleware *middleware.AuthMiddleware) {
	sc := &StorageController{}
	group.GET("/storage/sqlite-warning", authMiddleware.WithAdminNotRequired().Add(), httpserver.Handle(sc.getSqliteWarningHandler))
}

type StorageController struct{}

// getSqliteWarningHandler godoc
// @Summary Get whether the SQLite storage warning should be shown
// @Description Reports whether Pocket ID detected its SQLite database on a networked filesystem, which is unsupported and can lead to database corruption
// @Tags Storage
// @Produce json
// @Success 200 {object} map[string]bool "SQLite storage warning state"
// @Failure default {object} dto.ErrorDto "Error"
// @Router /api/storage/sqlite-warning [get]
func (sc *StorageController) getSqliteWarningHandler(c *gin.Context) error {
	c.JSON(http.StatusOK, gin.H{
		"showWarning": common.ShowSQLiteStorageWarning(),
	})
	return nil
}
