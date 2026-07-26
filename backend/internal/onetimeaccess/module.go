package onetimeaccess

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/italypaleale/francis/host/local"
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/appconfig"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	"github.com/pocket-id/pocket-id/backend/internal/utils/email"
	httpapi "github.com/pocket-id/pocket-id/backend/internal/utils/huma"
)

// EmailData is the data rendered in the one-time access email
type EmailData struct {
	Code              string
	LoginLink         string
	LoginLinkWithCode string
	ExpirationString  string
}

// EmailSender sends the one-time access email
type EmailSender interface {
	SendOneTimeAccessEmail(ctx context.Context, dbConfig *appconfig.AppConfigModel, to email.Address, data EmailData) error
}

type TokenService interface {
	GenerateAccessToken(user model.User, authenticationMethod string, sessionDuration time.Duration) (string, error)
}

type AuditLogger interface {
	Create(ctx context.Context, event model.AuditLogEvent, ipAddress, userAgent, userID string, data model.AuditLogData, tx *gorm.DB) (model.AuditLog, bool)
}

type UserProvider interface {
	GetUser(ctx context.Context, userID string) (model.User, error)
}

// AppConfigResolver loads the current application configuration, so handlers can pass it explicitly to the service methods that need it
type AppConfigResolver interface {
	GetConfig(ctx context.Context) (*appconfig.AppConfigModel, error)
}

type Dependencies struct {
	DB     *gorm.DB
	Actors *local.Host

	Signer       TokenService
	AuditLog     AuditLogger
	UserProvider UserProvider
	EmailSender  EmailSender
	AppConfig    AppConfigResolver
}

type Module struct {
	service *Service
	handler *handler
}

func New(deps Dependencies) (*Module, error) {
	// Register the actor that manages a one-time access token
	// Each token is its own actor, whose actor ID is the token's value
	err := deps.Actors.RegisterActor(TokenActorType, NewTokenActor)
	if err != nil {
		return nil, fmt.Errorf("error registering the %s actor: %w", TokenActorType, err)
	}

	service := newService(deps, deps.Actors.Service())
	return &Module{
		service: service,
		handler: newHandler(service, deps.AppConfig),
	}, nil
}

// RegisterRoutes mounts the one-time access token endpoints
// auth guards the admin routes and ownAuth the current user's own token, while the rate limiters throttle the public exchange and email endpoints
func (m *Module) RegisterRoutes(api huma.API, auth, ownAuth func(*huma.Operation), exchangeRateLimit, emailRateLimit func(huma.Context, func(huma.Context))) {
	httpapi.Register(api, huma.Operation{
		OperationID:   "create-own-one-time-access-token",
		Method:        http.MethodPost,
		Path:          "/api/users/me/one-time-access-token",
		Summary:       "Create one-time access token for current user",
		Tags:          []string{"Users"},
		DefaultStatus: http.StatusCreated,
	}, m.handler.createOwnToken, ownAuth)

	httpapi.Register(api, huma.Operation{
		OperationID:   "create-user-one-time-access-token",
		Method:        http.MethodPost,
		Path:          "/api/users/{id}/one-time-access-token",
		Summary:       "Create one-time access token for user",
		Tags:          []string{"Users"},
		DefaultStatus: http.StatusCreated,
	}, m.handler.createTokenForUser, auth)

	httpapi.Register(api, huma.Operation{
		OperationID:   "request-user-one-time-access-email",
		Method:        http.MethodPost,
		Path:          "/api/users/{id}/one-time-access-email",
		Summary:       "Request one-time access email for user",
		Tags:          []string{"Users"},
		DefaultStatus: http.StatusNoContent,
	}, m.handler.requestEmailAsAdmin, auth)

	httpapi.Register(api, huma.Operation{
		OperationID: "exchange-one-time-access-token",
		Method:      http.MethodPost,
		Path:        "/api/one-time-access-token/{token}",
		Summary:     "Exchange one-time access token",
		Tags:        []string{"Users"},
	}, m.handler.exchangeToken, httpapi.WithMiddleware(exchangeRateLimit))

	httpapi.Register(api, huma.Operation{
		OperationID:   "request-one-time-access-email",
		Method:        http.MethodPost,
		Path:          "/api/one-time-access-email",
		Summary:       "Request one-time access email",
		Tags:          []string{"Users"},
		DefaultStatus: http.StatusNoContent,
	}, m.handler.requestEmailAsUnauthenticatedUser, httpapi.WithMiddleware(emailRateLimit))
}
