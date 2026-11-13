package apikey

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/go-fuego/fuego/option"
	francishost "github.com/italypaleale/francis/host"
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/appconfig"
	"github.com/pocket-id/pocket-id/backend/internal/auditlogs"
	"github.com/pocket-id/pocket-id/backend/internal/authz"
	"github.com/pocket-id/pocket-id/backend/internal/httpserver"
	"github.com/pocket-id/pocket-id/backend/internal/openapi"
)

type AuditLogger interface {
	Record(ctx context.Context, event auditlogs.Event, origin auditlogs.Origin, userID string, data auditlogs.Data, tx *gorm.DB)
}

type Dependencies struct {
	DB              *gorm.DB
	Actors          francishost.Host
	StaticApiKey    string
	AppConfig       appconfig.AppConfigResolver
	EmailSender     APIKeyExpiryEmailSender
	AuditLog        AuditLogger
	CleanupDisabled bool
}

type Module struct {
	service *Service
	handler *handler
}

func New(deps Dependencies) (*Module, error) {
	service := newService(deps.DB, deps.StaticApiKey, deps.AuditLog)
	module := &Module{
		service: service,
		handler: newHandler(service),
	}

	// Register the cleanup job for expired API keys
	if !deps.CleanupDisabled {
		if deps.Actors == nil {
			return nil, errors.New("actor host is required for the API key expiration cron job")
		}
		if deps.AppConfig == nil || deps.EmailSender == nil {
			return nil, errors.New("notification dependencies are required for the API key expiration cron job")
		}

		expiryJob, err := newExpiryJob(service, deps.AppConfig, deps.EmailSender)
		if err != nil {
			return nil, err
		}

		err = deps.Actors.RegisterBuiltInActor(expiryJob)
		if err != nil {
			return nil, fmt.Errorf("error registering API key expiration cron actor: %w", err)
		}
	}

	return module, nil
}

// RegisterRoutes mounts the API key management endpoints
// Creating, renewing and editing keys requires a browser session so an API key cannot mint further API keys or widen their scopes
func (m *Module) RegisterRoutes(r *authz.Router) {
	group := r.Group("/api-keys")
	group.GET("", authz.AccountAPIKeys, httpserver.JSON(http.StatusOK, m.handler.list),
		option.Summary("List API keys"),
		option.Description("Get a paginated list of API keys belonging to the current user"),
		option.Tags("API Keys"),
		openapi.List[ApiKey](),
	)
	group.POST("", authz.AccountAPIKeysCreate, httpserver.BodyJSON(http.StatusCreated, m.handler.create),
		option.Summary("Create API key"),
		option.Description("Create a new API key for the current user"),
		option.Tags("API Keys"),
	)
	updateDocs := option.Group(
		option.Summary("Update API key"),
		option.Description("Change the name, description and scopes of an existing API key, where PATCH changes only the fields in the body; its token stays the same"),
		option.Tags("API Keys"),
		option.Path("id", "API Key ID"),
	)
	group.PUT("/:id", authz.AccountAPIKeysCreate, httpserver.UpdateJSON(m.handler.update), updateDocs)
	group.PATCH("/:id", authz.AccountAPIKeysCreate, httpserver.UpdateJSON(m.handler.update), updateDocs)
	group.POST("/:id/renew", authz.AccountAPIKeysCreate, httpserver.BodyJSON(http.StatusOK, m.handler.renew),
		option.Summary("Renew API key"),
		option.Description("While a key has not expired, its old token keeps working for the given grace period, but never past the key's previous expiration"),
		option.Tags("API Keys"),
		option.Path("id", "API Key ID"),
	)
	group.DELETE("/:id", authz.AccountAPIKeys, httpserver.NoContent(m.handler.revoke),
		option.Summary("Revoke API key"),
		option.Description("Revoke (delete) an existing API key by ID"),
		option.Tags("API Keys"),
		option.Path("id", "API Key ID"),
	)
}

// ValidateApiKey resolves the given raw API key to the stored key with its owner preloaded
// It is used by the authentication middleware
func (m *Module) ValidateApiKey(ctx context.Context, apiKey string) (ApiKey, error) {
	return m.service.ValidateApiKey(ctx, apiKey)
}
