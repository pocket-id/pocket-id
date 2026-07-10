package api

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/dto"
	"github.com/pocket-id/pocket-id/backend/internal/oidc"
	httpapi "github.com/pocket-id/pocket-id/backend/internal/utils/huma"
)

type Dependencies struct {
	DB *gorm.DB
	// Issuer is the OpenID Provider issuer URL, reserved so a custom API cannot claim it as its audience
	Issuer string
}

type Module struct {
	service *Service
	handler *handler
}

func New(deps Dependencies) *Module {
	service := newService(deps.DB, deps.Issuer)
	return &Module{
		service: service,
		handler: newHandler(service),
	}
}

// ClientAPIScopes implements the OIDC module's APIAccessProvider interface
func (m *Module) ClientAPIScopes(ctx context.Context, tx *gorm.DB, clientID string) (scopes []string, audiences []string, err error) {
	return m.service.ClientAPIScopesAndAudiences(ctx, tx, clientID)
}

// AllowedScopesForAudience implements the OIDC module's APIAccessProvider interface
func (m *Module) AllowedScopesForAudience(ctx context.Context, tx *gorm.DB, clientID, audience string, subjectType oidc.SubjectType) (scopes []string, apiExists bool, err error) {
	return m.service.AllowedScopesForAudience(ctx, tx, clientID, audience, subjectType)
}

// DescribePermissions implements the OIDC module's APIAccessProvider interface
func (m *Module) DescribePermissions(ctx context.Context, audience string, keys []string) ([]dto.ScopeInfoDto, error) {
	permissions, err := m.service.DescribePermissions(ctx, audience, keys)
	if err != nil {
		return nil, err
	}

	infos := make([]dto.ScopeInfoDto, len(permissions))
	for i, permission := range permissions {
		description := ""
		if permission.Description != nil {
			description = *permission.Description
		}
		infos[i] = dto.ScopeInfoDto{Key: permission.Key, Name: permission.Name, Description: description}
	}

	return infos, nil
}

// RegisterRoutes mounts the admin CRUD endpoints
func (m *Module) RegisterRoutes(api huma.API, adminAuth func(*huma.Operation)) {
	register := func(operation huma.Operation, registerHandler func(huma.Operation)) {
		adminAuth(&operation)
		registerHandler(operation)
	}

	register(apiOperation("list-apis", http.MethodGet, "/api/apis", "List APIs"), func(operation huma.Operation) { httpapi.Register(api, operation, m.handler.list) })
	createOperation := apiOperation("create-api", http.MethodPost, "/api/apis", "Create API")
	createOperation.DefaultStatus = http.StatusCreated
	register(createOperation, func(operation huma.Operation) { httpapi.Register(api, operation, m.handler.create) })
	register(apiOperation("get-api", http.MethodGet, "/api/apis/{id}", "Get API by ID"), func(operation huma.Operation) { httpapi.Register(api, operation, m.handler.get) })
	register(apiOperation("update-api", http.MethodPut, "/api/apis/{id}", "Update API"), func(operation huma.Operation) { httpapi.Register(api, operation, m.handler.update) })
	deleteOperation := apiOperation("delete-api", http.MethodDelete, "/api/apis/{id}", "Delete API")
	deleteOperation.DefaultStatus = http.StatusNoContent
	register(deleteOperation, func(operation huma.Operation) { httpapi.Register(api, operation, m.handler.delete) })
	register(apiOperation("update-api-permissions", http.MethodPut, "/api/apis/{id}/permissions", "Update API permissions"), func(operation huma.Operation) { httpapi.Register(api, operation, m.handler.updatePermissions) })
	register(apiOperation("get-client-api-access", http.MethodGet, "/api/api-access/{clientId}", "Get client API access"), func(operation huma.Operation) { httpapi.Register(api, operation, m.handler.getClientAccess) })
	register(apiOperation("update-client-api-access", http.MethodPut, "/api/api-access/{clientId}", "Update client API access"), func(operation huma.Operation) { httpapi.Register(api, operation, m.handler.updateClientAccess) })
}

func apiOperation(id, method, path, summary string) huma.Operation {
	return huma.Operation{OperationID: id, Method: method, Path: path, Summary: summary, Tags: []string{"APIs"}}
}
