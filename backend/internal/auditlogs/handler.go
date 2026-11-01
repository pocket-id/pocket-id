package auditlogs

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/pocket-id/pocket-id/backend/internal/apperror"
	"github.com/pocket-id/pocket-id/backend/internal/authz"
	"github.com/pocket-id/pocket-id/backend/internal/dto"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
)

type handler struct {
	service *service
}

func newHandler(service *service) *handler {
	return &handler{service: service}
}

// parseDateRange reads the optional from and to query parameters, RFC 3339 times that bound the list by creation time
func parseDateRange(c *gin.Context) (DateRange, error) {
	var r DateRange
	for name, bound := range map[string]*time.Time{"from": &r.From, "to": &r.To} {
		value := c.Query(name)
		if value == "" {
			continue
		}
		t, err := time.Parse(time.RFC3339, value)
		if err != nil {
			return r, apperror.InvalidField(name, "invalid", "must be an RFC 3339 time")
		}
		*bound = t
	}
	return r, nil
}

// listAuditLogsForUserHandler godoc
// @Summary List audit logs
// @Description Get a paginated list of audit logs for the current user
// @Tags Audit Logs
// @Param pagination[page] query int false "Page number for pagination" default(1)
// @Param pagination[limit] query int false "Number of items per page" default(20)
// @Param sort[column] query string false "Column to sort by"
// @Param sort[direction] query string false "Sort direction (asc or desc)" default("asc")
// @Param from query string false "Only entries created at or after this RFC 3339 time"
// @Param to query string false "Only entries created before this RFC 3339 time"
// @Success 200 {object} dto.Paginated[auditLogDto]
// @Failure default {object} dto.ErrorDto "Error"
// @Router /api/audit-logs [get]
func (h *handler) listAuditLogsForUserHandler(c *gin.Context) error {
	listRequestOptions := utils.ParseListRequestOptions(c)
	dateRange, err := parseDateRange(c)
	if err != nil {
		return err
	}

	userID := authz.PrincipalFrom(c).UserID

	// Fetch audit logs for the user
	logs, pagination, err := h.service.ListAuditLogsForUser(c.Request.Context(), userID, dateRange, listRequestOptions)
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
// @Param from query string false "Only entries created at or after this RFC 3339 time"
// @Param to query string false "Only entries created before this RFC 3339 time"
// @Success 200 {object} dto.Paginated[auditLogDto]
// @Failure default {object} dto.ErrorDto "Error"
// @Router /api/audit-logs/all [get]
func (h *handler) listAllAuditLogsHandler(c *gin.Context) error {
	listRequestOptions := utils.ParseListRequestOptions(c)
	dateRange, err := parseDateRange(c)
	if err != nil {
		return err
	}

	logs, pagination, err := h.service.ListAllAuditLogs(c.Request.Context(), dateRange, listRequestOptions)
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
