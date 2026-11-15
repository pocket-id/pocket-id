package usergroups

import (
	"context"
	"net/http"

	"gorm.io/gorm"

	"github.com/go-fuego/fuego/option"
	"github.com/pocket-id/pocket-id/backend/internal/appconfig"
	"github.com/pocket-id/pocket-id/backend/internal/auditlogs"
	"github.com/pocket-id/pocket-id/backend/internal/authz"
	"github.com/pocket-id/pocket-id/backend/internal/backchannellogout"
	"github.com/pocket-id/pocket-id/backend/internal/dto"
	"github.com/pocket-id/pocket-id/backend/internal/httpserver"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	"github.com/pocket-id/pocket-id/backend/internal/openapi"
)

type AuditLogger interface {
	Record(ctx context.Context, event auditlogs.Event, origin auditlogs.Origin, userID string, data auditlogs.Data, tx *gorm.DB)
}

// ScimSyncScheduler schedules SCIM after a group change has committed
type ScimSyncScheduler interface {
	ScheduleSync(ctx context.Context)
}

type Dependencies struct {
	DB                *gorm.DB
	AppConfig         appconfig.AppConfigResolver
	ScimSync          ScimSyncScheduler
	BackchannelLogout *backchannellogout.Service
	AuditLog          AuditLogger
}

type Module struct {
	service *Service
	handler *handler
}

func New(deps Dependencies) *Module {
	service := newService(deps.DB, deps.ScimSync, deps.BackchannelLogout, deps.AuditLog)
	return &Module{
		service: service,
		handler: newHandler(deps.AppConfig, service),
	}
}

func (m *Module) RegisterRoutes(r *authz.Router) {
	group := r.Group("/admin/user-groups")
	group.GET("", authz.GroupsRead, httpserver.JSON(http.StatusOK, m.handler.list),
		option.Summary("List user groups"),
		option.Description("Get a paginated list of user groups with optional search and sorting"),
		option.Tags("User Groups"),
		option.Query("search", "Search term to filter user groups by name"),
		openapi.List[model.UserGroup]("userCount"),
	)
	group.GET("/:id", authz.GroupsRead, httpserver.JSON(http.StatusOK, m.handler.get),
		option.Summary("Get user group by ID"),
		option.Description("Retrieve detailed information about a specific user group including its users"),
		option.Tags("User Groups"),
		option.Path("id", "User Group ID"),
	)
	group.POST("", authz.GroupsWrite, httpserver.BodyJSON(http.StatusCreated, m.handler.create),
		option.Summary("Create user group"),
		option.Description("Create a new user group"),
		option.Tags("User Groups"),
	)
	updateDocs := option.Group(
		option.Summary("Update user group"),
		option.Description("Update an existing user group by ID, where PATCH changes only the fields in the body"),
		option.Tags("User Groups"),
		option.Path("id", "User Group ID"),
	)
	group.PUT("/:id", authz.GroupsWrite, httpserver.UpdateJSON(m.handler.update), updateDocs)
	group.PATCH("/:id", authz.GroupsWrite, httpserver.UpdateJSON(m.handler.update), updateDocs)
	group.DELETE("/:id", authz.GroupsWrite, httpserver.NoContent(m.handler.delete),
		option.Summary("Delete user group"),
		option.Description("Delete a specific user group by ID"),
		option.Tags("User Groups"),
		option.Path("id", "User Group ID"),
	)
	group.PUT("/:id/users", authz.GroupsWrite, httpserver.BodyJSON(http.StatusOK, m.handler.updateUsers),
		option.Summary("Update users in a group"),
		option.Description("Update the list of users belonging to a specific user group"),
		option.Tags("User Groups"),
		option.Path("id", "User Group ID"),
	)
	group.PUT("/:id/allowed-oidc-clients", authz.GroupsWrite, httpserver.BodyJSON(http.StatusOK, m.handler.updateAllowedOidcClients),
		option.Summary("Update allowed OIDC clients"),
		option.Description("Update the OIDC clients allowed for a specific user group"),
		option.Tags("OIDC"),
		option.Path("id", "User Group ID"),
	)
}

// CreateInternal creates a user group within an existing transaction
// It's exported for the LDAP sync, which reconciles users and groups in a single transaction of its own
func (m *Module) CreateInternal(ctx context.Context, input dto.UserGroupCreateDto, tx *gorm.DB) (model.UserGroup, error) {
	return m.service.createInternal(ctx, input, auditlogs.LDAPSync, tx)
}

// UpdateInternal updates a user group within an existing transaction
// It's exported for the LDAP sync, which reconciles users and groups in a single transaction of its own
func (m *Module) UpdateInternal(ctx context.Context, cfg *appconfig.AppConfigModel, id string, input dto.UserGroupCreateDto, isLdapSync bool, tx *gorm.DB) (model.UserGroup, error) {
	return m.service.updateInternal(ctx, id, input, isLdapSync, tx, cfg)
}

// UpdateUsersInternal replaces the members of a user group within an existing transaction
// It's exported for the LDAP sync, which reconciles users and groups in a single transaction of its own
func (m *Module) UpdateUsersInternal(ctx context.Context, id string, userIDs []string, tx *gorm.DB) (model.UserGroup, error) {
	return m.service.updateUsersInternal(ctx, id, userIDs, auditlogs.LDAPSync, tx)
}

// DeleteInternal deletes a user group within an existing transaction
// It's exported for the LDAP sync, which deletes groups that are no longer in the directory
func (m *Module) DeleteInternal(ctx context.Context, group model.UserGroup, tx *gorm.DB) error {
	return m.service.deleteInternal(ctx, group, auditlogs.LDAPSync, tx)
}
