package bootstrap

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestEveryAPIRouteDeclaresItsAccess builds the complete route table and fails when an API route was registered without declaring a scope or public access
func TestEveryAPIRouteDeclaresItsAccess(t *testing.T) {
	engine, auth, err := routeTable()
	require.NoError(t, err)

	// Collect every API route that bypassed the authz routers
	apiRoutes := 0
	var undeclared []string
	for _, route := range engine.Routes() {
		if !strings.HasPrefix(route.Path, "/api/") {
			continue
		}
		apiRoutes++
		if !auth.IsDeclared(route.Method, route.Path) {
			undeclared = append(undeclared, route.Method+" "+route.Path)
		}
	}

	require.Empty(t, undeclared, "register these routes through authz.Router with a scope, or through Public() when they need no authentication")

	// Guard against the table silently shrinking, which would make the coverage check vacuous
	require.Greater(t, apiRoutes, 100)
}

// TestOpenAPISpec fails when the documentation of a route doesn't produce a valid OpenAPI spec
func TestOpenAPISpec(t *testing.T) {
	spec, err := OpenAPISpec()
	require.NoError(t, err)

	var parsed struct {
		Paths map[string]map[string]struct {
			Tags []string `json:"tags"`
		} `json:"paths"`
		Tags []struct {
			Name string `json:"name"`
		} `json:"tags"`
	}
	require.NoError(t, json.Unmarshal(spec, &parsed))

	// Every operation belongs to a group of the API reference, which lists only the tags the spec declares
	declared := map[string]bool{}
	for _, tag := range parsed.Tags {
		declared[tag.Name] = true
	}
	for path, operations := range parsed.Paths {
		for method, operation := range operations {
			require.NotEmpty(t, operation.Tags, "%s %s has no tag", method, path)
			for _, tag := range operation.Tags {
				require.True(t, declared[tag], "%s %s has the undeclared tag %q, add it to the tags in internal/openapi", method, path, tag)
			}
		}
	}
}
