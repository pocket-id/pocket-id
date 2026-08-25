package ldapsync

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/italypaleale/francis/host/local"
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/appconfig"
	"github.com/pocket-id/pocket-id/backend/internal/dto"
	"github.com/pocket-id/pocket-id/backend/internal/httpserver"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	"github.com/pocket-id/pocket-id/backend/internal/storage"
)

// UserSyncer applies the desired LDAP state to the users in the database
// Every method takes the transaction the sync runs in, since users, groups, and memberships are reconciled atomically
type UserSyncer interface {
	CreateUserInternal(ctx context.Context, dbConfig *appconfig.AppConfigModel, input dto.UserCreateDto, isLdapSync bool, tx *gorm.DB) (model.User, error)
	UpdateUserInternal(ctx context.Context, dbConfig *appconfig.AppConfigModel, userID string, input dto.UserCreateDto, updateOwnUser bool, isLdapSync bool, tx *gorm.DB) (model.User, error)
	DisableUserInternal(ctx context.Context, tx *gorm.DB, userID string) error
	DeleteUserInternal(ctx context.Context, dbConfig *appconfig.AppConfigModel, tx *gorm.DB, userID string, allowLdapDelete bool) error

	// UpdateProfilePicture stores a user's profile picture, which happens after the transaction has been committed since it touches the storage layer
	UpdateProfilePicture(ctx context.Context, userID string, file io.ReadSeeker) error
}

// GroupSyncer applies the desired LDAP state to the user groups in the database
type GroupSyncer interface {
	CreateInternal(ctx context.Context, input dto.UserGroupCreateDto, tx *gorm.DB) (model.UserGroup, error)
	UpdateInternal(ctx context.Context, dbConfig *appconfig.AppConfigModel, id string, input dto.UserGroupCreateDto, isLdapSync bool, tx *gorm.DB) (model.UserGroup, error)
	UpdateUsersInternal(ctx context.Context, id string, userIDs []string, tx *gorm.DB) (model.UserGroup, error)
}

// ScimSyncScheduler schedules SCIM after the LDAP transaction has committed
type ScimSyncScheduler interface {
	ScheduleSync(ctx context.Context)
}

type Dependencies struct {
	DB          *gorm.DB
	Actors      *local.Host
	HTTPClient  *http.Client
	FileStorage storage.FileStorage

	Users     UserSyncer
	Groups    GroupSyncer
	AppConfig appconfig.AppConfigResolver
	ScimSync  ScimSyncScheduler

	// ScheduleDisabled keeps the recurring sync from being armed
	// It's set in the test environment, where syncs are driven explicitly by the end-to-end tests
	ScheduleDisabled bool
}

type Module struct {
	service *Service
	handler *handler
}

func New(deps Dependencies) (*Module, error) {
	service := newService(deps)

	// Register the actor that drives the recurring sync
	// It's a singleton, so the host bootstraps it at startup and the alarm fires once per cluster rather than once per replica
	err := deps.Actors.RegisterSingletonActor(SyncActorType, NewSyncActor(service, deps.AppConfig, deps.ScheduleDisabled))
	if err != nil {
		return nil, fmt.Errorf("error registering the %s actor: %w", SyncActorType, err)
	}

	return &Module{
		service: service,
		handler: newHandler(service, deps.AppConfig),
	}, nil
}

// RegisterRoutes mounts the manual LDAP synchronization endpoint
// auth guards it, as it's an admin-only operation
func (m *Module) RegisterRoutes(apiGroup *gin.RouterGroup, auth gin.HandlerFunc) {
	apiGroup.POST("/application-configuration/sync-ldap", auth, httpserver.Handle(m.handler.syncLdap))
}

// SyncAll runs a full LDAP synchronization with the provided application configuration
func (m *Module) SyncAll(ctx context.Context, dbConfig *appconfig.AppConfigModel) error {
	return m.service.SyncAll(ctx, dbConfig)
}
