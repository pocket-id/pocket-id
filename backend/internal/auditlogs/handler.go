package auditlogs

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/pocket-id/pocket-id/backend/internal/dto"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
)

type handler struct {
	service *service
}

func newHandler(service *service) *handler {
	return &handler{service: service}
}

// listAuditLogsForUserHandler godoc
// @Summary List audit logs
// @Description Get a paginated list of audit logs for the current user
// @Tags Audit Logs
// @Param pagination[page] query int false "Page number for pagination" default(1)
// @Param pagination[limit] query int false "Number of items per page" default(20)
// @Param sort[column] query string false "Column to sort by"
// @Param sort[direction] query string false "Sort direction (asc or desc)" default("asc")
// @Success 200 {object} dto.Paginated[auditLogDto]
// @Failure default {object} dto.ErrorDto "Error"
// @Router /api/audit-logs [get]
func (h *handler) listAuditLogsForUserHandler(c *gin.Context) error {
	listRequestOptions := utils.ParseListRequestOptions(c)

	userID := c.GetString("userID")

	// Fetch audit logs for the user
	logs, pagination, err := h.service.ListAuditLogsForUser(c.Request.Context(), userID, listRequestOptions)
	if err != nil {
		return err
	}

	// Map the audit logs to DTOs
	var logsDtos []auditLogDto
	err = dto.MapStructList(logs, &logsDtos)
	if err != nil {
		return err
	}

	// Add device information to the logs
	for i, logsDto := range logsDtos {
		logsDto.Device = h.service.DeviceStringFromUserAgent(logs[i].UserAgent)
		logsDto.ActorUsername = logsDto.Data["actorUsername"]
		logsDtos[i] = logsDto
	}

	c.JSON(http.StatusOK, dto.Paginated[auditLogDto]{
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
// @Success 200 {object} dto.Paginated[auditLogDto]
// @Failure default {object} dto.ErrorDto "Error"
// @Router /api/audit-logs/all [get]
func (h *handler) listAllAuditLogsHandler(c *gin.Context) error {
	listRequestOptions := utils.ParseListRequestOptions(c)

	logs, pagination, err := h.service.ListAllAuditLogs(c.Request.Context(), listRequestOptions)
	if err != nil {
		return err
	}

	var logsDtos []auditLogDto
	err = dto.MapStructList(logs, &logsDtos)
	if err != nil {
		return err
	}

	for i, logsDto := range logsDtos {
		logsDto.Device = h.service.DeviceStringFromUserAgent(logs[i].UserAgent)
		logsDto.Username = logs[i].User.Username
		logsDto.ActorUsername = logsDto.Data["actorUsername"]
		logsDtos[i] = logsDto
	}

	c.JSON(http.StatusOK, dto.Paginated[auditLogDto]{
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
func (h *handler) listClientNamesHandler(c *gin.Context) error {
	names, err := h.service.ListClientNames(c.Request.Context())
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
func (h *handler) listUserNamesWithIdsHandler(c *gin.Context) error {
	users, err := h.service.ListUsernamesWithIds(c.Request.Context())
	if err != nil {
		return err
	}

	c.JSON(http.StatusOK, users)
	return nil
}
