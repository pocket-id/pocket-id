package apikey

import (
	"context"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/model"
)

type Dependencies struct {
	DB           *gorm.DB
	StaticApiKey string
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

	return &Module{
		service: service,
		handler: newHandler(service),
	}, nil
}

// RegisterRoutes mounts the API key management endpoints
// authWithoutApiKey disables API key authentication so an API key cannot be used to mint or renew further API keys
func (m *Module) RegisterRoutes(apiGroup *gin.RouterGroup, auth, authWithoutApiKey gin.HandlerFunc) {
	group := apiGroup.Group("/api-keys")
	group.GET("", auth, m.handler.list)
	group.POST("", authWithoutApiKey, m.handler.create)
	group.POST("/:id/renew", authWithoutApiKey, m.handler.renew)
	group.DELETE("/:id", auth, m.handler.revoke)
}

// ValidateApiKey resolves the user that owns the given raw API key
// It is used by the authentication middleware
func (m *Module) ValidateApiKey(ctx context.Context, apiKey string) (model.User, error) {
	return m.service.ValidateApiKey(ctx, apiKey)
}

// ListExpiringApiKeys returns API keys expiring within the given number of days that have not been notified yet
func (m *Module) ListExpiringApiKeys(ctx context.Context, daysAhead int) ([]ApiKey, error) {
	return m.service.ListExpiringApiKeys(ctx, daysAhead)
}

// MarkExpirationEmailSent records that the expiration notification email was sent for the given API key
func (m *Module) MarkExpirationEmailSent(ctx context.Context, apiKeyID string) error {
	return m.service.MarkExpirationEmailSent(ctx, apiKeyID)
}
