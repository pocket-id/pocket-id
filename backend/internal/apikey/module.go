package apikey

import (
	"context"
	"errors"
	"fmt"

	francishost "github.com/italypaleale/francis/host"
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/appconfig"
	"github.com/pocket-id/pocket-id/backend/internal/auditlogs"
	"github.com/pocket-id/pocket-id/backend/internal/authz"
	"github.com/pocket-id/pocket-id/backend/internal/httpserver"
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

func New(ctx context.Context, deps Dependencies) (*Module, error) {
	service, err := newService(ctx, deps.DB, deps.StaticApiKey, deps.AuditLog)
	if err != nil {
		return nil, err
	}

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
	group.GET("", authz.AccountAPIKeys, httpserver.Handle(m.handler.list))
	group.POST("", authz.AccountAPIKeysCreate, httpserver.Handle(m.handler.create))
	group.PUT("/:id", authz.AccountAPIKeysCreate, httpserver.Handle(m.handler.update))
	group.POST("/:id/renew", authz.AccountAPIKeysCreate, httpserver.Handle(m.handler.renew))
	group.DELETE("/:id", authz.AccountAPIKeys, httpserver.Handle(m.handler.revoke))
}

// ValidateApiKey resolves the given raw API key to the stored key with its owner preloaded
// It is used by the authentication middleware
func (m *Module) ValidateApiKey(ctx context.Context, apiKey string) (ApiKey, error) {
	return m.service.ValidateApiKey(ctx, apiKey)
}
