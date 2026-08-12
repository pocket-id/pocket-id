package controller

import (
	"net/http"

	"github.com/pocket-id/pocket-id/backend/internal/dto"
	"github.com/pocket-id/pocket-id/backend/internal/httpserver"
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

	group.GET("/audit-logs/all", authMiddleware.Add(), httpserver.Handle(alc.listAllAuditLogsHandler))
	group.GET("/audit-logs", authMiddleware.WithAdminNotRequired().Add(), httpserver.Handle(alc.listAuditLogsForUserHandler))
	group.GET("/audit-logs/filters/client-names", authMiddleware.Add(), httpserver.Handle(alc.listClientNamesHandler))
	group.GET("/audit-logs/filters/users", authMiddleware.Add(), httpserver.Handle(alc.listUserNamesWithIdsHandler))
}

type AuditLogController struct {
	auditLogService *service.AuditLogService
}

// listAuditLogsForUserHandler godoc
// @Summary List audit logs
// @Description Get a paginated list of audit logs for the current user
// @Tags Audit Logs
// @Param pagination[page] query int false "Page number for pagination" default(1)
// @Param pagination[limit] query int false "Number of items per page" default(20)
// @Param sort[column] query string false "Column to sort by"
// @Param sort[direction] query string false "Sort direction (asc or desc)" default("asc")
// @Success 200 {object} dto.Paginated[dto.AuditLogDto]
// @Failure default {object} dto.ErrorDto "Error"
// @Router /api/audit-logs [get]
func (alc *AuditLogController) listAuditLogsForUserHandler(c *gin.Context) error {
	listRequestOptions := utils.ParseListRequestOptions(c)

	userID := c.GetString("userID")

	// Fetch audit logs for the user
	logs, pagination, err := alc.auditLogService.ListAuditLogsForUser(c.Request.Context(), userID, listRequestOptions)
	if err != nil {
		return err
	}

	// Map the audit logs to DTOs
	var logsDtos []dto.AuditLogDto
	err = dto.MapStructList(logs, &logsDtos)
	if err != nil {
		return err
	}

	// Add device information to the logs
	for i, logsDto := range logsDtos {
		logsDto.Device = alc.auditLogService.DeviceStringFromUserAgent(logs[i].UserAgent)
		logsDto.ActorUsername = logsDto.Data["actorUsername"]
		logsDtos[i] = logsDto
	}

	c.JSON(http.StatusOK, dto.Paginated[dto.AuditLogDto]{
		Data:       logsDtos,
		Pagination: pagination,
	})
	return nil
}

// listAllAuditLogsHandler godoc
// @Summary List all audit logs
// @Description Get a paginated list of all audit logs (admin only)
// @Tags Audit Logs
// @Param pagination[page] query int false "Page number for pagination" default(1)
// @Param pagination[limit] query int false "Number of items per page" default(20)
// @Param sort[column] query string false "Column to sort by"
// @Param sort[direction] query string false "Sort direction (asc or desc)" default("asc")
// @Success 200 {object} dto.Paginated[dto.AuditLogDto]
// @Failure default {object} dto.ErrorDto "Error"
// @Router /api/audit-logs/all [get]
func (alc *AuditLogController) listAllAuditLogsHandler(c *gin.Context) error {
	listRequestOptions := utils.ParseListRequestOptions(c)

	logs, pagination, err := alc.auditLogService.ListAllAuditLogs(c.Request.Context(), listRequestOptions)
	if err != nil {
		return err
	}

	var logsDtos []dto.AuditLogDto
	err = dto.MapStructList(logs, &logsDtos)
	if err != nil {
		return err
	}

	for i, logsDto := range logsDtos {
		logsDto.Device = alc.auditLogService.DeviceStringFromUserAgent(logs[i].UserAgent)
		logsDto.Username = logs[i].User.Username
		logsDto.ActorUsername = logsDto.Data["actorUsername"]
		logsDtos[i] = logsDto
	}

	c.JSON(http.StatusOK, dto.Paginated[dto.AuditLogDto]{
		Data:       logsDtos,
		Pagination: pagination,
	})
	return nil
}

// listClientNamesHandler godoc
// @Summary List client names
// @Description Get a list of all client names for audit log filtering
// @Tags Audit Logs
// @Success 200 {array} string "List of client names"
// @Failure default {object} dto.ErrorDto "Error"
// @Router /api/audit-logs/filters/client-names [get]
func (alc *AuditLogController) listClientNamesHandler(c *gin.Context) error {
	names, err := alc.auditLogService.ListClientNames(c.Request.Context())
	if err != nil {
		return err
	}

	c.JSON(http.StatusOK, names)
	return nil
}

// listUserNamesWithIdsHandler godoc
// @Summary List users with IDs
// @Description Get a list of all usernames with their IDs for audit log filtering
// @Tags Audit Logs
// @Success 200 {object} map[string]string "Map of user IDs to usernames"
// @Failure default {object} dto.ErrorDto "Error"
// @Router /api/audit-logs/filters/users [get]
func (alc *AuditLogController) listUserNamesWithIdsHandler(c *gin.Context) error {
	users, err := alc.auditLogService.ListUsernamesWithIds(c.Request.Context())
	if err != nil {
		return err
	}

	c.JSON(http.StatusOK, users)
	return nil
}
