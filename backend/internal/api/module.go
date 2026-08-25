package api

import (
	"context"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/dto"
	"github.com/pocket-id/pocket-id/backend/internal/httpserver"
	"github.com/pocket-id/pocket-id/backend/internal/oidc"
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
func (m *Module) ClientAPIScopes(ctx context.Context, tx *gorm.DB, clientID string, isCIMDClient bool) (scopes []string, audiences []string, err error) {
	return m.service.ClientAPIScopesAndAudiences(ctx, tx, clientID, isCIMDClient)
}

// AllowedScopesForAudience implements the OIDC module's APIAccessProvider interface
func (m *Module) AllowedScopesForAudience(ctx context.Context, tx *gorm.DB, clientID, audience string, subjectType oidc.SubjectType) (scopes []string, apiExists bool, hasAccess bool, err error) {
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
// adminAuth is passed in as a gin handler so the module does not import internal/middleware
func (m *Module) RegisterRoutes(apiGroup *gin.RouterGroup, adminAuth gin.HandlerFunc) {
	apis := apiGroup.Group("/apis")
	apis.Use(adminAuth)
	apis.GET("", httpserver.Handle(m.handler.list))
	apis.POST("", httpserver.Handle(m.handler.create))
	apis.GET("/:id", httpserver.Handle(m.handler.get))
	apis.PUT("/:id", httpserver.Handle(m.handler.update))
	apis.DELETE("/:id", httpserver.Handle(m.handler.delete))
	apis.PUT("/:id/permissions", httpserver.Handle(m.handler.updatePermissions))
	apis.PUT("/:id/cimd-access", httpserver.Handle(m.handler.updateCimdAccess))

	// The same client grants are editable from either side of the relation, so the API can list and manage its clients too
	apis.GET("/:id/clients", httpserver.Handle(m.handler.listClients))
	apis.GET("/:id/assignable-clients", httpserver.Handle(m.handler.listAssignableClients))
	apis.PUT("/:id/clients/:clientId", httpserver.Handle(m.handler.updateClientAccessForApi))
	apis.DELETE("/:id/clients/:clientId", httpserver.Handle(m.handler.removeClientAccessForApi))

	access := apiGroup.Group("/api-access")
	access.Use(adminAuth)
	access.GET("/:clientId/apis", httpserver.Handle(m.handler.listClientApis))
	access.GET("/:clientId/assignable-apis", httpserver.Handle(m.handler.listAssignableApis))
}
