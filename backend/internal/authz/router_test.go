package authz

import (
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func noop(c *gin.Context) {
	c.Status(http.StatusNoContent)
}

func TestIsDeclared(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	m := NewMiddleware()
	apiGroup := engine.Group("/api")
	api := m.Router(apiGroup)

	// Declare routes through every router variant
	api.GET("/users", UsersRead, noop)
	api.Group("/api-keys").POST("", AccountAPIKeysCreate, noop)
	api.Group("/nested").Group("/deeper").DELETE("/:id", UsersWrite, noop)
	api.Optional().GET("/optional", AccountSession, noop)
	api.Public().POST("/signup", noop)
	m.Router(engine.Group("/")).Optional().GET("/authorize", AccountSession, noop)

	// Register routes past the routers
	apiGroup.GET("/forgotten", noop)
	apiGroup.POST("/users", noop)

	declared := map[string]bool{}
	for _, route := range engine.Routes() {
		declared[route.Method+" "+route.Path] = m.IsDeclared(route.Method, route.Path)
	}
	require.Equal(t, map[string]bool{
		"GET /api/users":                true,
		"POST /api/api-keys":            true,
		"DELETE /api/nested/deeper/:id": true,
		"GET /api/optional":             true,
		"POST /api/signup":              true,
		"GET /authorize":                true,
		"GET /api/forgotten":            false,
		"POST /api/users":               false,
	}, declared)
}

func TestRouterRejectsUnknownScopes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := NewMiddleware().Router(gin.New().Group("/api"))

	require.PanicsWithValue(t, `route GET /api/users requires unknown scope "users:everything"`, func() {
		r.GET("/users", Scope("users:everything"), noop)
	})
}

func TestJoinPathsMatchesGin(t *testing.T) {
	gin.SetMode(gin.TestMode)

	for _, test := range []struct{ base, relative string }{
		{"/api", ""},
		{"/api", "/users"},
		{"/api/", "users"},
		{"/api", "/users/"},
		{"/", "/authorize"},
		{"/api/api-keys", ""},
	} {
		engine := gin.New()
		engine.Group(test.base).GET(test.relative, noop)

		require.Equal(t, engine.Routes()[0].Path, joinPaths(engine.Group(test.base).BasePath(), test.relative), "%+v", test)
	}
}
