package usersignup

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"github.com/pocket-id/pocket-id/backend/internal/apperror"
	"github.com/pocket-id/pocket-id/backend/internal/authz"
)

// usersReadAuthenticator signs every request in with an API key that only holds users:read
type usersReadAuthenticator struct{}

func (usersReadAuthenticator) Kinds() authz.PrincipalKind {
	return authz.KindAPIKey
}

func (usersReadAuthenticator) Present(*gin.Context) bool {
	return true
}

func (usersReadAuthenticator) Authenticate(*gin.Context) (*authz.Principal, error) {
	return &authz.Principal{Kind: authz.KindAPIKey, UserID: "admin", Scopes: authz.ScopeSet{authz.UsersRead: {}}}, nil
}

func TestListingSignupTokensRequiresUsersWrite(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var errs []*gin.Error
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Next()
		errs = c.Errors
	})
	(&Module{}).RegisterRoutes(authz.NewMiddleware(usersReadAuthenticator{}).Router(router.Group("/api")), nil)

	request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/signup-tokens", nil)
	router.ServeHTTP(httptest.NewRecorder(), request)

	require.Len(t, errs, 1)
	appErr, ok := errors.AsType[*apperror.Error](errs[0].Err)
	require.True(t, ok)
	require.Equal(t, string(authz.UsersWrite), appErr.Details()["required_scope"])
}
