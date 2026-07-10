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
	listOperation := apiKeyOperation("list-api-keys", http.MethodGet, "/api/api-keys", "List API keys")
	auth(&listOperation)
	httpapi.Register(api, listOperation, m.handler.list)

	createOperation := apiKeyOperation("create-api-key", http.MethodPost, "/api/api-keys", "Create API key")
	createOperation.DefaultStatus = http.StatusCreated
	authWithoutAPIKey(&createOperation)
	httpapi.Register(api, createOperation, m.handler.create)

	renewOperation := apiKeyOperation("renew-api-key", http.MethodPost, "/api/api-keys/{id}/renew", "Renew API key")
	authWithoutAPIKey(&renewOperation)
	httpapi.Register(api, renewOperation, m.handler.renew)

	revokeOperation := apiKeyOperation("revoke-api-key", http.MethodDelete, "/api/api-keys/{id}", "Revoke API key")
	revokeOperation.DefaultStatus = http.StatusNoContent
	auth(&revokeOperation)
	httpapi.Register(api, revokeOperation, m.handler.revoke)
}

func apiKeyOperation(id, method, path, summary string) huma.Operation {
	return huma.Operation{OperationID: id, Method: method, Path: path, Summary: summary, Tags: []string{"API Keys"}}
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
