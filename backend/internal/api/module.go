package api

import (
	"context"
	"net/http"

	"gorm.io/gorm"

	"github.com/go-fuego/fuego/option"
	"github.com/pocket-id/pocket-id/backend/internal/authz"
	"github.com/pocket-id/pocket-id/backend/internal/dto"
	"github.com/pocket-id/pocket-id/backend/internal/httpserver"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	"github.com/pocket-id/pocket-id/backend/internal/oidc"
	"github.com/pocket-id/pocket-id/backend/internal/openapi"
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

// HeldScopes implements the OIDC module's APIAccessProvider interface
func (m *Module) HeldScopes(ctx context.Context, tx *gorm.DB, userID, audience string, scopes []string) ([]string, error) {
	return m.service.HeldScopes(ctx, tx, userID, audience, scopes)
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
		infos[i] = dto.ScopeInfoDto{Key: permission.Key, Name: permission.Name, Description: description, BuiltIn: permission.APIID == BuiltInAPIID}
	}

	return infos, nil
}

// RegisterRoutes mounts the admin CRUD endpoints
func (m *Module) RegisterRoutes(r *authz.Router) {
	apis := r.Group("/admin/apis")
	apis.GET("", authz.APIsRead, httpserver.JSON(http.StatusOK, m.handler.list),
		option.Summary("List APIs"),
		option.Description("Get a paginated list of APIs with optional search and sorting"),
		option.Tags("APIs"),
		option.Query("search", "Search term to filter APIs by name or resource"),
		openapi.List[API]("resource"),
	)
	apis.POST("", authz.APIsWrite, httpserver.BodyJSON(http.StatusCreated, m.handler.create),
		option.Summary("Create API"),
		option.Description("Create a new API resource server"),
		option.Tags("APIs"),
	)
	apis.GET("/:id", authz.APIsRead, httpserver.JSON(http.StatusOK, m.handler.get),
		option.Summary("Get API by ID"),
		option.Description("Retrieve a single API including its permissions"),
		option.Tags("APIs"),
		option.Path("id", "API ID"),
	)
	updateDocs := option.Group(
		option.Summary("Update API"),
		option.Description("Update an existing API by ID, where PATCH changes only the fields in the body"),
		option.Tags("APIs"),
		option.Path("id", "API ID"),
	)
	apis.PUT("/:id", authz.APIsWrite, httpserver.UpdateJSON(m.handler.update), updateDocs)
	apis.PATCH("/:id", authz.APIsWrite, httpserver.UpdateJSON(m.handler.update), updateDocs)
	apis.DELETE("/:id", authz.APIsWrite, httpserver.NoContent(m.handler.delete),
		option.Summary("Delete API"),
		option.Description("Delete an API by ID"),
		option.Tags("APIs"),
		option.Path("id", "API ID"),
	)
	apis.PUT("/:id/permissions", authz.APIsWrite, httpserver.BodyJSON(http.StatusOK, m.handler.updatePermissions),
		option.Summary("Update API permissions"),
		option.Description("Replace the full set of permissions for an API"),
		option.Tags("APIs"),
		option.Path("id", "API ID"),
	)
	apis.PUT("/:id/cimd-access", authz.APIsWrite, httpserver.BodyJSON(http.StatusOK, m.handler.updateCimdAccess),
		option.Summary("Update metadata document client access"),
		option.Description("Replace which permissions of an API every client registered through a Client ID Metadata Document may request"),
		option.Tags("APIs"),
		option.Path("id", "API ID"),
	)

	// The same client grants are editable from either side of the relation, so the API can list and manage its clients too
	apis.GET("/:id/clients", authz.APIsRead, httpserver.JSON(http.StatusOK, m.handler.listClients),
		option.Summary("List clients with access to an API"),
		option.Description("Get a paginated list of OIDC clients that may reach the API, with their permissions split into user-delegated and client (machine-to-machine) access"),
		option.Tags("APIs"),
		option.Path("id", "API ID"),
		option.Query("search", "Search term to filter clients by name"),
		openapi.List[model.OidcClient](),
	)
	apis.GET("/:id/assignable-clients", authz.APIsRead, httpserver.JSON(http.StatusOK, m.handler.listAssignableClients),
		option.Summary("List clients that can still be granted access to an API"),
		option.Description("Get a paginated list of OIDC clients that have no grant on the API yet"),
		option.Tags("APIs"),
		option.Path("id", "API ID"),
		option.Query("search", "Search term to filter clients by name"),
		openapi.List[model.OidcClient](),
	)
	apis.PUT("/:id/clients/:clientId", authz.APIsWrite, httpserver.BodyJSON(http.StatusOK, m.handler.updateClientAccessForApi),
		option.Summary("Update a client's access to an API"),
		option.Description("Replace the permissions of this API a single OIDC client may request, leaving its grants on other APIs untouched"),
		option.Tags("APIs"),
		option.Path("id", "API ID"),
		option.Path("clientId", "OIDC Client ID"),
	)
	apis.DELETE("/:id/clients/:clientId", authz.APIsWrite, httpserver.NoContent(m.handler.removeClientAccessForApi),
		option.Summary("Revoke a client's access to an API"),
		option.Description("Remove every permission of this API a single OIDC client was allowed to request"),
		option.Tags("APIs"),
		option.Path("id", "API ID"),
		option.Path("clientId", "OIDC Client ID"),
	)

	access := r.Group("/admin/api-access")
	access.GET("/:clientId/apis", authz.APIsRead, httpserver.JSON(http.StatusOK, m.handler.listClientApis),
		option.Summary("List APIs a client may access"),
		option.Description("Get every API the OIDC client may request tokens for, with its access and permissions split into user-delegated and client (machine-to-machine) access"),
		option.Tags("APIs"),
		option.Path("clientId", "OIDC Client ID"),
	)
	access.GET("/:clientId/assignable-apis", authz.APIsRead, httpserver.JSON(http.StatusOK, m.handler.listAssignableApis),
		option.Summary("List APIs a client can still be granted access to"),
		option.Description("Get a paginated list of APIs the OIDC client cannot already reach"),
		option.Tags("APIs"),
		option.Path("clientId", "OIDC Client ID"),
		option.Query("search", "Search term to filter APIs by name or resource"),
		openapi.List[API]("resource"),
	)
}
