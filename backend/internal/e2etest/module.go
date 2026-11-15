//go:build e2etest

package e2etest

import (
	"fmt"

	francishost "github.com/italypaleale/francis/host"
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/appconfig"
	"github.com/pocket-id/pocket-id/backend/internal/authz"
	"github.com/pocket-id/pocket-id/backend/internal/httpserver"
	"github.com/pocket-id/pocket-id/backend/internal/signingkey"
	"github.com/pocket-id/pocket-id/backend/internal/storage"
)

type Dependencies struct {
	DB          *gorm.DB
	Actors      francishost.Host
	AppConfig   *appconfig.AppConfigService
	SigningKey  *signingkey.Key
	BrowserKey  BrowserKeyReloader
	LdapSyncer  LdapSyncer
	FileStorage storage.FileStorage
}

// Module exposes the endpoints that only builds for the end-to-end tests contain
type Module struct {
	handler *handler
}

func New(deps Dependencies) (*Module, error) {
	service := &Service{
		db:               deps.DB,
		actors:           deps.Actors,
		appConfigService: deps.AppConfig,
		signingKey:       deps.SigningKey,
		browserKey:       deps.BrowserKey,
		ldapSyncer:       deps.LdapSyncer,
		fileStorage:      deps.FileStorage,
	}
	err := service.initExternalIdP()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize external IdP: %w", err)
	}
	return &Module{handler: &handler{service: service}}, nil
}

// RegisterRoutes mounts the endpoints the Playwright suite uses to reset the instance and to mint tokens
func (m *Module) RegisterRoutes(r *authz.PublicRouter) {
	r.POST("/test/reset", httpserver.Handle(m.handler.resetAndSeedHandler))
	r.POST("/test/accesstoken", httpserver.Handle(m.handler.signAccessToken))
	r.POST("/test/refreshtoken", httpserver.Handle(m.handler.signRefreshToken))

	r.GET("/externalidp/jwks.json", httpserver.Handle(m.handler.externalIdPJWKS))
	r.POST("/externalidp/sign", httpserver.Handle(m.handler.externalIdPSignToken))
}
