package middleware

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/stretchr/testify/require"
)

type mockSlogHandler struct {
	lastEvent string
}

func (h *mockSlogHandler) Enabled(context.Context, slog.Level) bool  { return true }
func (h *mockSlogHandler) Handle(_ context.Context, r slog.Record) error {
	if r.Message == "Security event" {
		h.lastEvent = r.Message
	}
	return nil
}
func (h *mockSlogHandler) WithAttrs(attrs []slog.Attr) slog.Handler { return h }
func (h *mockSlogHandler) WithGroup(name string) slog.Handler      { return h }

func TestErrorHandlerMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockHandler := &mockSlogHandler{}
	logger := slog.New(mockHandler)
	slog.SetDefault(logger)

	router := gin.New()
	router.Use(NewErrorHandlerMiddleware().Add())

	router.GET("/401", func(c *gin.Context) {
		_ = c.Error(&common.TokenInvalidError{})
	})

	router.GET("/403", func(c *gin.Context) {
		_ = c.Error(&common.MissingPermissionError{})
	})

	router.GET("/404", func(c *gin.Context) {
		_ = c.Error(&common.UserNotFoundError{})
	})

	t.Run("logs security event for 401", func(t *testing.T) {
		mockHandler.lastEvent = ""
		req := httptest.NewRequest(http.MethodGet, "/401", nil)
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, req)

		require.Equal(t, http.StatusUnauthorized, recorder.Code)
		require.Equal(t, "Security event", mockHandler.lastEvent)

		var body map[string]string
		err := json.Unmarshal(recorder.Body.Bytes(), &body)
		require.NoError(t, err)
		require.Equal(t, "Token is invalid", body["error"])
	})

	t.Run("logs security event for 403", func(t *testing.T) {
		mockHandler.lastEvent = ""
		req := httptest.NewRequest(http.MethodGet, "/403", nil)
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, req)

		require.Equal(t, http.StatusForbidden, recorder.Code)
		require.Equal(t, "Security event", mockHandler.lastEvent)
	})

	t.Run("does not log security event for 404", func(t *testing.T) {
		mockHandler.lastEvent = ""
		req := httptest.NewRequest(http.MethodGet, "/404", nil)
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, req)

		require.Equal(t, http.StatusNotFound, recorder.Code)
		require.Equal(t, "", mockHandler.lastEvent)
	})
}
