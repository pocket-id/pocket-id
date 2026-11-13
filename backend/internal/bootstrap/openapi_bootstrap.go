package bootstrap

import (
	"github.com/gin-gonic/gin"

	"github.com/pocket-id/pocket-id/backend/internal/api"
	"github.com/pocket-id/pocket-id/backend/internal/apikey"
	"github.com/pocket-id/pocket-id/backend/internal/appassets"
	"github.com/pocket-id/pocket-id/backend/internal/auditlogs"
	"github.com/pocket-id/pocket-id/backend/internal/authz"
	"github.com/pocket-id/pocket-id/backend/internal/claimmappingpolicy"
	"github.com/pocket-id/pocket-id/backend/internal/customclaims"
	"github.com/pocket-id/pocket-id/backend/internal/devicelogin"
	"github.com/pocket-id/pocket-id/backend/internal/emailverification"
	"github.com/pocket-id/pocket-id/backend/internal/environment"
	"github.com/pocket-id/pocket-id/backend/internal/feature"
	"github.com/pocket-id/pocket-id/backend/internal/ldapsync"
	"github.com/pocket-id/pocket-id/backend/internal/logopreset"
	"github.com/pocket-id/pocket-id/backend/internal/mcp"
	"github.com/pocket-id/pocket-id/backend/internal/middleware"
	"github.com/pocket-id/pocket-id/backend/internal/oidc"
	"github.com/pocket-id/pocket-id/backend/internal/onetimeaccess"
	"github.com/pocket-id/pocket-id/backend/internal/openapi"
	"github.com/pocket-id/pocket-id/backend/internal/scimsync"
	"github.com/pocket-id/pocket-id/backend/internal/session"
	"github.com/pocket-id/pocket-id/backend/internal/usergroups"
	"github.com/pocket-id/pocket-id/backend/internal/users"
	"github.com/pocket-id/pocket-id/backend/internal/usersignup"
	"github.com/pocket-id/pocket-id/backend/internal/webauthn"
)

// OpenAPISpec returns the OpenAPI spec of the API, built from the route table without a database or any other service
func OpenAPISpec() ([]byte, error) {
	_, auth, err := routeTable()
	if err != nil {
		return nil, err
	}
	return openapi.Spec(auth.Routes())
}

// routeTable registers every route on a new engine
// Registration only stores handler references and never calls them, so modules without dependencies are enough
func routeTable() (*gin.Engine, *authz.Middleware, error) {
	// Gin's debug mode prints every route to stdout, where cmd/openapi writes the spec
	gin.SetMode(gin.TestMode)

	// appConfigService stays nil, which works because its RegisterRoutes never dereferences its receiver
	svc := &services{
		apiKeyModule:             &apikey.Module{},
		appAssetsModule:          &appassets.Module{},
		auditLogsModule:          &auditlogs.Module{},
		deviceLoginModule:        &devicelogin.Module{},
		ldapSyncModule:           &ldapsync.Module{},
		scimSyncModule:           &scimsync.Module{},
		oidcModule:               &oidc.Module{},
		webauthnModule:           &webauthn.Module{},
		userSignUpModule:         &usersignup.Module{},
		oneTimeAccessModule:      &onetimeaccess.Module{},
		sessionModule:            &session.Module{},
		emailVerificationModule:  &emailverification.Module{},
		apiModule:                &api.Module{},
		claimMappingPolicyModule: &claimmappingpolicy.Module{},
		customClaimsModule:       &customclaims.Module{},
		environmentModule:        &environment.Module{},
		logoPresetModule:         &logopreset.Module{},
		featureModule:            &feature.Module{},
		usersModule:              &users.Module{},
		userGroupsModule:         &usergroups.Module{},
		mcpModule:                &mcp.Module{},
	}

	engine := gin.New()
	auth := middleware.NewAuthorization(nil, nil, nil, nil, nil)
	if err := registerRoutes(engine, nil, svc, auth, nil); err != nil {
		return nil, nil, err
	}
	return engine, auth, nil
}
