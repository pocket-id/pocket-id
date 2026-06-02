package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/service"
	testutils "github.com/pocket-id/pocket-id/backend/internal/utils/testing"
	"github.com/stretchr/testify/require"
)

type mockSlogHandler struct {
	lastEvent string
}

func (h *mockSlogHandler) Enabled(_ context.Context, _ slog.Level) bool { return true }
func (h *mockSlogHandler) Handle(_ context.Context, r slog.Record) error {
	if r.Message == "Security event" {
		h.lastEvent = r.Message
	}
	return nil
}
func (h *mockSlogHandler) WithAttrs(attrs []slog.Attr) slog.Handler { return h }
func (h *mockSlogHandler) WithGroup(name string) slog.Handler       { return h }

func TestErrorHandlerMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockHandler := &mockSlogHandler{}
	slog.SetDefault(slog.New(mockHandler))
	router := gin.New()
	router.Use(NewErrorHandlerMiddleware().Add())

	// Setup database and services
	db := testutils.NewDatabaseForTest(t)
	auditLogService := service.NewAuditLogService(db, nil, nil, nil)
	oneTimeAccessService := service.NewOneTimeAccessService(db, nil, nil, auditLogService, nil, nil)
	appConfigService, _ := service.NewAppConfigService(context.Background(), db)
	webAuthnService, _ := service.NewWebAuthnService(db, nil, auditLogService, appConfigService)

	router.GET("/one_time_access_invalid_token", func(c *gin.Context) {
		_, _, err := oneTimeAccessService.ExchangeOneTimeAccessToken(c.Request.Context(), "invalid-token", "", "127.0.0.1", "test-agent")
		_ = c.Error(err)
	})

	router.GET("/webauthn_protocol_error", func(c *gin.Context) {
		credentialAssertionData, err := protocol.ParseCredentialRequestResponseBody(c.Request.Body)
		_, _, err = webAuthnService.VerifyLogin(c.Request.Context(), "sessionId", credentialAssertionData, "127.0.0.1", "test-agent")
		_ = c.Error(err)
	})

	router.GET("/404", func(c *gin.Context) {
		_ = c.Error(&common.UserNotFoundError{})
	})

	t.Run("logs security event for OneTimeAccess invalid token", func(t *testing.T) {
		// given
		mockHandler.lastEvent = ""
		request := httptest.NewRequest(http.MethodGet, "/one_time_access_invalid_token", nil)
		recorder := httptest.NewRecorder()
		var initialSignInFailureCount int64
		db.Table("audit_logs").Select("count(Event)").Count(&initialSignInFailureCount)

		// when
		router.ServeHTTP(recorder, request)

		// then
		var body map[string]string
		err := json.Unmarshal(recorder.Body.Bytes(), &body)
		require.NoError(t, err)
		require.Equal(t, "Token is invalid or expired", body["error"])
		require.Equal(t, http.StatusUnauthorized, recorder.Code)
		require.Equal(t, "Security event", mockHandler.lastEvent)

		var signInFailureCount int64
		db.Table("audit_logs").Count(&signInFailureCount)
		require.Equal(t, initialSignInFailureCount+1, signInFailureCount)
	})

	t.Run("logs security event for Webauthn protocol error", func(t *testing.T) {
		// given
		mockHandler.lastEvent = ""
		jsonBody := []byte(`{
		  "id": "cmFuZG9tLWlk",
		  "rawId": "cmFuZG9tLWlk",
		  "type": "public-key",
		  "response": {
		    "authenticatorData": "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",
		    "clientDataJSON": "eyJ0eXBlIjogIndlYmF1dGhuLmdldCIsICJjaGFsbGVuZ2UiOiAiY2hhbGxlbmdlIiwgIm9yaWdpbiI6ICJodHRwczovL2V4YW1wbGUuY29tIn0",
		    "signature": "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"
		  }
		}`)
		request := httptest.NewRequest(http.MethodGet, "/webauthn_protocol_error", bytes.NewBuffer(jsonBody))
		recorder := httptest.NewRecorder()
		var initialSignInFailureCount int64
		db.Table("audit_logs").Select("count(Event)").Count(&initialSignInFailureCount)

		// when
		router.ServeHTTP(recorder, request)

		// then
		var body map[string]string
		err := json.Unmarshal(recorder.Body.Bytes(), &body)
		require.NoError(t, err)
		require.Equal(t, "Something went wrong. Please try again later", body["error"])
		require.Equal(t, http.StatusBadRequest, recorder.Code)
		require.Equal(t, "Security event", mockHandler.lastEvent)

		var signInFailureCount int64
		db.Table("audit_logs").Count(&signInFailureCount)
		require.Equal(t, initialSignInFailureCount+1, signInFailureCount)
	})

	t.Run("does not log security event for 404", func(t *testing.T) {
		// given
		mockHandler.lastEvent = ""
		request := httptest.NewRequest(http.MethodGet, "/404", nil)
		recorder := httptest.NewRecorder()
		var initialSignInFailureCount int64
		db.Table("audit_logs").Select("count(Event)").Count(&initialSignInFailureCount)

		// when
		router.ServeHTTP(recorder, request)

		// then
		var body map[string]string
		err := json.Unmarshal(recorder.Body.Bytes(), &body)
		require.NoError(t, err)
		require.Equal(t, "User not found", body["error"])
		require.Equal(t, http.StatusNotFound, recorder.Code)
		require.Equal(t, "", mockHandler.lastEvent)

		var signInFailureCount int64
		db.Table("audit_logs").Count(&signInFailureCount)
		require.Equal(t, initialSignInFailureCount, signInFailureCount)
	})
}
