package authz

import (
	"fmt"
	"net/http"
	"path"
	"slices"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-fuego/fuego"

	"github.com/pocket-id/pocket-id/backend/internal/httpserver"
)

// AdminPathPrefix is the path below which every route requiring an admin scope lives, and nothing else, so a reverse proxy can keep the admin API private with a single rule
const AdminPathPrefix = "/api/admin/"

// Access is what a route requires of its caller: a Scope, or Authenticated or Public for routes that check no scope
type Access interface {
	access()
}

func (Scope) access() {}

// routeAccess is the access of a route that checks no scope
type routeAccess string

func (routeAccess) access() {}

const (
	// Public routes require no credential at all
	Public routeAccess = "public"
	// Authenticated routes accept every valid credential whatever scopes it holds, for details that are harmless to any caller but should not be scraped anonymously
	Authenticated routeAccess = "authenticated"
)

// Router registers routes together with the access each one requires and their OpenAPI documentation
// Every route under /api must be registered through a Router, which a test over the complete route table checks with IsDeclared
type Router struct {
	group    *gin.RouterGroup
	auth     *Middleware
	optional bool
	handlers []gin.HandlerFunc
}

// Group returns a router for routes below the relative path
func (r *Router) Group(relativePath string) *Router {
	return &Router{group: r.group.Group(relativePath), auth: r.auth, optional: r.optional, handlers: r.handlers}
}

// Use returns a router whose routes run the handlers before authorization, for example to render its errors in a way the route's clients understand
func (r *Router) Use(handlers ...gin.HandlerFunc) *Router {
	return &Router{group: r.group.Group("", handlers...), auth: r.auth, optional: r.optional, handlers: r.handlers}
}

// With returns a router whose routes run the handlers after authorization and before the route's own handler, such as a rate limit or a body size limit
func (r *Router) With(handlers ...gin.HandlerFunc) *Router {
	return &Router{group: r.group, auth: r.auth, optional: r.optional, handlers: slices.Concat(r.handlers, handlers)}
}

// Optional returns a router whose routes let requests without a usable credential through as anonymous
// Handlers on these routes must check PrincipalFrom before relying on a signed-in user
func (r *Router) Optional() *Router {
	return &Router{group: r.group, auth: r.auth, optional: true, handlers: r.handlers}
}

func (r *Router) GET(relativePath string, access Access, handler httpserver.Endpoint, docs ...fuego.RouteOption) {
	r.Handle(http.MethodGet, relativePath, access, handler, docs...)
}

func (r *Router) POST(relativePath string, access Access, handler httpserver.Endpoint, docs ...fuego.RouteOption) {
	r.Handle(http.MethodPost, relativePath, access, handler, docs...)
}

func (r *Router) PUT(relativePath string, access Access, handler httpserver.Endpoint, docs ...fuego.RouteOption) {
	r.Handle(http.MethodPut, relativePath, access, handler, docs...)
}

func (r *Router) PATCH(relativePath string, access Access, handler httpserver.Endpoint, docs ...fuego.RouteOption) {
	r.Handle(http.MethodPatch, relativePath, access, handler, docs...)
}

func (r *Router) DELETE(relativePath string, access Access, handler httpserver.Endpoint, docs ...fuego.RouteOption) {
	r.Handle(http.MethodDelete, relativePath, access, handler, docs...)
}

// Handle registers a route that requires the access, running authorization before every other handler of the route
// The docs follow the documentation that the handler's types imply, so they can override it
func (r *Router) Handle(method, relativePath string, access Access, handler httpserver.Endpoint, docs ...fuego.RouteOption) {
	fullPath := joinPaths(r.group.BasePath(), relativePath)
	docs = slices.Concat(handler.Docs, docs)

	// An unknown scope can never be granted, so it is a programming error just like a duplicate route in gin
	scope, scoped := access.(Scope)
	if scoped && !scope.Known() {
		panic(fmt.Sprintf("route %s %s requires unknown scope %q", method, fullPath, scope))
	}

	// A route outside the admin prefix that needs an admin scope, or the other way round, would slip past a proxy that blocks the prefix or be blocked by it
	isAdmin := scoped && definitions[scope].category == CategoryAdmin
	if isAdmin != strings.HasPrefix(fullPath, AdminPathPrefix) {
		panic(fmt.Sprintf("route %s %s requires %q, so it must be registered below %s if and only if that is an admin scope", method, fullPath, access, AdminPathPrefix))
	}

	// Public routes run no authorization at all
	if access == Public {
		r.auth.register(r.group, method, relativePath, slices.Concat(r.handlers, []gin.HandlerFunc{handler.Handler}), docs, nil)
		return
	}

	r.auth.register(r.group, method, relativePath, r.chain(r.auth.require(access, r.optional), handler.Handler), docs, security(access, r.optional))
}

// Dispatch registers a route that serves several admin operations needing different scopes, such as the MCP endpoint
// It accepts every credential holding some admin scope except a browser session, and leaves the rest of the scope check to the handler, which must check the principal's scopes before each operation
func (r *Router) Dispatch(method, relativePath string, handler gin.HandlerFunc) {
	fullPath := joinPaths(r.group.BasePath(), relativePath)
	if !strings.HasPrefix(fullPath, AdminPathPrefix) {
		panic(fmt.Sprintf("dispatch route %s %s must be registered below %s", method, fullPath, AdminPathPrefix))
	}

	r.auth.register(r.group, method, relativePath, r.chain(r.auth.requireDispatch(), handler), nil, nil)
}

// chain runs authorization first, then the router's handlers and finally the route's own handler
func (r *Router) chain(authorize, handler gin.HandlerFunc) []gin.HandlerFunc {
	chain := make([]gin.HandlerFunc, 0, len(r.handlers)+2)
	chain = append(chain, authorize)
	chain = append(chain, r.handlers...)
	return append(chain, handler)
}

// joinPaths mirrors how gin builds a route's absolute path so declared routes match gin's route table
func joinPaths(absolutePath, relativePath string) string {
	if relativePath == "" {
		return absolutePath
	}

	finalPath := path.Join(absolutePath, relativePath)
	if strings.HasSuffix(relativePath, "/") && !strings.HasSuffix(finalPath, "/") {
		return finalPath + "/"
	}
	return finalPath
}
