package bootstrap

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/job"
	"github.com/pocket-id/pocket-id/backend/internal/storage"
	testutils "github.com/pocket-id/pocket-id/backend/internal/utils/testing"
)

func TestHumaRouterOpenAPI(t *testing.T) {
	originalConfig := common.EnvConfig
	t.Cleanup(func() { common.EnvConfig = originalConfig })
	common.EnvConfig.AppEnv = common.AppEnvTest
	common.EnvConfig.AppURL = "https://test.example.com"
	common.EnvConfig.InternalAppURL = "https://test.example.com"
	common.EnvConfig.EncryptionKey = []byte("0123456789abcdef0123456789abcdef")
	common.EnvConfig.DisableRateLimiting = true

	db := testutils.NewDatabaseForTest(t)
	fileStorage, err := storage.NewDatabaseStorage(db)
	require.NoError(t, err)
	scheduler, err := job.NewScheduler()
	require.NoError(t, err)
	services, err := initServices(t.Context(), db, "test-instance", http.DefaultClient, map[string]string{}, fileStorage, scheduler)
	require.NoError(t, err)
	router, err := initEngine()
	require.NoError(t, err)
	require.NoError(t, registerRoutes(router, db, services, nil))

	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/openai.json", nil))
	require.Equal(t, http.StatusOK, response.Code)

	var document struct {
		Paths map[string]map[string]struct {
			OperationID string         `json:"operationId"`
			Responses   map[string]any `json:"responses"`
		} `json:"paths"`
		Components struct {
			SecuritySchemes map[string]any `json:"securitySchemes"`
		} `json:"components"`
	}
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &document))

	for _, path := range []string{
		"/authorize",
		"/.well-known/openid-configuration",
		"/.well-known/jwks.json",
		"/api/users",
		"/api/user-groups",
		"/api/oidc/token",
		"/api/oidc/interactions/{id}",
		"/api/webauthn/reauthenticate",
		"/api/signup",
		"/api/api-keys",
		"/api/apis",
		"/healthz",
	} {
		require.Contains(t, document.Paths, path)
	}

	operationIDs := map[string]struct{}{}
	for _, methods := range document.Paths {
		for _, operation := range methods {
			require.NotEmpty(t, operation.OperationID)
			require.NotContains(t, operation.Responses, "422")
			_, duplicate := operationIDs[operation.OperationID]
			require.False(t, duplicate, "duplicate operation ID %q", operation.OperationID)
			operationIDs[operation.OperationID] = struct{}{}
		}
	}
	for _, scheme := range []string{"BearerAuth", "SessionCookie", "ApiKeyAuth", "OIDCAccessToken", "OIDCClientBasic"} {
		require.Contains(t, document.Components.SecuritySchemes, scheme)
	}
	require.NotContains(t, response.Body.String(), `"$schema"`)
	require.NotContains(t, response.Header().Get("Link"), "schema")

	response = httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/users", nil))
	require.Equal(t, http.StatusUnauthorized, response.Code)
	require.Equal(t, "application/json; charset=utf-8", response.Header().Get("Content-Type"))
	require.JSONEq(t, `{"error":"You are not signed in"}`, response.Body.String())

	response = httptest.NewRecorder()
	request := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/signup/setup", nil)
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(response, request)
	require.Equal(t, http.StatusBadRequest, response.Code)
	require.JSONEq(t, `{"error":"Request body is required"}`, response.Body.String())

	response = httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/docs", nil))
	require.Equal(t, http.StatusNotFound, response.Code)

	response = httptest.NewRecorder()
	newHTTPServer(router, nil).Handler.ServeHTTP(response, httptest.NewRequestWithContext(t.Context(), http.MethodHead, "/healthz", nil))
	require.Equal(t, http.StatusNoContent, response.Code)
	require.Empty(t, response.Body.String())
}
