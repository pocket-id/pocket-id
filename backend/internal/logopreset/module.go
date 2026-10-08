package logopreset

import (
	"net/http"

	"github.com/pocket-id/pocket-id/backend/internal/authz"
	"github.com/pocket-id/pocket-id/backend/internal/httpserver"
)

type Dependencies struct {
	HTTPClient *http.Client

	// BaseURL is the root of the icon library, or empty when the icon library is disabled
	BaseURL string
}

// Module lets admins search an icon collection in the selfh.st/icons layout for OIDC client logos
// The chosen icon is downloaded through the regular logo URL flow of the OIDC client update
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

// RegisterRoutes mounts the logo preset endpoints
func (m *Module) RegisterRoutes(r *authz.Router) {
	r.GET("/oidc/logo-presets", authz.OidcClientsRead, httpserver.Handle(m.handler.search))
}
