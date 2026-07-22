package apikey

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/model"
	httpapi "github.com/pocket-id/pocket-id/backend/internal/utils/huma"
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
func (m *Module) RegisterRoutes(api huma.API, auth, authWithoutAPIKey func(*huma.Operation)) {
	httpapi.Register(api, huma.Operation{
		OperationID: "list-api-keys",
		Method:      http.MethodGet,
		Path:        "/api/api-keys",
		Summary:     "List API keys",
		Tags:        []string{"API Keys"},
	}, m.handler.list, auth)

	httpapi.Register(api, huma.Operation{
		OperationID:   "create-api-key",
		Method:        http.MethodPost,
		Path:          "/api/api-keys",
		Summary:       "Create API key",
		Tags:          []string{"API Keys"},
		DefaultStatus: http.StatusCreated,
	}, m.handler.create, authWithoutAPIKey)

	httpapi.Register(api, huma.Operation{
		OperationID: "renew-api-key",
		Method:      http.MethodPost,
		Path:        "/api/api-keys/{id}/renew",
		Summary:     "Renew API key",
		Tags:        []string{"API Keys"},
	}, m.handler.renew, authWithoutAPIKey)

	httpapi.Register(api, huma.Operation{
		OperationID:   "revoke-api-key",
		Method:        http.MethodDelete,
		Path:          "/api/api-keys/{id}",
		Summary:       "Revoke API key",
		Tags:          []string{"API Keys"},
		DefaultStatus: http.StatusNoContent,
	}, m.handler.revoke, auth)
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
