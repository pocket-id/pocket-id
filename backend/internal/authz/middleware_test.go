package authz

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"github.com/pocket-id/pocket-id/backend/internal/apperror"
	"github.com/pocket-id/pocket-id/backend/internal/httpserver"
)

// fakeAuthenticator accepts any request carrying its header and resolves to the configured outcome
type fakeAuthenticator struct {
	kind       PrincipalKind
	header     string
	user       string
	admin      bool
	unverified bool
	sudoUntil  time.Time
	err        error
	calls      int
}

func (a *fakeAuthenticator) Kinds() PrincipalKind {
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
	if a.unverified {
		principal.RestrictUnverified()
	}
	if a.kind == KindSession {
		principal.AuthenticationMethod = "passkey"
		principal.AuthenticationTime = time.Unix(1700000000, 0)
		principal.SudoUntil = a.sudoUntil
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

	// Routes with an admin scope must live below the admin prefix
	base := "/api"
	if definitions[scope].category == CategoryAdmin {
		base = "/api/admin"
	}
	r := m.Router(router.Group(base))
	if optional {
		r = r.Optional()
	}
	r.GET("/route", scope, httpserver.Endpoint{Handler: func(c *gin.Context) {
		result.principal = PrincipalFrom(c)
		c.Status(http.StatusNoContent)
	}})

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, base+"/route", nil)
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
	m := NewMiddleware(nil, session, apiKey)

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

func TestMiddlewareExplainsScopesWithheldUntilVerified(t *testing.T) {
	session := &fakeAuthenticator{kind: KindSession, header: "X-Session", user: "session-user", unverified: true}
	m := NewMiddleware(nil, session)

	t.Run("allows a scope needed to verify the email", func(t *testing.T) {
		result := serve(t, m, AccountWrite, false, map[string]string{"X-Session": "1"})

		require.Equal(t, http.StatusNoContent, result.status)
	})

	t.Run("asks to verify the email for a withheld scope, even on optional routes", func(t *testing.T) {
		for _, optional := range []bool{false, true} {
			result := serve(t, m, AccountAuthorize, optional, map[string]string{"X-Session": "1"})

			require.True(t, apperror.IsCode(result.err, apperror.CodeEmailVerificationRequired), "optional=%v", optional)
		}
	})

	t.Run("reports a scope the role never grants as missing", func(t *testing.T) {
		result := serve(t, m, UsersRead, false, map[string]string{"X-Session": "1"})

		require.True(t, apperror.IsCode(result.err, apperror.CodeForbidden))
	})
}

func TestMiddlewareFallsThroughInvalidCredentials(t *testing.T) {
	invalidSession := &fakeAuthenticator{kind: KindSession, header: "X-Session", err: apperror.NotSignedIn()}
	apiKey := &fakeAuthenticator{kind: KindAPIKey, header: "X-Key", user: "key-user"}
	m := NewMiddleware(nil, invalidSession, apiKey)

	result := serve(t, m, AccountRead, false, map[string]string{"X-Session": "1", "X-Key": "1"})

	require.Equal(t, http.StatusNoContent, result.status)
	require.Equal(t, "key-user", result.principal.UserID)
}

func TestMiddlewareStopsOnRejectedCredentials(t *testing.T) {
	disabledSession := &fakeAuthenticator{kind: KindSession, header: "X-Session", err: apperror.UserDisabled()}
	apiKey := &fakeAuthenticator{kind: KindAPIKey, header: "X-Key", user: "key-user"}
	m := NewMiddleware(nil, disabledSession, apiKey)

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
	m := NewMiddleware(nil, session, invalidSession, apiKey)

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
	m := NewMiddleware(nil, &fakeAuthenticator{kind: KindSession, header: "X-Session", err: failure})

	result := serve(t, m, AccountRead, false, map[string]string{"X-Session": "1"})

	require.ErrorIs(t, result.err, failure)
}

func TestDispatchRouteRequiresAnAdminScope(t *testing.T) {
	session := &fakeAuthenticator{kind: KindSession, header: "X-Session", user: "session-user", admin: true}
	apiKey := &fakeAuthenticator{kind: KindAPIKey, header: "X-Key", user: "key-user"}
	adminAPIKey := &fakeAuthenticator{kind: KindAPIKey, header: "X-Admin-Key", user: "admin-key-user", admin: true}
	m := NewMiddleware(nil, session, apiKey, adminAPIKey)

	serveDispatch := func(t *testing.T, headers map[string]string) middlewareResult {
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
		m.Router(router.Group("/api/admin")).Dispatch(http.MethodPost, "/mcp", func(c *gin.Context) {
			result.principal = PrincipalFrom(c)
			c.Status(http.StatusNoContent)
		})

		req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/admin/mcp", nil)
		for key, value := range headers {
			req.Header.Set(key, value)
		}
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, req)
		result.status = recorder.Code
		return result
	}

	t.Run("accepts an API key that holds admin scopes", func(t *testing.T) {
		result := serveDispatch(t, map[string]string{"X-Admin-Key": "1"})

		require.Equal(t, http.StatusNoContent, result.status)
		require.Equal(t, "admin-key-user", result.principal.UserID)
	})

	t.Run("rejects an API key that holds only account scopes", func(t *testing.T) {
		result := serveDispatch(t, map[string]string{"X-Key": "1"})

		require.True(t, apperror.IsCode(result.err, apperror.CodeForbidden))
	})

	t.Run("rejects a browser session", func(t *testing.T) {
		result := serveDispatch(t, map[string]string{"X-Session": "1"})

		require.True(t, apperror.IsCode(result.err, apperror.CodeNotSignedIn))
		require.Zero(t, session.calls, "the session must not be validated")
	})

	t.Run("rejects missing credentials", func(t *testing.T) {
		result := serveDispatch(t, nil)

		require.True(t, apperror.IsCode(result.err, apperror.CodeNotSignedIn))
	})
}

