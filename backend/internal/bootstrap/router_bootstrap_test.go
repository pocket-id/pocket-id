package bootstrap

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/pocket-id/pocket-id/backend/internal/apperror"
	"github.com/pocket-id/pocket-id/backend/internal/middleware"
	"github.com/stretchr/testify/require"
)

func TestRequestLoggerUsesStructuredErrorMetadata(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var output bytes.Buffer
	previousLogger := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&output, nil)))
	t.Cleanup(func() {
		slog.SetDefault(previousLogger)
	})

	router := gin.New()
	initLogger(router)
	router.Use(middleware.NewErrorHandlerMiddleware().Add())
	router.GET("/api/users/me", func(c *gin.Context) {
		_ = c.Error(apperror.NotSignedIn())
		c.Abort()
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/users/me", nil)
	router.ServeHTTP(recorder, request)

	var body struct {
		RequestID string `json:"request_id"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
	require.NotEmpty(t, body.RequestID)

	logLine := output.String()
	require.Contains(t, logLine, "level=INFO")
	require.Contains(t, logLine, `msg="HTTP request completed"`)
	require.Contains(t, logLine, "request_id="+body.RequestID)
	require.Contains(t, logLine, "error_code=not_signed_in")
	require.Contains(t, logLine, "status=401")
	require.NotContains(t, logLine, "Request with errors")
	require.NotContains(t, logLine, "Error #01")
}

func TestRequestLoggerKeepsRateLimitsAtWarningLevel(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var output bytes.Buffer
	previousLogger := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&output, nil)))
	t.Cleanup(func() {
		slog.SetDefault(previousLogger)
	})

	router := gin.New()
	initLogger(router)
	router.Use(middleware.NewErrorHandlerMiddleware().Add())
	router.GET("/api/limited", func(c *gin.Context) {
		_ = c.Error(apperror.TooManyRequests())
		c.Abort()
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/limited", nil)
	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusTooManyRequests, recorder.Code)
	require.Contains(t, output.String(), "level=WARN")
	require.Contains(t, output.String(), "error_code=rate_limited")
}

func TestRequestLoggerRespectsConfiguredMinimumLevel(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var output bytes.Buffer
	previousLogger := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&output, &slog.HandlerOptions{
		Level: slog.LevelWarn,
	})))
	t.Cleanup(func() {
		slog.SetDefault(previousLogger)
	})

	router := gin.New()
	initLogger(router)
	router.GET("/api/status", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/status", nil)
	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusNoContent, recorder.Code)
	require.Empty(t, output.String())
}

func TestRequestLoggerLogsAtConfiguredMinimumLevel(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var output bytes.Buffer
	previousLogger := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&output, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))
	t.Cleanup(func() {
		slog.SetDefault(previousLogger)
	})

	router := gin.New()
	initLogger(router)
	router.GET("/api/status", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/status", nil)
	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusNoContent, recorder.Code)
	require.Contains(t, output.String(), "level=INFO")
}
