package emailverification

import (
	"fmt"

	"github.com/gin-gonic/gin"
	francishost "github.com/italypaleale/francis/host"
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/appconfig"
	"github.com/pocket-id/pocket-id/backend/internal/authz"
	"github.com/pocket-id/pocket-id/backend/internal/httpserver"
)

type Dependencies struct {
	DB     *gorm.DB
	Actors francishost.Host

	Users       UserProvider
	EmailSender EmailSender
	AppConfig   appconfig.AppConfigResolver
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
func (m *Module) RegisterRoutes(r *authz.Router, sendRateLimit, verifyRateLimit gin.HandlerFunc) {
	r.POST("/users/me/send-email-verification", authz.AccountWrite, sendRateLimit, httpserver.Handle(m.handler.send))
	r.POST("/users/me/verify-email", authz.AccountWrite, verifyRateLimit, httpserver.Handle(m.handler.verify))
}
