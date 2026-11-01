package claimmappingpolicy

import (
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/authz"
	"github.com/pocket-id/pocket-id/backend/internal/httpserver"
)

type Dependencies struct {
	DB *gorm.DB
}

type Module struct {
	handler *handler
}

func New(deps Dependencies) *Module {
	return &Module{
		handler: newHandler(newService(deps.DB)),
	}
}

// RegisterRoutes mounts the admin CRUD endpoints and the client assignment endpoints
func (m *Module) RegisterRoutes(r *authz.Router) {
	policies := r.Group("/oidc/claim-mapping-policies")
	policies.GET("", authz.OidcClaimMappingPoliciesRead, httpserver.Handle(m.handler.list))
	policies.POST("", authz.OidcClaimMappingPoliciesWrite, httpserver.Handle(m.handler.create))
	policies.GET("/:id", authz.OidcClaimMappingPoliciesRead, httpserver.Handle(m.handler.get))
	policies.PUT("/:id", authz.OidcClaimMappingPoliciesWrite, httpserver.Handle(m.handler.update))
	policies.DELETE("/:id", authz.OidcClaimMappingPoliciesWrite, httpserver.Handle(m.handler.delete))

	// Clients are attached from the policy side, a client without a policy falls back to the default one
	policies.GET("/:id/clients", authz.OidcClaimMappingPoliciesRead, httpserver.Handle(m.handler.listClients))
	policies.GET("/:id/assignable-clients", authz.OidcClaimMappingPoliciesRead, httpserver.Handle(m.handler.listAssignableClients))
	policies.POST("/:id/clients/:clientId", authz.OidcClaimMappingPoliciesWrite, httpserver.Handle(m.handler.assignClient))
	policies.DELETE("/:id/clients/:clientId", authz.OidcClaimMappingPoliciesWrite, httpserver.Handle(m.handler.removeClient))
}
