package api

import (
	"context"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

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
func (m *Module) ClientAPIScopes(ctx context.Context, tx *gorm.DB, clientID string) (scopes []string, audiences []string, err error) {
	return m.service.ClientAPIScopesAndAudiences(ctx, tx, clientID)
}

// AllowedScopesForAudience implements the OIDC module's APIAccessProvider interface
func (m *Module) AllowedScopesForAudience(ctx context.Context, tx *gorm.DB, clientID, audience string, subjectType oidc.SubjectType) (scopes []string, apiExists bool, err error) {
	return m.service.AllowedScopesForAudience(ctx, tx, clientID, audience, subjectType)
}

// DescribePermissions implements the OIDC module's APIAccessProvider interface
func (m *Module) DescribePermissions(ctx context.Context, audience string, keys []string) ([]oidc.PermissionInfo, error) {
	permissions, err := m.service.DescribePermissions(ctx, audience, keys)
	if err != nil {
		return nil, err
	}

	infos := make([]oidc.PermissionInfo, len(permissions))
	for i, permission := range permissions {
		description := ""
		if permission.Description != nil {
			description = *permission.Description
		}
		infos[i] = oidc.PermissionInfo{Key: permission.Key, Name: permission.Name, Description: description}
	}

	return infos, nil
}

// RegisterRoutes mounts the admin CRUD endpoints
// adminAuth is passed in as a gin handler so the module does not import internal/middleware
func (m *Module) RegisterRoutes(apiGroup *gin.RouterGroup, adminAuth gin.HandlerFunc) {
	apis := apiGroup.Group("/apis")
	apis.Use(adminAuth)
	apis.GET("", m.handler.list)
	apis.POST("", m.handler.create)
	apis.GET("/:id", m.handler.get)
	apis.PUT("/:id", m.handler.update)
	apis.DELETE("/:id", m.handler.delete)
	apis.PUT("/:id/permissions", m.handler.updatePermissions)

	// The per-client API-access allow-list lives on a separate path so it does not collide with the /apis/:id wildcard
	access := apiGroup.Group("/api-access")
	access.Use(adminAuth)
	access.GET("/:clientId", m.handler.getClientAccess)
	access.PUT("/:clientId", m.handler.updateClientAccess)
}
