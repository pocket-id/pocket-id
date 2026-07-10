package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/apikey"
	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
	"github.com/pocket-id/pocket-id/backend/internal/service"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
	httpapi "github.com/pocket-id/pocket-id/backend/internal/utils/huma"
	testutils "github.com/pocket-id/pocket-id/backend/internal/utils/testing"
)

func TestWithApiKeyAuthDisabled(t *testing.T) {
	gin.SetMode(gin.TestMode)

	originalEnvConfig := common.EnvConfig
	defer func() {
		common.EnvConfig = originalEnvConfig
	}()
	common.EnvConfig.AppURL = "https://test.example.com"
	common.EnvConfig.EncryptionKey = []byte("0123456789abcdef0123456789abcdef")

	db := testutils.NewDatabaseForTest(t)

	appConfigService, err := service.NewAppConfigService(t.Context(), db)
	require.NoError(t, err)

	jwtService, err := service.NewJwtService(t.Context(), db, appConfigService)
	require.NoError(t, err)

	userService := service.NewUserService(db, jwtService, nil, nil, appConfigService, nil, nil, nil, nil)
	apiKeyModule, err := apikey.New(t.Context(), apikey.Dependencies{DB: db})
	require.NoError(t, err)

	authMiddleware := NewAuthMiddleware(apiKeyModule, userService, jwtService)

	user := createUserForAuthMiddlewareTest(t, db)
	jwtToken, err := jwtService.GenerateAccessToken(user, "")
	require.NoError(t, err)

	apiKeyToken := "middleware-test-api-key-raw-token"
	apiKeyRecord := apikey.ApiKey{
		Name:      "Middleware API Key",
		Key:       utils.CreateSha256Hash(apiKeyToken),
		UserID:    user.ID,
		ExpiresAt: datatype.DateTime(time.Now().Add(24 * time.Hour)),
	}
	require.NoError(t, db.Create(&apiKeyRecord).Error)

	router := gin.New()
	router.Use(NewErrorHandlerMiddleware().Add())
	router.GET("/api/protected", authMiddleware.WithAdminNotRequired().WithApiKeyAuthDisabled().Add(), func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	t.Run("rejects API key auth when API key auth is disabled", func(t *testing.T) {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/protected", nil)
		req.Header.Set("X-API-Key", apiKeyToken)
		recorder := httptest.NewRecorder()

		router.ServeHTTP(recorder, req)

		require.Equal(t, http.StatusForbidden, recorder.Code)

		var body map[string]string
		err := json.Unmarshal(recorder.Body.Bytes(), &body)
		require.NoError(t, err)
		require.Equal(t, "API key authentication is not allowed for this endpoint", body["error"])
	})

	t.Run("allows JWT auth when API key auth is disabled", func(t *testing.T) {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/protected", nil)
		req.Header.Set("Authorization", "Bearer "+jwtToken)
		recorder := httptest.NewRecorder()

		router.ServeHTTP(recorder, req)

		require.Equal(t, http.StatusNoContent, recorder.Code)
	})

	t.Run("Huma decorator preserves JWT-only behavior and documentation", func(t *testing.T) {
		humaRouter := gin.New()
		api := httpapi.New(humaRouter, humaRouter.Group("/"))
		operation := huma.Operation{OperationID: "huma-protected", Method: http.MethodGet, Path: "/api/huma-protected"}
		authMiddleware.WithAdminNotRequired().WithApiKeyAuthDisabled().Huma(api)(&operation)
		require.Equal(t, []map[string][]string{{"BearerAuth": {}}, {"SessionCookie": {}}}, operation.Security)
		httpapi.Register(api, operation, func(context.Context, *struct{}) (*struct{}, error) { return &struct{}{}, nil })

		request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/huma-protected", nil)
		request.Header.Set("X-API-Key", apiKeyToken)
		response := httptest.NewRecorder()
		humaRouter.ServeHTTP(response, request)
		require.Equal(t, http.StatusForbidden, response.Code)
		require.JSONEq(t, `{"error":"API key authentication is not allowed for this endpoint"}`, response.Body.String())

		request = httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/huma-protected", nil)
		request.Header.Set("Authorization", "Bearer "+jwtToken)
		response = httptest.NewRecorder()
		humaRouter.ServeHTTP(response, request)
		require.Equal(t, http.StatusNoContent, response.Code)
	})

	t.Run("Huma admin decorator records admin authorization separately from security scopes", func(t *testing.T) {
		humaRouter := gin.New()
		api := httpapi.New(humaRouter, humaRouter.Group("/"))
		operation := huma.Operation{}
		authMiddleware.Huma(api)(&operation)
		require.Equal(t, true, operation.Extensions["x-pocket-id-admin-required"])
		for _, requirement := range operation.Security {
			for _, scopes := range requirement {
				require.Empty(t, scopes)
			}
		}
	})
}

func createUserForAuthMiddlewareTest(t *testing.T, db *gorm.DB) model.User {
	t.Helper()

	user := model.User{
		Username:    "auth-user",
		Email:       new("auth@example.com"),
		FirstName:   "Auth",
		LastName:    "User",
		DisplayName: "Auth User",
	}

	err := db.Create(&user).Error
	require.NoError(t, err)

	return user
}
