package emailverification

import (
	"context"
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

	EmailSender EmailSender
	ScimSync    ScimSyncScheduler
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

	service := newService(deps.DB, deps.Actors.Service(), deps.EmailSender, deps.ScimSync, deps.AppURL)
	return &Module{
		service: service,
		handler: newHandler(service, deps.AppConfig),
	}, nil
}

// Send issues a new verification token for the user's current email and emails the verification link
func (m *Module) Send(ctx context.Context, dbConfig *appconfig.AppConfigModel, userID string) error {
	return m.service.Send(ctx, dbConfig, userID)
}

// RequestEmailChange emails a verification link to a new address, which replaces the user's current email once the link is used
func (m *Module) RequestEmailChange(ctx context.Context, dbConfig *appconfig.AppConfigModel, userID, newEmail string) error {
	return m.service.RequestEmailChange(ctx, dbConfig, userID, newEmail)
}

// RegisterRoutes mounts the email verification endpoints
func (m *Module) RegisterRoutes(r *authz.Router, sendRateLimit, verifyRateLimit gin.HandlerFunc) {
	r.POST("/users/me/send-email-verification", authz.AccountWrite, sendRateLimit, httpserver.Handle(m.handler.send))
	r.POST("/users/me/verify-email", authz.AccountWrite, verifyRateLimit, httpserver.Handle(m.handler.verify))
	r.GET("/users/me/email-change", authz.AccountRead, httpserver.Handle(m.handler.pendingEmailChange))
	r.DELETE("/users/me/email-change", authz.AccountWrite, httpserver.Handle(m.handler.cancelEmailChange))
}
