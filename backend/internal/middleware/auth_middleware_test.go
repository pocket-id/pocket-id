package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/apikey"
	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
	"github.com/pocket-id/pocket-id/backend/internal/service"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
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

	emailService, err := service.NewEmailService(db, appConfigService)
	require.NoError(t, err)
	geoLiteService := service.NewGeoLiteService(nil)
	auditLogService := service.NewAuditLogService(db, appConfigService, emailService, geoLiteService)

	jwtService, err := service.NewJwtService(t.Context(), db, appConfigService)
	require.NoError(t, err)

	userService := service.NewUserService(db, jwtService, nil, nil, appConfigService, nil, nil, nil, nil)
	apiKeyModule, err := apikey.New(t.Context(), apikey.Dependencies{DB: db})
	require.NoError(t, err)

	authMiddleware := NewAuthMiddleware(apiKeyModule, userService, jwtService, auditLogService)

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
