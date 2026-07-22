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
	httpapi.Register(api, huma.Operation{
		OperationID: "list-apis",
		Method:      http.MethodGet,
		Path:        "/api/apis",
		Summary:     "List APIs",
		Tags:        []string{"APIs"},
	}, m.handler.list, adminAuth)

	httpapi.Register(api, huma.Operation{
		OperationID:   "create-api",
		Method:        http.MethodPost,
		Path:          "/api/apis",
		Summary:       "Create API",
		Tags:          []string{"APIs"},
		DefaultStatus: http.StatusCreated,
	}, m.handler.create, adminAuth)

	httpapi.Register(api, huma.Operation{
		OperationID: "get-api",
		Method:      http.MethodGet,
		Path:        "/api/apis/{id}",
		Summary:     "Get API by ID",
		Tags:        []string{"APIs"},
	}, m.handler.get, adminAuth)

	httpapi.Register(api, huma.Operation{
		OperationID: "update-api",
		Method:      http.MethodPut,
		Path:        "/api/apis/{id}",
		Summary:     "Update API",
		Tags:        []string{"APIs"},
	}, m.handler.update, adminAuth)

	httpapi.Register(api, huma.Operation{
		OperationID:   "delete-api",
		Method:        http.MethodDelete,
		Path:          "/api/apis/{id}",
		Summary:       "Delete API",
		Tags:          []string{"APIs"},
		DefaultStatus: http.StatusNoContent,
	}, m.handler.delete, adminAuth)

	httpapi.Register(api, huma.Operation{
		OperationID: "update-api-permissions",
		Method:      http.MethodPut,
		Path:        "/api/apis/{id}/permissions",
		Summary:     "Update API permissions",
		Tags:        []string{"APIs"},
	}, m.handler.updatePermissions, adminAuth)

	httpapi.Register(api, huma.Operation{
		OperationID: "get-client-api-access",
		Method:      http.MethodGet,
		Path:        "/api/api-access/{clientId}",
		Summary:     "Get client API access",
		Tags:        []string{"APIs"},
	}, m.handler.getClientAccess, adminAuth)

	httpapi.Register(api, huma.Operation{
		OperationID: "update-client-api-access",
		Method:      http.MethodPut,
		Path:        "/api/api-access/{clientId}",
		Summary:     "Update client API access",
		Tags:        []string{"APIs"},
	}, m.handler.updateClientAccess, adminAuth)
}
