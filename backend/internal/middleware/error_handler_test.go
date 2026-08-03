package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pocket-id/pocket-id/backend/internal/apperror"
	"github.com/pocket-id/pocket-id/backend/internal/dto"
	"github.com/pocket-id/pocket-id/backend/internal/httpserver"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestErrorHandlerMiddlewareStructuredError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(NewErrorHandlerMiddleware().Add())
	var handlerRequestID string
	router.GET("/users/1", httpserver.Handle(func(c *gin.Context) error {
		handlerRequestID = RequestID(c)
		cause := errors.New("database connection details")
		return apperror.Wrap(cause, apperror.CodeUserNotFound, http.StatusNotFound, "User not found")
	}))

	recorder := httptest.NewRecorder()
	request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/users/1", nil)
	router.ServeHTTP(recorder, request)

	var body struct {
		Error     string            `json:"error"`
		Code      apperror.Code     `json:"code"`
		Details   map[string]string `json:"details"`
		RequestID string            `json:"request_id"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
	require.Equal(t, http.StatusNotFound, recorder.Code)
	require.Equal(t, "User not found", body.Error)
	require.Equal(t, apperror.CodeUserNotFound, body.Code)
	require.Empty(t, body.Details)
	require.NotEmpty(t, body.RequestID)
	require.Equal(t, body.RequestID, handlerRequestID)
	require.Equal(t, body.RequestID, recorder.Header().Get(requestIDHeader))
	require.NotContains(t, recorder.Body.String(), "database connection details")
}

func TestErrorHandlerMiddlewareHidesUnexpectedError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(NewErrorHandlerMiddleware().Add())
	router.GET("/failure", func(c *gin.Context) {
		_ = c.Error(errors.New("private database details"))
		c.Abort()
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/failure", nil)
	router.ServeHTTP(recorder, request)

	var body map[string]any
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
	require.Equal(t, http.StatusInternalServerError, recorder.Code)
	require.Equal(t, "Something went wrong", body["error"])
	require.Equal(t, string(apperror.CodeInternal), body["code"])
	require.NotContains(t, recorder.Body.String(), "private database details")
	require.NotEmpty(t, body["request_id"])
}

func TestErrorHandlerMiddlewareHidesRecoveredPanic(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(NewErrorHandlerMiddleware().Add())
	router.GET("/panic", httpserver.Handle(func(*gin.Context) error {
		panic("private panic details")
	}))

	recorder := httptest.NewRecorder()
	request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/panic", nil)
	router.ServeHTTP(recorder, request)

	var body map[string]any
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
	require.Equal(t, http.StatusInternalServerError, recorder.Code)
	require.Equal(t, "Something went wrong", body["error"])
	require.Equal(t, string(apperror.CodeInternal), body["code"])
	require.NotContains(t, recorder.Body.String(), "private panic details")
	require.NotEmpty(t, body["request_id"])
}

func TestErrorHandlerMiddlewareIncludesSafeDetails(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(NewErrorHandlerMiddleware().Add())
	router.GET("/users", func(c *gin.Context) {
		_ = c.Error(apperror.AlreadyInUse("email"))
		c.Abort()
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/users", nil)
	router.ServeHTTP(recorder, request)

	var body struct {
		Details map[string]string `json:"details"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
	require.Equal(t, map[string]string{"property": "email"}, body.Details)
}

func TestErrorHandlerMiddlewareSetsRetryAfter(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(NewErrorHandlerMiddleware().Add())
	router.GET("/limited", func(c *gin.Context) {
		_ = c.Error(apperror.TooManyRequests().WithRetryAfter(1500 * time.Millisecond))
		c.Abort()
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/limited", nil)
	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusTooManyRequests, recorder.Code)
	require.Equal(t, "2", recorder.Header().Get("Retry-After"))
}

func TestValidationResponseUsesJSONFieldNames(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(NewErrorHandlerMiddleware().Add())
	router.POST("/users", httpserver.Handle(func(c *gin.Context) error {
		var input dto.UserCreateDto
		return httpserver.BindJSON(c, &input)
	}))

	recorder := httptest.NewRecorder()
	request := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/users", strings.NewReader(`{}`))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)

	var body struct {
		Details struct {
			Fields []apperror.FieldError `json:"fields"`
		} `json:"details"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
	require.Equal(t, http.StatusBadRequest, recorder.Code)
	require.Contains(t, body.Details.Fields, apperror.FieldError{
		Field:   "username",
		Code:    "required",
		Message: "is required",
	})
}

func TestClassifyUnmappedPersistenceErrorAsInternal(t *testing.T) {
	classified := classifyError(gorm.ErrRecordNotFound)

	require.Equal(t, apperror.CodeInternal, classified.code)
	require.Equal(t, http.StatusInternalServerError, classified.status)
}

func TestClassifyDeadlineAsRequestTimeout(t *testing.T) {
	classified := classifyError(context.DeadlineExceeded)

	require.Equal(t, apperror.CodeRequestTimeout, classified.code)
	require.Equal(t, http.StatusGatewayTimeout, classified.status)
	require.Equal(t, "Request timed out", classified.message)
}
