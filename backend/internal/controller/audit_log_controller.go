package controller

import (
	"net/http"

	"github.com/pocket-id/pocket-id/backend/internal/dto"
	"github.com/pocket-id/pocket-id/backend/internal/middleware"
	"github.com/pocket-id/pocket-id/backend/internal/utils"

	"github.com/gin-gonic/gin"
	"github.com/pocket-id/pocket-id/backend/internal/service"
)

func NewAuditLogController(group *gin.RouterGroup, auditLogService *service.AuditLogService, authMiddleware *middleware.AuthMiddleware) {
	alc := AuditLogController{
		auditLogService: auditLogService,
	}

	group.GET("/audit-logs", authMiddleware.Add(false), alc.listAuditLogsForUserHandler)
}

type AuditLogController struct {
	auditLogService *service.AuditLogService
}

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

	c.JSON(http.StatusOK, gin.H{
		"data":       logsDtos,
		"pagination": pagination,
	})
}
