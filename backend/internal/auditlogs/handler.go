package auditlogs

import (
	"cmp"
	"context"
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

// parseDateRange parses the optional from and to bounds, RFC 3339 times that bound the list by creation time
func parseDateRange(from, to string) (DateRange, error) {
	var r DateRange
	for name, bound := range map[string]struct {
		value  string
		target *time.Time
	}{"from": {from, &r.From}, "to": {to, &r.To}} {
		if bound.value == "" {
			continue
		}
		t, err := time.Parse(time.RFC3339, bound.value)
		if err != nil {
			return r, apperror.InvalidField(name, "invalid", "must be an RFC 3339 time")
		}
		*bound.target = t
	}
	return r, nil
}

func (h *handler) listAuditLogsForUserHandler(c *gin.Context) (dto.Paginated[auditLogDto], error) {
	listRequestOptions := utils.ParseListRequestOptions(c)
	dateRange, err := parseDateRange(c.Query("from"), c.Query("to"))
	if err != nil {
		return dto.Paginated[auditLogDto]{}, err
	}

	userID := authz.PrincipalFrom(c).UserID

	// Fetch audit logs for the user
	logs, pagination, err := h.service.ListAuditLogsForUser(c.Request.Context(), userID, dateRange, listRequestOptions)
	if err != nil {
		return dto.Paginated[auditLogDto]{}, err
	}

	logsDtos, err := h.toDtos(logs)
	if err != nil {
		return dto.Paginated[auditLogDto]{}, err
	}

	return dto.Paginated[auditLogDto]{
		Data:       logsDtos,
		Pagination: pagination,
	}, nil
}

func (h *handler) listAllAuditLogsHandler(c *gin.Context) (dto.Paginated[auditLogDto], error) {
	listRequestOptions := utils.ParseListRequestOptions(c)
	dateRange, err := parseDateRange(c.Query("from"), c.Query("to"))
	if err != nil {
		return dto.Paginated[auditLogDto]{}, err
	}

	return h.listAllAuditLogs(c.Request.Context(), dateRange, listRequestOptions)
}

func (h *handler) listAllAuditLogs(ctx context.Context, dateRange DateRange, listRequestOptions utils.ListRequestOptions) (dto.Paginated[auditLogDto], error) {
	logs, pagination, err := h.service.ListAllAuditLogs(ctx, dateRange, listRequestOptions)
	if err != nil {
		return dto.Paginated[auditLogDto]{}, err
	}

	logsDtos, err := h.toDtos(logs)
	if err != nil {
		return dto.Paginated[auditLogDto]{}, err
	}

	return dto.Paginated[auditLogDto]{Data: logsDtos, Pagination: pagination}, nil
}

func (h *handler) toDtos(logs []AuditLog) ([]auditLogDto, error) {
	var logsDtos []auditLogDto
	err := dto.MapStructList(logs, &logsDtos)
	if err != nil {
		return nil, err
	}

	for i, logsDto := range logsDtos {
		logsDto.Device = h.service.DeviceStringFromUserAgent(logs[i].UserAgent)
		logsDto.Username = cmp.Or(logs[i].User.Username, logsDto.Data["username"])
		logsDtos[i] = logsDto
	}
	return logsDtos, nil
}

func (h *handler) listClientNamesHandler(c *gin.Context) ([]string, error) {
	return h.service.ListClientNames(c.Request.Context())
}

func (h *handler) listUserNamesWithIdsHandler(c *gin.Context) (map[string]string, error) {
	return h.service.ListUsernamesWithIds(c.Request.Context())
}
