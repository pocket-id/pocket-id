package emailverification

import (
	"context"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/italypaleale/francis/host/local"
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/appconfig"
)

type AppConfigResolver interface {
	GetConfig(ctx context.Context) (*appconfig.AppConfigModel, error)
}

type Dependencies struct {
	DB     *gorm.DB
	Actors *local.Host

	Users       UserProvider
	EmailSender EmailSender
	AppConfig   AppConfigResolver
	AppURL      string
}

type Module struct {
	service *Service
	handler *handler
}

func New(deps Dependencies) (*Module, error) {
	err := deps.Actors.RegisterActor(ActorType, NewActor)
	if err != nil {
		return nil, fmt.Errorf("error registering the %s actor: %w", ActorType, err)
	}

	service := newService(deps.DB, deps.Actors.Service(), deps.Users, deps.EmailSender, deps.AppURL)
	return &Module{
		service: service,
		handler: newHandler(service, deps.AppConfig),
	}, nil
}

// RegisterRoutes mounts the email verification endpoints
func (m *Module) RegisterRoutes(apiGroup *gin.RouterGroup, userAuth, sendRateLimit, verifyRateLimit gin.HandlerFunc) {
	apiGroup.POST("/users/me/send-email-verification", sendRateLimit, userAuth, m.handler.send)
	apiGroup.POST("/users/me/verify-email", verifyRateLimit, userAuth, m.handler.verify)
}
