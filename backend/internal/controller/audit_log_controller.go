package controller

import (
	"net/http"

	"github.com/pocket-id/pocket-id/backend/internal/dto"
	"github.com/pocket-id/pocket-id/backend/internal/middleware"
	"github.com/pocket-id/pocket-id/backend/internal/utils"

	"github.com/gin-gonic/gin"
	"github.com/pocket-id/pocket-id/backend/internal/service"
)

// NewAuditLogController creates a new controller for audit log management
// @Summary Audit log controller
// @Description Initializes API endpoints for accessing audit logs
// @Tags Audit Logs
func NewAuditLogController(group *gin.RouterGroup, auditLogService *service.AuditLogService, authMiddleware *middleware.AuthMiddleware) {
	alc := AuditLogController{
		auditLogService: auditLogService,
	}

	group.GET("/audit-logs", authMiddleware.WithAdminNotRequired().Add(), alc.listAuditLogsForUserHandler)
}

type AuditLogController struct {
	auditLogService *service.AuditLogService
}

// listAuditLogsForUserHandler godoc
// @Summary List audit logs
// @Description Get a paginated list of audit logs for the current user
// @Tags Audit Logs
// @Param page query int false "Page number, starting from 1" default(1)
// @Param limit query int false "Number of items per page" default(10)
// @Param sort_column query string false "Column to sort by" default("created_at")
// @Param sort_direction query string false "Sort direction (asc or desc)" default("desc")
// @Success 200 {object} dto.Paginated[dto.AuditLogDto]
// @Router /audit-logs [get]
func (alc *AuditLogController) listAuditLogsForUserHandler(c *gin.Context) {
	var sortedPaginationRequest utils.SortedPaginationRequest
	if err := c.ShouldBindQuery(&sortedPaginationRequest); err != nil {
		c.Error(err)
		return
	}

	userID := c.GetString("userID")

	// Fetch audit logs for the user
	logs, pagination, err := alc.auditLogService.ListAuditLogsForUser(userID, sortedPaginationRequest)
	if err != nil {
		c.Error(err)
		return
	}

	// Map the audit logs to DTOs
	var logsDtos []dto.AuditLogDto
	err = dto.MapStructList(logs, &logsDtos)
	if err != nil {
		c.Error(err)
		return
	}

	// Add device information to the logs
	for i, logsDto := range logsDtos {
		logsDto.Device = alc.auditLogService.DeviceStringFromUserAgent(logs[i].UserAgent)
		logsDtos[i] = logsDto
	}

	c.JSON(http.StatusOK, dto.Paginated[dto.AuditLogDto]{
		Data:       logsDtos,
		Pagination: pagination,
	})
}
