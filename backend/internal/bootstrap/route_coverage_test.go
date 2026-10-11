package bootstrap

import (
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"github.com/pocket-id/pocket-id/backend/internal/api"
	"github.com/pocket-id/pocket-id/backend/internal/apikey"
	"github.com/pocket-id/pocket-id/backend/internal/auditlogs"
	"github.com/pocket-id/pocket-id/backend/internal/devicelogin"
	"github.com/pocket-id/pocket-id/backend/internal/emailverification"
	"github.com/pocket-id/pocket-id/backend/internal/environment"
	"github.com/pocket-id/pocket-id/backend/internal/ldapsync"
	"github.com/pocket-id/pocket-id/backend/internal/logopreset"
	"github.com/pocket-id/pocket-id/backend/internal/middleware"
	"github.com/pocket-id/pocket-id/backend/internal/oidc"
	"github.com/pocket-id/pocket-id/backend/internal/onetimeaccess"
	"github.com/pocket-id/pocket-id/backend/internal/scimsync"
	"github.com/pocket-id/pocket-id/backend/internal/usersignup"
	"github.com/pocket-id/pocket-id/backend/internal/webauthn"
)

// TestEveryAPIRouteDeclaresItsAccess builds the complete route table and fails when an API route was registered without declaring a scope or public access
func TestEveryAPIRouteDeclaresItsAccess(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Registration only stores handler references and never calls them, so modules without dependencies are enough
	svc := &services{
		apiKeyModule:            &apikey.Module{},
		auditLogsModule:         &auditlogs.Module{},
		deviceLoginModule:       &devicelogin.Module{},
		ldapSyncModule:          &ldapsync.Module{},
		scimSyncModule:          &scimsync.Module{},
		oidcModule:              &oidc.Module{},
		webauthnModule:          &webauthn.Module{},
		userSignUpModule:        &usersignup.Module{},
		oneTimeAccessModule:     &onetimeaccess.Module{},
		emailVerificationModule: &emailverification.Module{},
		apiModule:               &api.Module{},
		environmentModule:       &environment.Module{},
		logoPresetModule:        &logopreset.Module{},
	}

	engine := gin.New()
	auth := middleware.NewAuthorization(nil, nil, nil)
	require.NoError(t, registerRoutes(engine, nil, svc, auth, nil))

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
