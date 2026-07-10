package controller

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/pocket-id/pocket-id/backend/internal/dto"
	"github.com/pocket-id/pocket-id/backend/internal/middleware"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	"github.com/pocket-id/pocket-id/backend/internal/service"
	httpapi "github.com/pocket-id/pocket-id/backend/internal/utils/huma"
)

// NewAuditLogController registers audit log routes
func NewAuditLogController(api huma.API, auditLogService *service.AuditLogService, authMiddleware *middleware.AuthMiddleware) {
	controller := &AuditLogController{auditLogService: auditLogService}

	allOperation := huma.Operation{OperationID: "list-all-audit-logs", Method: http.MethodGet, Path: "/api/audit-logs/all", Summary: "List all audit logs", Tags: []string{"Audit Logs"}}
	authMiddleware.Huma(api)(&allOperation)
	httpapi.Register(api, allOperation, controller.listAllAuditLogsHandler)

	userOperation := huma.Operation{OperationID: "list-current-user-audit-logs", Method: http.MethodGet, Path: "/api/audit-logs", Summary: "List audit logs for the current user", Tags: []string{"Audit Logs"}}
	authMiddleware.WithAdminNotRequired().Huma(api)(&userOperation)
	httpapi.Register(api, userOperation, controller.listAuditLogsForUserHandler)

	clientsOperation := huma.Operation{OperationID: "list-audit-log-client-names", Method: http.MethodGet, Path: "/api/audit-logs/filters/client-names", Summary: "List client names", Tags: []string{"Audit Logs"}}
	authMiddleware.Huma(api)(&clientsOperation)
	httpapi.Register(api, clientsOperation, controller.listClientNamesHandler)

	usersOperation := huma.Operation{OperationID: "list-audit-log-users", Method: http.MethodGet, Path: "/api/audit-logs/filters/users", Summary: "List users with IDs", Tags: []string{"Audit Logs"}}
	authMiddleware.Huma(api)(&usersOperation)
	httpapi.Register(api, usersOperation, controller.listUserNamesWithIDsHandler)
}

type AuditLogController struct {
	auditLogService *service.AuditLogService
}

func (alc *AuditLogController) listAuditLogsForUserHandler(ctx context.Context, input *httpapi.ListInput) (*httpapi.BodyOutput[dto.Paginated[dto.AuditLogDto]], error) {
	logs, pagination, err := alc.auditLogService.ListAuditLogsForUser(ctx, httpapi.UserID(ctx), input.ListRequestOptions)
	if err != nil {
		return nil, err
	}

	logsDTOs, err := alc.mapAuditLogs(logs, false)
	if err != nil {
		return nil, err
	}
	return &httpapi.BodyOutput[dto.Paginated[dto.AuditLogDto]]{Body: dto.Paginated[dto.AuditLogDto]{Data: logsDTOs, Pagination: pagination}}, nil
}

func (alc *AuditLogController) listAllAuditLogsHandler(ctx context.Context, input *httpapi.ListInput) (*httpapi.BodyOutput[dto.Paginated[dto.AuditLogDto]], error) {
	logs, pagination, err := alc.auditLogService.ListAllAuditLogs(ctx, input.ListRequestOptions)
	if err != nil {
		return nil, err
	}

	logsDTOs, err := alc.mapAuditLogs(logs, true)
	if err != nil {
		return nil, err
	}
	return &httpapi.BodyOutput[dto.Paginated[dto.AuditLogDto]]{Body: dto.Paginated[dto.AuditLogDto]{Data: logsDTOs, Pagination: pagination}}, nil
}

func (alc *AuditLogController) mapAuditLogs(logs []model.AuditLog, includeUsername bool) ([]dto.AuditLogDto, error) {
	var logsDTOs []dto.AuditLogDto
	if err := dto.MapStructList(logs, &logsDTOs); err != nil {
		return nil, err
	}
	for i := range logsDTOs {
		logsDTOs[i].Device = alc.auditLogService.DeviceStringFromUserAgent(logs[i].UserAgent)
		logsDTOs[i].ActorUsername = logsDTOs[i].Data["actorUsername"]
		if includeUsername {
			logsDTOs[i].Username = logs[i].User.Username
		}
	}
	return logsDTOs, nil
}

func (alc *AuditLogController) listClientNamesHandler(ctx context.Context, _ *httpapi.EmptyInput) (*httpapi.BodyOutput[[]string], error) {
	names, err := alc.auditLogService.ListClientNames(ctx)
	if err != nil {
		return nil, err
	}
	return &httpapi.BodyOutput[[]string]{Body: names}, nil
}

func (alc *AuditLogController) listUserNamesWithIDsHandler(ctx context.Context, _ *httpapi.EmptyInput) (*httpapi.BodyOutput[map[string]string], error) {
	users, err := alc.auditLogService.ListUsernamesWithIds(ctx)
	if err != nil {
		return nil, err
	}
	return &httpapi.BodyOutput[map[string]string]{Body: users}, nil
}