func TestRequireSudo(t *testing.T) {
	admin := NewScopeSet(AdminAccessWrite)
	inSudo := time.Now().Add(time.Minute)

	tests := []struct {
		name      string
		principal Principal
		wantCode  apperror.Code
	}{
		{name: "an API key passes without sudo mode", principal: Principal{Kind: KindAPIKey, Scopes: admin}},
		{name: "a session in sudo mode passes", principal: Principal{Kind: KindSession, Scopes: admin, SudoUntil: inSudo}},
		{name: "a session outside of sudo mode has to prove its passkey", principal: Principal{Kind: KindSession, Scopes: admin}, wantCode: apperror.CodeSudoRequired},
		{name: "sudo mode doesn't stand in for the scope", principal: Principal{Kind: KindSession, SudoUntil: inSudo}, wantCode: apperror.CodeForbidden},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.principal.RequireAdminAccess()
			if tt.wantCode == "" {
				require.NoError(t, err)
				return
			}
			require.True(t, apperror.IsCode(err, tt.wantCode), err)
		})
	}
}

// sudoRecorder records the sessions whose sudo mode the middleware kept alive
type sudoRecorder []string

func (r *sudoRecorder) KeepSudo(_ context.Context, userID, _ string) {
	*r = append(*r, userID)
}

func TestMiddlewareKeepsUsedSudo(t *testing.T) {
	gin.SetMode(gin.TestMode)

	kept := &sudoRecorder{}
	session := &fakeAuthenticator{kind: KindSession, header: "X-Session", user: "session-user", sudoUntil: time.Now().Add(time.Minute)}
	router := gin.New()
	r := NewMiddleware(kept, session).Router(router.Group("/api"))
	r.GET("/plain", AccountRead, httpserver.Endpoint{Handler: func(*gin.Context) {}})
	r.GET("/sudo", AccountRead, httpserver.Endpoint{Handler: func(c *gin.Context) {
		// Services check a copy of the principal, which must still reach the middleware
		principal := PrincipalFrom(c)
		require.NoError(t, principal.RequireSudo())
	}})

	for _, path := range []string{"/api/plain", "/api/sudo"} {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, path, nil)
		req.Header.Set("X-Session", "token")
		router.ServeHTTP(httptest.NewRecorder(), req)
	}

	// Only the request that relied on sudo mode keeps it alive
	require.Equal(t, &sudoRecorder{"session-user"}, kept)
}
