package authz

import (
	"fmt"
	"net/http"
	"path"
	"strings"

	"github.com/gin-gonic/gin"
)

// Router registers routes together with the scope each one requires
// Every route under /api must be registered through a Router or PublicRouter, which a test over the complete route table checks with IsDeclared
type Router struct {
	group    *gin.RouterGroup
	auth     *Middleware
	optional bool
}

// Group returns a router for routes below the relative path
func (r *Router) Group(relativePath string) *Router {
	return &Router{group: r.group.Group(relativePath), auth: r.auth, optional: r.optional}
}

// Optional returns a router whose routes let requests without a usable credential through as anonymous
// Handlers on these routes must check PrincipalFrom before relying on a signed-in user
func (r *Router) Optional() *Router {
	return &Router{group: r.group, auth: r.auth, optional: true}
}

// Public returns a router for routes that require no authentication at all
func (r *Router) Public() *PublicRouter {
	return &PublicRouter{group: r.group, auth: r.auth}
}

func (r *Router) GET(relativePath string, scope Scope, handlers ...gin.HandlerFunc) {
	r.Handle(http.MethodGet, relativePath, scope, handlers...)
}

func (r *Router) POST(relativePath string, scope Scope, handlers ...gin.HandlerFunc) {
	r.Handle(http.MethodPost, relativePath, scope, handlers...)
}

func (r *Router) PUT(relativePath string, scope Scope, handlers ...gin.HandlerFunc) {
	r.Handle(http.MethodPut, relativePath, scope, handlers...)
}

func (r *Router) PATCH(relativePath string, scope Scope, handlers ...gin.HandlerFunc) {
	r.Handle(http.MethodPatch, relativePath, scope, handlers...)
}

func (r *Router) DELETE(relativePath string, scope Scope, handlers ...gin.HandlerFunc) {
	r.Handle(http.MethodDelete, relativePath, scope, handlers...)
}

// Handle registers a route that requires the scope, running authorization before every other handler of the route
func (r *Router) Handle(method, relativePath string, scope Scope, handlers ...gin.HandlerFunc) {
	// An unknown scope can never be granted, so it is a programming error just like a duplicate route in gin
	if !scope.Known() {
		panic(fmt.Sprintf("route %s %s requires unknown scope %q", method, joinPaths(r.group.BasePath(), relativePath), scope))
	}

	chain := make([]gin.HandlerFunc, 0, len(handlers)+1)
	chain = append(chain, r.auth.require(scope, r.optional))
	chain = append(chain, handlers...)
	r.group.Handle(method, relativePath, chain...)
	r.auth.declare(method, joinPaths(r.group.BasePath(), relativePath))
}

// PublicRouter registers routes that require no authentication
// Routing them through here keeps public access an explicit decision instead of a missing middleware
type PublicRouter struct {
	group *gin.RouterGroup
	auth  *Middleware
}

func (r *PublicRouter) GET(relativePath string, handlers ...gin.HandlerFunc) {
	r.Handle(http.MethodGet, relativePath, handlers...)
}

func (r *PublicRouter) POST(relativePath string, handlers ...gin.HandlerFunc) {
	r.Handle(http.MethodPost, relativePath, handlers...)
}

// Handle registers a public route
func (r *PublicRouter) Handle(method, relativePath string, handlers ...gin.HandlerFunc) {
	r.group.Handle(method, relativePath, handlers...)
	r.auth.declare(method, joinPaths(r.group.BasePath(), relativePath))
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
