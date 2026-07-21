package webauthn

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"github.com/pocket-id/pocket-id/backend/internal/model"
	"github.com/pocket-id/pocket-id/backend/internal/utils/cookie"
	httpapi "github.com/pocket-id/pocket-id/backend/internal/utils/huma"
	testutils "github.com/pocket-id/pocket-id/backend/internal/utils/testing"
)

func TestRequestWithBodyReconstructsUnderlyingRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	api := httpapi.New(router, router.Group("/"))
	type input struct {
		Body map[string]string
	}
	type output struct {
		Body map[string]string
	}
	httpapi.Register(api, huma.Operation{
		OperationID: "reconstruct-request",
		Method:      http.MethodPost,
		Path:        "/api/reconstruct",
	}, func(ctx context.Context, _ *input) (*output, error) {
		request := requestWithBody(ctx, []byte(`{"credential":"value"}`))
		body, err := io.ReadAll(request.Body)
		require.NoError(t, err)
		require.True(t, bytes.Equal([]byte(`{"credential":"value"}`), body))
		require.Equal(t, int64(len(body)), request.ContentLength)
		return &output{Body: map[string]string{"status": "ok"}}, nil
	})

	request := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/reconstruct", http.NoBody)
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	require.Equal(t, http.StatusBadRequest, response.Code)

	request = httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/reconstruct", strings.NewReader(`{"input":"present"}`))
	request.Header.Set("Content-Type", "application/json")
	response = httptest.NewRecorder()
	router.ServeHTTP(response, request)
	require.Equal(t, http.StatusOK, response.Code)
}

func TestLogoutClearsAccessTokenCookie(t *testing.T) {
	output, err := (&handler{}).logout(t.Context(), &httpapi.EmptyInput{})
	require.NoError(t, err)
	require.Len(t, output.SetCookie, 1)
	require.Equal(t, -1, output.SetCookie[0].MaxAge)
	require.Contains(t, output.SetCookie[0].String(), "Max-Age=0")
}

func TestReauthenticateFallsBackWithoutSessionCookie(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	user := model.User{Base: model.Base{ID: "handler-reauth-user"}, Username: "handler-reauth-user"}
	require.NoError(t, db.Create(&user).Error)

	signer := newFakeSigner()
	accessToken, err := signer.GenerateAccessToken(user, authenticationMethodPhishingResistant, time.Hour)
	require.NoError(t, err)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	api := httpapi.New(router, router.Group("/"))
	h := &handler{service: &Service{db: db, signer: signer}}
	httpapi.Register(api, huma.Operation{
		OperationID:   "test-reauthenticate-fallback",
		Method:        http.MethodPost,
		Path:          "/api/test-reauthenticate-fallback",
		DefaultStatus: http.StatusNoContent,
	}, h.reauthenticate)

	testCases := []struct {
		name string
		body io.Reader
	}{
		{name: "empty body"},
		{name: "invalid assertion", body: strings.NewReader(`{"invalid":true}`)},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			request := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/test-reauthenticate-fallback", testCase.body)
			if testCase.body != nil {
				request.Header.Set("Content-Type", "application/json")
			}
			request.AddCookie(cookie.NewAccessTokenCookie(60, accessToken))
			response := httptest.NewRecorder()

			router.ServeHTTP(response, request)

			require.Equal(t, http.StatusNoContent, response.Code)
			require.Contains(t, response.Header().Get("Set-Cookie"), cookie.ReauthenticationTokenCookieName+"=")
		})
	}
}
