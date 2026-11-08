package usergroups

import (
	"context"

	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/appconfig"
	"github.com/pocket-id/pocket-id/backend/internal/authz"
	"github.com/pocket-id/pocket-id/backend/internal/backchannellogout"
	"github.com/pocket-id/pocket-id/backend/internal/dto"
	"github.com/pocket-id/pocket-id/backend/internal/httpserver"
	"github.com/pocket-id/pocket-id/backend/internal/model"
)

// ScimSyncScheduler schedules SCIM after a group change has committed
type ScimSyncScheduler interface {
	ScheduleSync(ctx context.Context)
}

type Dependencies struct {
	DB                *gorm.DB
	AppConfig         appconfig.AppConfigResolver
	ScimSync          ScimSyncScheduler
	BackchannelLogout *backchannellogout.Service
}

type Module struct {
	service *Service
	handler *handler
}

func New(deps Dependencies) *Module {
	service := newService(deps.DB, deps.ScimSync, deps.BackchannelLogout)
	return &Module{
		service: service,
		handler: newHandler(deps.AppConfig, service),
	}
}

func (m *Module) RegisterRoutes(r *authz.Router) {
	group := r.Group("/user-groups")
	group.GET("", authz.GroupsRead, httpserver.Handle(m.handler.list))
	group.GET("/:id", authz.GroupsRead, httpserver.Handle(m.handler.get))
	group.POST("", authz.GroupsWrite, httpserver.Handle(m.handler.create))
	group.PUT("/:id", authz.GroupsWrite, httpserver.Handle(m.handler.update))
	group.DELETE("/:id", authz.GroupsWrite, httpserver.Handle(m.handler.delete))
	group.PUT("/:id/users", authz.GroupsWrite, httpserver.Handle(m.handler.updateUsers))
	group.PUT("/:id/allowed-oidc-clients", authz.GroupsWrite, httpserver.Handle(m.handler.updateAllowedOidcClients))
}

// CreateInternal creates a user group within an existing transaction
// It's exported for the LDAP sync, which reconciles users and groups in a single transaction of its own
func (m *Module) CreateInternal(ctx context.Context, input dto.UserGroupCreateDto, tx *gorm.DB) (model.UserGroup, error) {
	return m.service.createInternal(ctx, input, tx)
}

// UpdateInternal updates a user group within an existing transaction
// It's exported for the LDAP sync, which reconciles users and groups in a single transaction of its own
func (m *Module) UpdateInternal(ctx context.Context, cfg *appconfig.AppConfigModel, id string, input dto.UserGroupCreateDto, isLdapSync bool, tx *gorm.DB) (model.UserGroup, error) {
	return m.service.updateInternal(ctx, id, input, isLdapSync, tx, cfg)
}

// UpdateUsersInternal replaces the members of a user group within an existing transaction
// It's exported for the LDAP sync, which reconciles users and groups in a single transaction of its own
func (m *Module) UpdateUsersInternal(ctx context.Context, id string, userIDs []string, tx *gorm.DB) (model.UserGroup, error) {
	return m.service.updateUsersInternal(ctx, id, userIDs, tx)
}
