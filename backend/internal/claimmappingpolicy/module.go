package claimmappingpolicy

import (
	"net/http"

	"gorm.io/gorm"

	"github.com/go-fuego/fuego/option"
	"github.com/pocket-id/pocket-id/backend/internal/authz"
	"github.com/pocket-id/pocket-id/backend/internal/httpserver"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	"github.com/pocket-id/pocket-id/backend/internal/openapi"
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
	policies := r.Group("/admin/oidc/claim-mapping-policies")
	policies.GET("", authz.OidcClaimMappingPoliciesRead, httpserver.JSON(http.StatusOK, m.handler.list),
		option.Summary("List OIDC claim mapping policies"),
		option.Description("Get a paginated list of OIDC claim mapping policies with optional search and sorting"),
		option.Tags("OIDC Claim Mapping Policies"),
		option.Query("search", "Search term to filter claim mapping policies by name"),
		openapi.List[OidcClaimMappingPolicy](),
	)
	policies.POST("", authz.OidcClaimMappingPoliciesWrite, httpserver.BodyJSON(http.StatusCreated, m.handler.create),
		option.Summary("Create OIDC claim mapping policy"),
		option.Description("Create a new OIDC claim mapping policy"),
		option.Tags("OIDC Claim Mapping Policies"),
	)
	policies.GET("/:id", authz.OidcClaimMappingPoliciesRead, httpserver.JSON(http.StatusOK, m.handler.get),
		option.Summary("Get an OIDC claim mapping policy"),
		option.Description("Retrieve an existing OIDC claim mapping policy"),
		option.Tags("OIDC Claim Mapping Policies"),
		option.Path("id", "Claim mapping policy ID"),
	)
	updateDocs := option.Group(
		option.Summary("Update an OIDC claim mapping policy"),
		option.Description("Update an existing OIDC claim mapping policy, where PATCH changes only the fields in the body"),
		option.Tags("OIDC Claim Mapping Policies"),
		option.Path("id", "Claim mapping policy ID"),
	)
	policies.PUT("/:id", authz.OidcClaimMappingPoliciesWrite, httpserver.UpdateJSON(m.handler.update), updateDocs)
	policies.PATCH("/:id", authz.OidcClaimMappingPoliciesWrite, httpserver.UpdateJSON(m.handler.update), updateDocs)
	policies.DELETE("/:id", authz.OidcClaimMappingPoliciesWrite, httpserver.NoContent(m.handler.delete),
		option.Summary("Delete an OIDC claim mapping policy"),
		option.Description("Delete an OIDC claim mapping policy by ID"),
		option.Tags("OIDC Claim Mapping Policies"),
		option.Path("id", "Claim mapping policy ID"),
	)

	// Clients are attached from the policy side, a client without a policy falls back to the default one
	policies.GET("/:id/clients", authz.OidcClaimMappingPoliciesRead, httpserver.JSON(http.StatusOK, m.handler.listClients),
		option.Summary("List OIDC clients using a claim mapping policy"),
		option.Description("Get a paginated list of the OIDC clients assigned to a claim mapping policy"),
		option.Tags("OIDC Claim Mapping Policies"),
		option.Path("id", "Claim mapping policy ID"),
		option.Query("search", "Search term to filter clients by name"),
		openapi.List[model.OidcClient](),
	)
	policies.GET("/:id/assignable-clients", authz.OidcClaimMappingPoliciesRead, httpserver.JSON(http.StatusOK, m.handler.listAssignableClients),
		option.Summary("List OIDC clients that can still be assigned to a claim mapping policy"),
		option.Description("Get a paginated list of the OIDC clients that are not on this claim mapping policy yet"),
		option.Tags("OIDC Claim Mapping Policies"),
		option.Path("id", "Claim mapping policy ID"),
		option.Query("search", "Search term to filter clients by name"),
		openapi.List[model.OidcClient](),
	)
	policies.POST("/:id/clients/:clientId", authz.OidcClaimMappingPoliciesWrite, httpserver.NoContent(m.handler.assignClient),
		option.Summary("Assign an OIDC client to a claim mapping policy"),
		option.Description("Point an OIDC client at this claim mapping policy, replacing the policy it was on"),
		option.Tags("OIDC Claim Mapping Policies"),
		option.Path("id", "Claim mapping policy ID"),
		option.Path("clientId", "OIDC Client ID"),
	)
	policies.DELETE("/:id/clients/:clientId", authz.OidcClaimMappingPoliciesWrite, httpserver.NoContent(m.handler.removeClient),
		option.Summary("Detach an OIDC client from a claim mapping policy"),
		option.Description("Remove an OIDC client from this claim mapping policy, so it falls back to the default one"),
		option.Tags("OIDC Claim Mapping Policies"),
		option.Path("id", "Claim mapping policy ID"),
		option.Path("clientId", "OIDC Client ID"),
	)

	scopes := r.Group("/admin/oidc/custom-scopes")
	scopes.GET("", authz.OidcClaimMappingPoliciesRead, httpserver.JSON(http.StatusOK, m.handler.listCustomScopes),
		option.Summary("List custom OIDC scopes"),
		option.Description("Get a paginated list of the custom OIDC scopes with optional search and sorting"),
		option.Tags("OIDC Custom Scopes"),
		option.Query("search", "Search term to filter scopes by key or name"),
		openapi.List[CustomScope](),
	)
	scopes.GET("/keys", authz.OidcClaimMappingPoliciesRead, httpserver.JSON(http.StatusOK, m.handler.listCustomScopeKeys),
		option.Summary("List the keys of all custom OIDC scopes"),
		option.Description("Get the keys of every custom OIDC scope, sorted, for pickers that offer all of them at once"),
		option.Tags("OIDC Custom Scopes"),
	)
	scopes.POST("", authz.OidcClaimMappingPoliciesWrite, httpserver.BodyJSON(http.StatusCreated, m.handler.createCustomScope),
		option.Summary("Create a custom OIDC scope"),
		option.Description("Create a custom OIDC scope, which claim mappings can then be released with"),
		option.Tags("OIDC Custom Scopes"),
	)
	updateCustomScopeDocs := option.Group(
		option.Summary("Update a custom OIDC scope"),
		option.Description("Update the name and description of a custom OIDC scope, whose key can't change, where PATCH changes only the fields in the body"),
		option.Tags("OIDC Custom Scopes"),
		option.Path("id", "Custom scope ID"),
	)
	scopes.PUT("/:id", authz.OidcClaimMappingPoliciesWrite, httpserver.UpdateJSON(m.handler.updateCustomScope), updateCustomScopeDocs)
	scopes.PATCH("/:id", authz.OidcClaimMappingPoliciesWrite, httpserver.UpdateJSON(m.handler.updateCustomScope), updateCustomScopeDocs)
	scopes.DELETE("/:id", authz.OidcClaimMappingPoliciesWrite, httpserver.NoContent(m.handler.deleteCustomScope),
		option.Summary("Delete a custom OIDC scope"),
		option.Description("Delete a custom OIDC scope that no claim mapping is released with"),
		option.Tags("OIDC Custom Scopes"),
		option.Path("id", "Custom scope ID"),
	)
}
