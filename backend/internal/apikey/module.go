package apikey

import (
	"context"
	"errors"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/italypaleale/francis/host/local"
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/appconfig"
	"github.com/pocket-id/pocket-id/backend/internal/httpserver"
	"github.com/pocket-id/pocket-id/backend/internal/model"
)

type Dependencies struct {
	DB              *gorm.DB
	Actors          *local.Host
	StaticApiKey    string
	AppConfig       appconfig.AppConfigResolver
	EmailSender     APIKeyExpiryEmailSender
	CleanupDisabled bool
}

type Module struct {
	service *Service
	handler *handler
}

func New(ctx context.Context, deps Dependencies) (*Module, error) {
	service, err := newService(ctx, deps.DB, deps.StaticApiKey)
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
// authWithoutApiKey disables API key authentication so an API key cannot be used to mint or renew further API keys
func (m *Module) RegisterRoutes(apiGroup *gin.RouterGroup, auth, authWithoutApiKey gin.HandlerFunc) {
	group := apiGroup.Group("/api-keys")
	group.GET("", auth, httpserver.Handle(m.handler.list))
	group.POST("", authWithoutApiKey, httpserver.Handle(m.handler.create))
	group.POST("/:id/renew", authWithoutApiKey, httpserver.Handle(m.handler.renew))
	group.DELETE("/:id", auth, httpserver.Handle(m.handler.revoke))
}

// ValidateApiKey resolves the user that owns the given raw API key
// It is used by the authentication middleware
func (m *Module) ValidateApiKey(ctx context.Context, apiKey string) (model.User, error) {
	return m.service.ValidateApiKey(ctx, apiKey)
}
