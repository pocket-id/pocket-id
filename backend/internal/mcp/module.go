package mcp

import (
	"context"
	"maps"
	"net/http"
	"slices"

	"github.com/gin-gonic/gin"
	"github.com/go-fuego/fuego/option"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/pocket-id/pocket-id/backend/internal/authz"
	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/httpserver"
	"github.com/pocket-id/pocket-id/backend/internal/openapi"
)

const instructions = "Pocket ID is a user-friendly, passwordless-only OpenID Connect provider and OAuth 2.0 authorization server. " +
	"Use these tools to manage its users, groups and OIDC clients, and use search_docs to answer questions about setting up Pocket ID or connecting apps to it. " +
	"You only see the tools the signed-in user has granted you."

type Dependencies struct {
	Tools      *Tools
	HTTPClient *http.Client
}

// Module serves the MCP endpoint, which exposes the registered tools to AI assistants
type Module struct {
	handler http.Handler
}

type callContextKey struct{}

func New(deps Dependencies) *Module {
	registerDocsTool(deps.Tools, newDocsIndex(deps.HTTPClient, docsIndexURL))

	schemaCache := sdk.NewSchemaCache()
	newServer := func(r *http.Request) *sdk.Server {
		call, _ := r.Context().Value(callContextKey{}).(Call)
		server := sdk.NewServer(
			&sdk.Implementation{Name: "pocket-id", Title: "Pocket ID", Version: common.Version},
			&sdk.ServerOptions{Instructions: instructions, SchemaCache: schemaCache},
		)
		deps.Tools.addAllowed(server, call)
		return server
	}

	return &Module{
		handler: sdk.NewStreamableHTTPHandler(newServer, &sdk.StreamableHTTPOptions{
			// Stateless requests need no session store, so the endpoint works the same behind several replicas
			Stateless:    true,
			JSONResponse: true,
			// The SDK rejects requests that reach a loopback address with another Host header, which is every request behind a reverse proxy on the same host
			// Every request is authenticated before it reaches the SDK, so DNS rebinding gains an attacker nothing
			DisableLocalhostProtection: true,
		}),
	}
}

func (m *Module) RegisterRoutes(r *authz.Router) {
	for _, method := range []string{http.MethodPost, http.MethodGet, http.MethodDelete} {
		r.Dispatch(method, authz.MCPPath, m.serve)
	}
	r.Public().GET(authz.MCPResourceMetadataPath, httpserver.Endpoint{Handler: protectedResourceMetadata},
		option.Summary("Get OAuth 2.0 protected resource metadata of the MCP endpoint"),
		option.Description("Returns the RFC 9728 protected resource metadata of Pocket ID's MCP endpoint, which MCP clients use to find the authorization server and the scopes they can request"),
		option.Tags("Well Known"),
		openapi.Response[map[string]any](http.StatusOK, "OAuth 2.0 protected resource metadata"),
	)
}

func (m *Module) serve(c *gin.Context) {
	call := Call{
		Principal: authz.PrincipalFrom(c),
		IPAddress: c.ClientIP(),
		UserAgent: c.Request.UserAgent(),
	}
	ctx := context.WithValue(c.Request.Context(), callContextKey{}, call)
	m.handler.ServeHTTP(c.Writer, c.Request.WithContext(ctx))
}

func protectedResourceMetadata(c *gin.Context) {
	heldScopes := authz.UserScopes(true, authz.KindOAuthUser)
	scopes := make([]string, 0, len(heldScopes)+1)
	for _, scope := range slices.Sorted(maps.Keys(heldScopes)) {
		scopes = append(scopes, string(scope))
	}
	// MCP clients request the listed scopes, so offline_access is listed too to get them a refresh token
	scopes = append(scopes, "offline_access")

	c.JSON(http.StatusOK, gin.H{
		"resource":                 authz.MCPResourceURI(common.EnvConfig.AppURL),
		"authorization_servers":    []string{common.EnvConfig.AppURL},
		"scopes_supported":         scopes,
		"bearer_methods_supported": []string{"header"},
		"resource_name":            "Pocket ID",
	})
}
