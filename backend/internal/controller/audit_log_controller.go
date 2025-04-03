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

	group.GET("/audit-logs/all", authMiddleware.Add(), alc.listAllAuditLogsHandler)
	group.GET("/audit-logs", authMiddleware.WithAdminNotRequired().Add(), alc.listAuditLogsForUserHandler)
	group.GET("/audit-logs/filters/client-names", authMiddleware.Add(), alc.listClientNamesHandler)
	group.GET("/audit-logs/filters/users", authMiddleware.Add(), alc.listUserNamesWithIdsHandler)
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
// @Router /api/audit-logs [get]
func (alc *AuditLogController) listAuditLogsForUserHandler(c *gin.Context) {
	var sortedPaginationRequest utils.SortedPaginationRequest
	if err := c.ShouldBindQuery(&sortedPaginationRequest); err != nil {
		_ = c.Error(err)
		return
	}

	userID := c.GetString("userID")

	// Fetch audit logs for the user
	logs, pagination, err := alc.auditLogService.ListAuditLogsForUser(userID, sortedPaginationRequest)
	if err != nil {
		_ = c.Error(err)
		return
	}

	// Map the audit logs to DTOs
	var logsDtos []dto.AuditLogDto
	err = dto.MapStructList(logs, &logsDtos)
	if err != nil {
		_ = c.Error(err)
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

// listAllAuditLogsHandler godoc
// @Summary List all audit logs
// @Description Get a paginated list of all audit logs (admin only)
// @Tags Audit Logs
// @Param page query int false "Page number, starting from 1" default(1)
// @Param limit query int false "Number of items per page" default(10)
// @Param sort_column query string false "Column to sort by" default("created_at")
// @Param sort_direction query string false "Sort direction (asc or desc)" default("desc")
// @Param user_id query string false "Filter by user ID"
// @Param event query string false "Filter by event type"
// @Param client_name query string false "Filter by client name"
// @Success 200 {object} dto.Paginated[dto.AuditLogDto]
// @Router /api/audit-logs/all [get]
func (alc *AuditLogController) listAllAuditLogsHandler(c *gin.Context) {
	var sortedPaginationRequest utils.SortedPaginationRequest
	if err := c.ShouldBindQuery(&sortedPaginationRequest); err != nil {
		_ = c.Error(err)
		return
	}

	var filters dto.AuditLogFilterDto
	if err := c.ShouldBindQuery(&filters); err != nil {
		_ = c.Error(err)
		return
	}

	logs, pagination, err := alc.auditLogService.ListAllAuditLogs(sortedPaginationRequest, filters)
	if err != nil {
		_ = c.Error(err)
		return
	}

	var logsDtos []dto.AuditLogDto
	err = dto.MapStructList(logs, &logsDtos)
	if err != nil {
		_ = c.Error(err)
		return
	}

	for i, logsDto := range logsDtos {
		logsDto.Device = alc.auditLogService.DeviceStringFromUserAgent(logs[i].UserAgent)
		logsDto.Username = logs[i].User.Username
		logsDtos[i] = logsDto
	}

	c.JSON(http.StatusOK, dto.Paginated[dto.AuditLogDto]{
		Data:       logsDtos,
		Pagination: pagination,
	})
}

// listClientNamesHandler godoc
// @Summary List client names
// @Description Get a list of all client names for audit log filtering
// @Tags Audit Logs
// @Success 200 {array} string "List of client names"
// @Router /api/audit-logs/filters/client-names [get]
func (alc *AuditLogController) listClientNamesHandler(c *gin.Context) {
	names, err := alc.auditLogService.ListClientNames()
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, names)
}

// listUserNamesWithIdsHandler godoc
// @Summary List users with IDs
// @Description Get a list of all usernames with their IDs for audit log filtering
// @Tags Audit Logs
// @Success 200 {object} map[string]string "Map of user IDs to usernames"
// @Router /api/audit-logs/filters/users [get]
func (alc *AuditLogController) listUserNamesWithIdsHandler(c *gin.Context) {
	users, err := alc.auditLogService.ListUsernamesWithIds()
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, users)
}
