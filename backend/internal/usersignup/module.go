package usersignup

import (
	"context"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/appconfig"
	"github.com/pocket-id/pocket-id/backend/internal/dto"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	httpapi "github.com/pocket-id/pocket-id/backend/internal/utils/huma"
)

type TokenService interface {
	GenerateAccessToken(user model.User, authenticationMethod string, sessionDuration time.Duration) (string, error)
}

type AuditLogger interface {
	Create(ctx context.Context, event model.AuditLogEvent, ipAddress, userAgent, userID string, data model.AuditLogData, tx *gorm.DB) (model.AuditLog, bool)
}

type UserCreator interface {
	CreateUserInternal(ctx context.Context, dbConfig *appconfig.AppConfigModel, input dto.UserCreateDto, isLdapSync bool, tx *gorm.DB) (model.User, error)
}

// AppConfigResolver loads the current application configuration, so handlers can pass it explicitly to the service methods that need it
type AppConfigResolver interface {
	GetConfig(ctx context.Context) (*appconfig.AppConfigModel, error)
}

type Dependencies struct {
	DB *gorm.DB

	Signer      TokenService
	AuditLog    AuditLogger
	UserCreator UserCreator
	AppConfig   AppConfigResolver
}

type Module struct {
	service *Service
	handler *handler
}

func New(deps Dependencies) *Module {
	service := newService(deps)
	return &Module{
		service: service,
		handler: newHandler(service, deps.AppConfig),
	}
}

// RegisterRoutes mounts the signup and signup-token management endpoints
// adminAuth guards the admin token-management routes; signupRateLimit throttles public self-signup
func (m *Module) RegisterRoutes(api huma.API, adminAuth func(*huma.Operation), signupRateLimit func(huma.Context, func(huma.Context))) {
	httpapi.Register(api, huma.Operation{
		OperationID:   "create-signup-token",
		Method:        http.MethodPost,
		Path:          "/api/signup-tokens",
		Summary:       "Create signup token",
		Tags:          []string{"Users"},
		DefaultStatus: http.StatusCreated,
	}, m.handler.createSignupToken, adminAuth)

	httpapi.Register(api, huma.Operation{
		OperationID: "list-signup-tokens",
		Method:      http.MethodGet,
		Path:        "/api/signup-tokens",
		Summary:     "List signup tokens",
		Tags:        []string{"Users"},
	}, m.handler.listSignupTokens, adminAuth)

	httpapi.Register(api, huma.Operation{
		OperationID:   "delete-signup-token",
		Method:        http.MethodDelete,
		Path:          "/api/signup-tokens/{id}",
		Summary:       "Delete signup token",
		Tags:          []string{"Users"},
		DefaultStatus: http.StatusNoContent,
	}, m.handler.deleteSignupToken, adminAuth)

	httpapi.Register(api, huma.Operation{
		OperationID:   "signup",
		Method:        http.MethodPost,
		Path:          "/api/signup",
		Summary:       "Sign up",
		Tags:          []string{"Users"},
		DefaultStatus: http.StatusCreated,
	}, m.handler.signup, httpapi.WithMiddleware(signupRateLimit))

	httpapi.Register(api, huma.Operation{
		OperationID:   "check-initial-admin-setup",
		Method:        http.MethodGet,
		Path:          "/api/signup/setup",
		Summary:       "Check initial admin setup availability",
		Tags:          []string{"Users"},
		DefaultStatus: http.StatusNoContent,
	}, m.handler.checkInitialAdminSetupAvailable)

	httpapi.Register(api, huma.Operation{
		OperationID: "signup-initial-admin",
		Method:      http.MethodPost,
		Path:        "/api/signup/setup",
		Summary:     "Sign up initial admin user",
		Tags:        []string{"Users"},
	}, m.handler.signUpInitialAdmin)
}
