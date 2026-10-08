package controller

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"github.com/pocket-id/pocket-id/backend/internal/apikey"
	"github.com/pocket-id/pocket-id/backend/internal/apperror"
	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/instanceid"
	"github.com/pocket-id/pocket-id/backend/internal/middleware"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	"github.com/pocket-id/pocket-id/backend/internal/service"
	testutils "github.com/pocket-id/pocket-id/backend/internal/utils/testing"
)

func TestImageUploadRoutesLimitRequestSize(t *testing.T) {
	gin.SetMode(gin.TestMode)

	originalEnvConfig := common.EnvConfig
	defer func() {
		common.EnvConfig = originalEnvConfig
	}()
	common.EnvConfig.EncryptionKey = []byte("0123456789abcdef0123456789abcdef")

	db := testutils.NewDatabaseForTest(t)

	instanceID, err := instanceid.Load(t.Context(), db)
	require.NoError(t, err)

	jwtService, err := service.NewJwtService(t.Context(), db, instanceID)
	require.NoError(t, err)

	userService := service.NewUserService(db, jwtService, nil, nil, nil, nil, nil)
	apiKeyModule, err := apikey.New(t.Context(), apikey.Dependencies{DB: db, CleanupDisabled: true})
	require.NoError(t, err)

	auth := middleware.NewAuthorization(apiKeyModule, userService, jwtService)
	fileSizeLimitMiddleware := middleware.NewFileSizeLimitMiddleware()

	user := model.User{Username: "upload-admin", IsAdmin: true}
	require.NoError(t, db.Create(&user).Error)

	token, err := jwtService.GenerateAccessToken(user, "", time.Hour)
	require.NoError(t, err)

	router := gin.New()
	router.Use(middleware.NewErrorHandlerMiddleware().Add())
	apiRouter := auth.Router(router.Group("/api"))
	NewUserController(apiRouter, fileSizeLimitMiddleware, nil, userService, nil)
	NewAppImagesController(apiRouter, fileSizeLimitMiddleware, nil)

	routes := []string{
		"/api/users/user-id/profile-picture",
		"/api/users/me/profile-picture",
		"/api/application-images/logo",
		"/api/application-images/email",
		"/api/application-images/background",
		"/api/application-images/favicon",
		"/api/application-images/default-profile-picture",
	}

	for _, route := range routes {
		t.Run("rejects an oversized upload to "+route, func(t *testing.T) {
			status, code := uploadFile(t, router, route, token, 10<<20+1)

			require.Equal(t, http.StatusRequestEntityTooLarge, status)
			require.Equal(t, apperror.CodeFileTooLarge, code)
		})
	}

	t.Run("passes an upload under the limit on to the handler", func(t *testing.T) {
		status, code := uploadFile(t, router, "/api/users/me/profile-picture", token, 9<<20)

		require.Equal(t, http.StatusBadRequest, status)
		require.Equal(t, apperror.CodeInvalidImage, code)
	})
}

func uploadFile(t *testing.T, router *gin.Engine, target string, token string, size int) (int, apperror.Code) {
	t.Helper()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", "upload.png")
	require.NoError(t, err)
	_, err = part.Write(bytes.Repeat([]byte("x"), size))
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	request := httptest.NewRequestWithContext(t.Context(), http.MethodPut, target, &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	request.Header.Set("Authorization", "Bearer "+token)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	var response struct {
		Code apperror.Code `json:"code"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	return recorder.Code, response.Code
}
