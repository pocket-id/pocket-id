package authz

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"github.com/pocket-id/pocket-id/backend/internal/apperror"
)

// fakeAuthenticator accepts any request carrying its header and resolves to the configured outcome
type fakeAuthenticator struct {
	kind   PrincipalKind
	header string
	user   string
	admin  bool
	err    error
	calls  int
}

func (a *fakeAuthenticator) Kind() PrincipalKind {
	return a.kind
}

func (a *fakeAuthenticator) Present(c *gin.Context) bool {
	return c.GetHeader(a.header) != ""
}

func (a *fakeAuthenticator) Authenticate(*gin.Context) (*Principal, error) {
	a.calls++
	if a.err != nil {
		return nil, a.err
	}

	principal := &Principal{Kind: a.kind, UserID: a.user, Scopes: UserScopes(a.admin, a.kind)}
	if a.kind == KindSession {
		principal.AuthenticationMethod = "passkey"
		principal.AuthenticationTime = time.Unix(1700000000, 0)
	}
	return principal, nil
}

type middlewareResult struct {
	status    int
	err       error
	principal Principal
}

// serve runs one request through a route that requires the scope and reports what the middleware decided
func serve(t *testing.T, m *Middleware, scope Scope, optional bool, headers map[string]string) middlewareResult {
	t.Helper()
	gin.SetMode(gin.TestMode)

	var result middlewareResult
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Next()
		if len(c.Errors) > 0 {
			result.err = c.Errors.Last().Err
		}
	})

	r := m.Router(router.Group("/api"))
	if optional {
		r = r.Optional()
	}
	r.GET("/route", scope, func(c *gin.Context) {
		result.principal = PrincipalFrom(c)
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/route", nil)
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	result.status = recorder.Code
	return result
}

func TestMiddlewareAuthorizes(t *testing.T) {
	session := &fakeAuthenticator{kind: KindSession, header: "X-Session", user: "session-user"}
	apiKey := &fakeAuthenticator{kind: KindAPIKey, header: "X-Key", user: "key-user", admin: true}
	m := NewMiddleware(session, apiKey)

	t.Run("attaches the principal", func(t *testing.T) {
		result := serve(t, m, AccountRead, false, map[string]string{"X-Session": "1"})

		require.Equal(t, http.StatusNoContent, result.status)
		require.Equal(t, "session-user", result.principal.UserID)
		require.Equal(t, KindSession, result.principal.Kind)
		require.Equal(t, "passkey", result.principal.AuthenticationMethod)
	})

	t.Run("rejects missing credentials", func(t *testing.T) {
		result := serve(t, m, AccountRead, false, nil)

		require.True(t, apperror.IsCode(result.err, apperror.CodeNotSignedIn))
	})

	t.Run("rejects a principal without the scope and names the scope", func(t *testing.T) {
		result := serve(t, m, UsersRead, false, map[string]string{"X-Session": "1"})

		var appErr *apperror.Error
		require.ErrorAs(t, result.err, &appErr)
		require.Equal(t, apperror.CodeForbidden, appErr.Code())
		require.Equal(t, string(UsersRead), appErr.Details()["required_scope"])
	})

	t.Run("the first authenticator that resolves wins", func(t *testing.T) {
		result := serve(t, m, AccountRead, false, map[string]string{"X-Session": "1", "X-Key": "1"})

		require.Equal(t, "session-user", result.principal.UserID)
	})

	t.Run("rejects a credential kind that can never hold the scope without validating it", func(t *testing.T) {
		calls := apiKey.calls
		result := serve(t, m, AccountSession, false, map[string]string{"X-Key": "1"})

		require.True(t, apperror.IsCode(result.err, apperror.CodeAPIKeyAuthNotAllowed))
		require.Equal(t, calls, apiKey.calls, "the API key must not be validated or marked as used")
	})

	t.Run("a valid credential of an allowed kind wins over a rejected kind", func(t *testing.T) {
		result := serve(t, m, AccountSession, false, map[string]string{"X-Session": "1", "X-Key": "1"})

		require.Equal(t, http.StatusNoContent, result.status)
		require.Equal(t, "session-user", result.principal.UserID)
	})
}

func TestMiddlewareFallsThroughInvalidCredentials(t *testing.T) {
	invalidSession := &fakeAuthenticator{kind: KindSession, header: "X-Session", err: apperror.NotSignedIn()}
	apiKey := &fakeAuthenticator{kind: KindAPIKey, header: "X-Key", user: "key-user"}
	m := NewMiddleware(invalidSession, apiKey)

	result := serve(t, m, AccountRead, false, map[string]string{"X-Session": "1", "X-Key": "1"})

	require.Equal(t, http.StatusNoContent, result.status)
	require.Equal(t, "key-user", result.principal.UserID)
}

func TestMiddlewareStopsOnRejectedCredentials(t *testing.T) {
	disabledSession := &fakeAuthenticator{kind: KindSession, header: "X-Session", err: apperror.UserDisabled()}
	apiKey := &fakeAuthenticator{kind: KindAPIKey, header: "X-Key", user: "key-user"}
	m := NewMiddleware(disabledSession, apiKey)

	for _, optional := range []bool{false, true} {
		result := serve(t, m, AccountRead, optional, map[string]string{"X-Session": "1", "X-Key": "1"})

		require.True(t, apperror.IsCode(result.err, apperror.CodeUserDisabled), "optional=%v", optional)
		require.Zero(t, apiKey.calls)
	}
}

func TestMiddlewareOptional(t *testing.T) {
	session := &fakeAuthenticator{kind: KindSession, header: "X-Session", user: "session-user"}
	invalidSession := &fakeAuthenticator{kind: KindSession, header: "X-Expired", err: apperror.NotSignedIn()}
	apiKey := &fakeAuthenticator{kind: KindAPIKey, header: "X-Key", user: "key-user"}
	m := NewMiddleware(session, invalidSession, apiKey)

	t.Run("continues anonymously without credentials", func(t *testing.T) {
		result := serve(t, m, AccountSession, true, nil)

		require.Equal(t, http.StatusNoContent, result.status)
		require.Equal(t, Principal{}, result.principal)
	})

	t.Run("continues anonymously with an invalid credential", func(t *testing.T) {
		result := serve(t, m, AccountSession, true, map[string]string{"X-Expired": "1"})

		require.Equal(t, http.StatusNoContent, result.status)
		require.Equal(t, Principal{}, result.principal)
	})

	t.Run("ignores a credential kind that can never hold the scope", func(t *testing.T) {
		result := serve(t, m, AccountSession, true, map[string]string{"X-Key": "1"})

		require.Equal(t, http.StatusNoContent, result.status)
		require.Equal(t, Principal{}, result.principal)
		require.Zero(t, apiKey.calls)
	})

	t.Run("attaches the principal when signed in", func(t *testing.T) {
		result := serve(t, m, AccountSession, true, map[string]string{"X-Session": "1"})

		require.Equal(t, "session-user", result.principal.UserID)
	})

	t.Run("still rejects a signed-in principal without the scope", func(t *testing.T) {
		result := serve(t, m, UsersRead, true, map[string]string{"X-Session": "1"})

		require.True(t, apperror.IsCode(result.err, apperror.CodeForbidden))
	})
}

func TestMiddlewarePassesThroughUnexpectedErrors(t *testing.T) {
	failure := errors.New("database unavailable")
	m := NewMiddleware(&fakeAuthenticator{kind: KindSession, header: "X-Session", err: failure})

	result := serve(t, m, AccountRead, false, map[string]string{"X-Session": "1"})

	require.ErrorIs(t, result.err, failure)
}
