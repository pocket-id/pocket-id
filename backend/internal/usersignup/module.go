package usersignup

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/dto"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	httpapi "github.com/pocket-id/pocket-id/backend/internal/utils/huma"
)

type TokenService interface {
	GenerateAccessToken(user model.User, authenticationMethod string) (string, error)
}

type AuditLogger interface {
	Create(ctx context.Context, event model.AuditLogEvent, ipAddress, userAgent, userID string, data model.AuditLogData, tx *gorm.DB) (model.AuditLog, bool)
}

type AppConfigProvider interface {
	GetDbConfig() *model.AppConfig
}

type UserCreator interface {
	CreateUserInternal(ctx context.Context, input dto.UserCreateDto, isLdapSync bool, tx *gorm.DB) (model.User, error)
}

type Dependencies struct {
	DB *gorm.DB

	Signer      TokenService
	AuditLog    AuditLogger
	AppConfig   AppConfigProvider
	UserCreator UserCreator
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
	createTokenOperation := signupOperation("create-signup-token", http.MethodPost, "/api/signup-tokens", "Create signup token")
	createTokenOperation.DefaultStatus = http.StatusCreated
	adminAuth(&createTokenOperation)
	httpapi.Register(api, createTokenOperation, m.handler.createSignupToken)

	listTokensOperation := signupOperation("list-signup-tokens", http.MethodGet, "/api/signup-tokens", "List signup tokens")
	adminAuth(&listTokensOperation)
	httpapi.Register(api, listTokensOperation, m.handler.listSignupTokens)

	deleteTokenOperation := signupOperation("delete-signup-token", http.MethodDelete, "/api/signup-tokens/{id}", "Delete signup token")
	deleteTokenOperation.DefaultStatus = http.StatusNoContent
	adminAuth(&deleteTokenOperation)
	httpapi.Register(api, deleteTokenOperation, m.handler.deleteSignupToken)

	selfSignupOperation := signupOperation("signup", http.MethodPost, "/api/signup", "Sign up")
	selfSignupOperation.DefaultStatus = http.StatusCreated
	selfSignupOperation.Middlewares = append(selfSignupOperation.Middlewares, signupRateLimit)
	httpapi.Register(api, selfSignupOperation, m.handler.signup)

	setupAvailableOperation := signupOperation("check-initial-admin-setup", http.MethodGet, "/api/signup/setup", "Check initial admin setup availability")
	setupAvailableOperation.DefaultStatus = http.StatusNoContent
	httpapi.Register(api, setupAvailableOperation, m.handler.checkInitialAdminSetupAvailable)

	httpapi.Register(api, signupOperationForInitialAdmin(), m.handler.signUpInitialAdmin)
}

func signupOperation(id, method, path, summary string) huma.Operation {
	return huma.Operation{OperationID: id, Method: method, Path: path, Summary: summary, Tags: []string{"Users"}}
}

func signupOperationForInitialAdmin() huma.Operation {
	return signupOperation("signup-initial-admin", http.MethodPost, "/api/signup/setup", "Sign up initial admin user")
}
