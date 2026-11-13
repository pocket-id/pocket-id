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

// Router registers routes together with the scope each one requires and their OpenAPI documentation
// Every route under /api must be registered through a Router or PublicRouter, which a test over the complete route table checks with IsDeclared
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

// Public returns a router for routes that require no authentication at all
func (r *Router) Public() *PublicRouter {
	return &PublicRouter{group: r.group, auth: r.auth, handlers: r.handlers}
}

func (r *Router) GET(relativePath string, scope Scope, handler httpserver.Endpoint, docs ...fuego.RouteOption) {
	r.Handle(http.MethodGet, relativePath, scope, handler, docs...)
}

func (r *Router) POST(relativePath string, scope Scope, handler httpserver.Endpoint, docs ...fuego.RouteOption) {
	r.Handle(http.MethodPost, relativePath, scope, handler, docs...)
}

func (r *Router) PUT(relativePath string, scope Scope, handler httpserver.Endpoint, docs ...fuego.RouteOption) {
	r.Handle(http.MethodPut, relativePath, scope, handler, docs...)
}

func (r *Router) PATCH(relativePath string, scope Scope, handler httpserver.Endpoint, docs ...fuego.RouteOption) {
	r.Handle(http.MethodPatch, relativePath, scope, handler, docs...)
}

func (r *Router) DELETE(relativePath string, scope Scope, handler httpserver.Endpoint, docs ...fuego.RouteOption) {
	r.Handle(http.MethodDelete, relativePath, scope, handler, docs...)
}

// Handle registers a route that requires the scope, running authorization before every other handler of the route
// The docs follow the documentation that the handler's types imply, so they can override it
func (r *Router) Handle(method, relativePath string, scope Scope, handler httpserver.Endpoint, docs ...fuego.RouteOption) {
	// An unknown scope can never be granted, so it is a programming error just like a duplicate route in gin
	if !scope.Known() {
		panic(fmt.Sprintf("route %s %s requires unknown scope %q", method, joinPaths(r.group.BasePath(), relativePath), scope))
	}

	r.auth.register(r.group, method, relativePath, r.chain(r.auth.require(scope, r.optional), handler.Handler), slices.Concat(handler.Docs, docs))
}

// Dispatch registers a route that serves several operations needing different scopes, such as the MCP endpoint
// It accepts every credential except a browser session and leaves the scope check to the handler, which must check the principal's scopes before each operation
func (r *Router) Dispatch(method, relativePath string, handler gin.HandlerFunc) {
	r.auth.register(r.group, method, relativePath, r.chain(r.auth.requireDispatch(), handler), nil)
}

// chain runs authorization first, then the router's handlers and finally the route's own handler
func (r *Router) chain(authorize, handler gin.HandlerFunc) []gin.HandlerFunc {
	chain := make([]gin.HandlerFunc, 0, len(r.handlers)+2)
	chain = append(chain, authorize)
	chain = append(chain, r.handlers...)
	return append(chain, handler)
}

// PublicRouter registers routes that require no authentication
// Routing them through here keeps public access an explicit decision instead of a missing middleware
type PublicRouter struct {
	group    *gin.RouterGroup
	auth     *Middleware
	handlers []gin.HandlerFunc
}

func (r *PublicRouter) GET(relativePath string, handler httpserver.Endpoint, docs ...fuego.RouteOption) {
	r.Handle(http.MethodGet, relativePath, handler, docs...)
}

func (r *PublicRouter) POST(relativePath string, handler httpserver.Endpoint, docs ...fuego.RouteOption) {
	r.Handle(http.MethodPost, relativePath, handler, docs...)
}

// Handle registers a public route
func (r *PublicRouter) Handle(method, relativePath string, handler httpserver.Endpoint, docs ...fuego.RouteOption) {
	r.auth.register(r.group, method, relativePath, slices.Concat(r.handlers, []gin.HandlerFunc{handler.Handler}), slices.Concat(handler.Docs, docs))
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
