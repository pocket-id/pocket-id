package bootstrap

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"github.com/pocket-id/pocket-id/backend/internal/apikey"
	"github.com/pocket-id/pocket-id/backend/internal/appassets"
	"github.com/pocket-id/pocket-id/backend/internal/appconfig"
	"github.com/pocket-id/pocket-id/backend/internal/apperror"
	"github.com/pocket-id/pocket-id/backend/internal/middleware"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	"github.com/pocket-id/pocket-id/backend/internal/session"
	"github.com/pocket-id/pocket-id/backend/internal/users"
	"github.com/pocket-id/pocket-id/backend/internal/utils/cookie"
	testutils "github.com/pocket-id/pocket-id/backend/internal/utils/testing"
)

func TestImageUploadRoutesLimitRequestSize(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db := testutils.NewDatabaseForTest(t)

	usersModule := users.New(users.Dependencies{DB: db})
	apiKeyModule, err := apikey.New(apikey.Dependencies{DB: db, CleanupDisabled: true})
	require.NoError(t, err)

	user := model.User{Username: "upload-admin", IsAdmin: true}
	require.NoError(t, db.Create(&user).Error)

	auth := middleware.NewAuthorization(apiKeyModule, nil, usersModule, staticSessionValidator{userID: user.ID}, appconfig.StaticResolver{})
	fileSizeLimitMiddleware := middleware.NewFileSizeLimitMiddleware()

	router := gin.New()
	router.Use(middleware.NewErrorHandlerMiddleware().Add())
	apiRouter := auth.Router(router.Group("/api"))
	usersModule.RegisterRoutes(apiRouter, fileSizeLimitMiddleware.Add(10<<20))
	// The size limit rejects the oversized uploads before any handler runs, so the asset routes need no initialized module
	(&appassets.Module{}).RegisterRoutes(apiRouter, fileSizeLimitMiddleware.Add(10<<20))

	routes := []string{
		"/api/users/user-id/profile-picture",
		"/api/users/me/profile-picture",
		"/api/application-assets/logo",
		"/api/application-assets/email",
		"/api/application-assets/background",
		"/api/application-assets/favicon",
		"/api/application-assets/default-profile-picture",
	}

	for _, route := range routes {
		t.Run("rejects an oversized upload to "+route, func(t *testing.T) {
			status, code := uploadFile(t, router, route, 10<<20+1)

			require.Equal(t, http.StatusRequestEntityTooLarge, status)
			require.Equal(t, apperror.CodeFileTooLarge, code)
		})
	}

	t.Run("passes an upload under the limit on to the handler", func(t *testing.T) {
		status, code := uploadFile(t, router, "/api/users/me/profile-picture", 9<<20)

		require.Equal(t, http.StatusBadRequest, status)
		require.Equal(t, apperror.CodeInvalidImage, code)
	})
}

func uploadFile(t *testing.T, router *gin.Engine, target string, size int) (int, apperror.Code) {
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
	request.AddCookie(&http.Cookie{Name: cookie.SessionTokenCookieName, Value: "session-token", Secure: true, HttpOnly: true, SameSite: http.SameSiteLaxMode})
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	var response struct {
		Code apperror.Code `json:"code"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	return recorder.Code, response.Code
}

// staticSessionValidator accepts any session token as a session of one user
type staticSessionValidator struct {
	userID string
}

func (v staticSessionValidator) Validate(context.Context, string) (string, session.Session, error) {
	return v.userID, session.Session{ID: "session-id"}, nil
}
