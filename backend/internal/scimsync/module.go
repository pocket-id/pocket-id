package scimsync

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/go-fuego/fuego/option"
	"github.com/italypaleale/francis/actor"
	francishost "github.com/italypaleale/francis/host"
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/authz"
	"github.com/pocket-id/pocket-id/backend/internal/httpserver"
	"github.com/pocket-id/pocket-id/backend/internal/openapi"
)

type Dependencies struct {
	DB         *gorm.DB
	Actors     francishost.Host
	HTTPClient *http.Client

	// ScheduleDisabled keeps automatic synchronizations from being armed
	// It's set in the test environment, where SCIM syncs are driven explicitly by the end-to-end tests
	ScheduleDisabled bool
}

type Module struct {
	service          *Service
	handler          *handler
	actors           *actor.Service
	scheduleDisabled bool
}

func New(deps Dependencies) (*Module, error) {
	service := newService(deps.DB, deps.HTTPClient)

	// Register the singleton so recurring and debounced synchronizations run once for the entire cluster
	err := deps.Actors.RegisterSingletonActor(ActorType, NewActor(service, deps.ScheduleDisabled))
	if err != nil {
		return nil, fmt.Errorf("error registering the %s actor: %w", ActorType, err)
	}

	return &Module{
		service:          service,
		handler:          newHandler(service),
		actors:           deps.Actors.Service(),
		scheduleDisabled: deps.ScheduleDisabled,
	}, nil
}

// RegisterRoutes mounts the SCIM service provider endpoints, which belong to an OIDC client
func (m *Module) RegisterRoutes(r *authz.Router) {
	r.GET("/oidc/clients/:id/scim-service-provider", authz.OidcClientsRead, httpserver.JSON(http.StatusOK, m.handler.getServiceProviderByClient),
		option.Summary("Get SCIM service provider"),
		option.Description("Get the SCIM service provider configuration for an OIDC client"),
		option.Tags("OIDC"),
		option.Path("id", "Client ID"),
	)
	r.POST("/scim/service-provider", authz.OidcClientsWrite, httpserver.BodyJSON(http.StatusCreated, m.handler.createServiceProvider),
		option.Summary("Create SCIM service provider"),
		option.Description("Create a new SCIM service provider"),
		option.Tags("SCIM"),
	)
	r.POST("/scim/service-provider/:id/sync", authz.OidcClientsWrite, httpserver.Handle(m.handler.syncServiceProvider),
		option.Summary("Sync SCIM service provider"),
		option.Description("Trigger synchronization for a SCIM service provider"),
		option.Tags("SCIM"),
		option.Path("id", "Service Provider ID"),
		openapi.Status(http.StatusOK, ""),
	)
	updateServiceProviderDocs := option.Group(
		option.Summary("Update SCIM service provider"),
		option.Description("Update an existing SCIM service provider, where PATCH changes only the fields in the body, an omitted token keeps the stored one, an empty token removes it and a changed endpoint requires the token"),
		option.Tags("SCIM"),
		option.Path("id", "Service Provider ID"),
	)
	r.PUT("/scim/service-provider/:id", authz.OidcClientsWrite, httpserver.UpdateJSON(m.handler.updateServiceProvider), updateServiceProviderDocs)
	r.PATCH("/scim/service-provider/:id", authz.OidcClientsWrite, httpserver.UpdateJSON(m.handler.updateServiceProvider), updateServiceProviderDocs)
	r.DELETE("/scim/service-provider/:id", authz.OidcClientsWrite, httpserver.NoContent(m.handler.deleteServiceProvider),
		option.Summary("Delete SCIM service provider"),
		option.Description("Delete a SCIM service provider by ID"),
		option.Tags("SCIM"),
		option.Path("id", "Service Provider ID"),
	)
}

// ScheduleSync schedules a debounced cluster-wide synchronization after SCIM-relevant data changes
func (m *Module) ScheduleSync(ctx context.Context) {
	if m.scheduleDisabled {
		return
	}

	_, err := m.actors.Invoke(ctx, ActorType, actor.SingletonActorID, methodScheduleSync, nil)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to schedule SCIM sync", slog.Any("error", err))
	}
}
