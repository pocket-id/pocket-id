package usersignup

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/dto"
	"github.com/pocket-id/pocket-id/backend/internal/model"
)

type TokenService interface {
	GenerateAccessToken(user model.User, authenticationMethod string, sessionDuration time.Duration) (string, error)
}

type AuditLogger interface {
	Create(ctx context.Context, event model.AuditLogEvent, ipAddress, userAgent, userID string, data model.AuditLogData, tx *gorm.DB) (model.AuditLog, bool)
}

type UserCreator interface {
	CreateUserInternal(ctx context.Context, input dto.UserCreateDto, isLdapSync bool, tx *gorm.DB) (model.User, error)
}

type Dependencies struct {
	DB *gorm.DB

	Signer      TokenService
	AuditLog    AuditLogger
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
		handler: newHandler(service),
	}
}

// RegisterRoutes mounts the signup and signup-token management endpoints
// adminAuth guards the admin token-management routes; signupRateLimit throttles public self-signup
func (m *Module) RegisterRoutes(apiGroup *gin.RouterGroup, adminAuth, signupRateLimit gin.HandlerFunc) {
	apiGroup.POST("/signup-tokens", adminAuth, m.handler.createSignupToken)
	apiGroup.GET("/signup-tokens", adminAuth, m.handler.listSignupTokens)
	apiGroup.DELETE("/signup-tokens/:id", adminAuth, m.handler.deleteSignupToken)
	apiGroup.POST("/signup", signupRateLimit, m.handler.signup)
	apiGroup.GET("/signup/setup", m.handler.checkInitialAdminSetupAvailable)
	apiGroup.POST("/signup/setup", m.handler.signUpInitialAdmin)
}
