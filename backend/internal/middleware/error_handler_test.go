package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/service"
	testutils "github.com/pocket-id/pocket-id/backend/internal/utils/testing"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
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

	router.GET("/one_time_access_invalid_token", func(c *gin.Context) {
		_, _, err := oneTimeAccessService.ExchangeOneTimeAccessToken(c.Request.Context(), "invalid-token", "", "127.0.0.1", "test-agent")
		_ = c.Error(err)
	})

	router.GET("/webauthn_protocol_error", func(c *gin.Context) {
		// Emulate a failed WebAuthn login: parsing/validating the assertion surfaces a
		// *protocol.Error, and the failed sign-in is recorded in the audit log.
		_, err := protocol.ParseCredentialRequestResponseBody(c.Request.Body)
		if err == nil {
			err = &protocol.Error{Type: "verification_error", Details: "webauthn login failed"}
		}
		auditLogService.CreateSignInFailure(c.Request.Context(), "127.0.0.1", "test-agent", "")
		_ = c.Error(err)
	})

	router.GET("/404", func(c *gin.Context) {
		_ = c.Error(&common.UserNotFoundError{})
	})

	t.Run("logs security event for OneTimeAccess invalid token", func(t *testing.T) {
		// given
		mockHandler.lastEvent = ""
		request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/one_time_access_invalid_token", nil)
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
		request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/webauthn_protocol_error", bytes.NewBuffer(jsonBody))
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
		request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/404", nil)
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
		require.Empty(t, mockHandler.lastEvent)

		var signInFailureCount int64
		db.Table("audit_logs").Count(&signInFailureCount)
		require.Equal(t, initialSignInFailureCount, signInFailureCount)
	})
}

// testAppErrorWithDesc is a test-local error type that implements common.AppErrorDescription
type testAppErrorWithDesc struct {
	statusCode  int
	message     string
	description string
}

func (e testAppErrorWithDesc) Error() string       { return e.message }
func (e testAppErrorWithDesc) HttpStatusCode() int { return e.statusCode }
func (e testAppErrorWithDesc) Description() string { return e.description }

//nolint:gocognit
func TestErrorHandlerMiddlewareUnit(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockHandler := &mockSlogHandler{}
	slog.SetDefault(slog.New(mockHandler))

	router := gin.New()
	router.Use(NewErrorHandlerMiddleware().Add())

	v := validator.New()

	// Struct types used for generating real validator.ValidationErrors
	type requiredField struct {
		Name string `validate:"required"`
	}
	type emailField struct {
		Email string `validate:"email"`
	}
	type minField struct {
		Name string `validate:"min=5"`
	}
	type maxField struct {
		Name string `validate:"max=3"`
	}
	type urlField struct {
		URL string `validate:"url"`
	}
	type multiField struct {
		Name  string `validate:"required"`
		Email string `validate:"email"`
	}

	router.GET("/no-error", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	router.GET("/gorm-not-found", func(c *gin.Context) {
		_ = c.Error(gorm.ErrRecordNotFound)
	})
	router.GET("/validation-required", func(c *gin.Context) {
		_ = c.Error(v.Struct(requiredField{}))
	})
	router.GET("/validation-email", func(c *gin.Context) {
		_ = c.Error(v.Struct(emailField{Email: "not-an-email"}))
	})
	router.GET("/validation-min", func(c *gin.Context) {
		_ = c.Error(v.Struct(minField{Name: "ab"}))
	})
	router.GET("/validation-max", func(c *gin.Context) {
		_ = c.Error(v.Struct(maxField{Name: "abcd"}))
	})
	router.GET("/validation-url", func(c *gin.Context) {
		_ = c.Error(v.Struct(urlField{URL: "not-a-url"}))
	})
	router.GET("/validation-multi", func(c *gin.Context) {
		// Both fields fail: Name is empty (required) and Email is malformed
		_ = c.Error(v.Struct(multiField{Email: "bad"}))
	})
	router.GET("/slice-validation-error", func(c *gin.Context) {
		// Wrap a ValidationErrors inside a SliceValidationError to exercise the slice unwrapping path
		inner := v.Struct(emailField{Email: "bad"})
		_ = c.Error(binding.SliceValidationError{inner})
	})
	router.GET("/app-error-desc-400", func(c *gin.Context) {
		_ = c.Error(testAppErrorWithDesc{statusCode: http.StatusBadRequest, message: "bad thing", description: "more details"})
	})
	router.GET("/app-error-desc-403", func(c *gin.Context) {
		_ = c.Error(testAppErrorWithDesc{statusCode: http.StatusForbidden, message: "forbidden thing", description: "access denied details"})
	})
	router.GET("/app-error-403", func(c *gin.Context) {
		_ = c.Error(common.MissingPermissionError{})
	})
	router.GET("/protocol-policy-restriction", func(c *gin.Context) {
		_ = c.Error(&protocol.Error{Type: "policy_restriction", Details: "restricted"})
	})
	router.GET("/protocol-spec-unimplemented", func(c *gin.Context) {
		_ = c.Error(&protocol.Error{Type: "spec_unimplemented", Details: "not done"})
	})
	router.GET("/protocol-unknown-type", func(c *gin.Context) {
		_ = c.Error(&protocol.Error{Type: "some_unknown_type", Details: "unknown"})
	})
	router.GET("/unhandled-error", func(c *gin.Context) {
		_ = c.Error(errors.New("some unexpected internal error"))
	})

	t.Run("no error passes through with original status", func(t *testing.T) {
		// given
		mockHandler.lastEvent = ""
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/no-error", nil)
		rec := httptest.NewRecorder()

		// when
		router.ServeHTTP(rec, req)

		// then
		require.Equal(t, http.StatusOK, rec.Code)
		require.Empty(t, rec.Body.Bytes())
		require.Empty(t, mockHandler.lastEvent)
	})

	t.Run("gorm ErrRecordNotFound returns 404", func(t *testing.T) {
		// given
		mockHandler.lastEvent = ""
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/gorm-not-found", nil)
		rec := httptest.NewRecorder()

		// when
		router.ServeHTTP(rec, req)

		// then
		var body map[string]string
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		require.Equal(t, http.StatusNotFound, rec.Code)
		require.Equal(t, "Record not found", body["error"])
		require.Empty(t, mockHandler.lastEvent)
	})

	t.Run("validation error with required tag returns 400 with field name", func(t *testing.T) {
		// given
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/validation-required", nil)
		rec := httptest.NewRecorder()

		// when
		router.ServeHTTP(rec, req)

		// then
		var body map[string]string
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		require.Equal(t, http.StatusBadRequest, rec.Code)
		require.Equal(t, "Name is required", body["error"])
	})

	t.Run("validation error with email tag returns 400 with field name", func(t *testing.T) {
		// given
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/validation-email", nil)
		rec := httptest.NewRecorder()

		// when
		router.ServeHTTP(rec, req)

		// then
		var body map[string]string
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		require.Equal(t, http.StatusBadRequest, rec.Code)
		require.Equal(t, "Email must be a valid email address", body["error"])
	})

	t.Run("validation error with min tag includes the minimum length in the message", func(t *testing.T) {
		// given
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/validation-min", nil)
		rec := httptest.NewRecorder()

		// when
		router.ServeHTTP(rec, req)

		// then
		var body map[string]string
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		require.Equal(t, http.StatusBadRequest, rec.Code)
		require.Equal(t, "Name must be at least 5 characters long", body["error"])
	})

	t.Run("validation error with max tag includes the maximum length in the message", func(t *testing.T) {
		// given
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/validation-max", nil)
		rec := httptest.NewRecorder()

		// when
		router.ServeHTTP(rec, req)

		// then
		var body map[string]string
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		require.Equal(t, http.StatusBadRequest, rec.Code)
		require.Equal(t, "Name must be at most 3 characters long", body["error"])
	})

	t.Run("validation error with url tag returns 400 with field name", func(t *testing.T) {
		// given
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/validation-url", nil)
		rec := httptest.NewRecorder()

		// when
		router.ServeHTTP(rec, req)

		// then
		var body map[string]string
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		require.Equal(t, http.StatusBadRequest, rec.Code)
		require.Equal(t, "URL must be a valid URL", body["error"])
	})

	t.Run("multiple validation errors are joined with a comma", func(t *testing.T) {
		// given
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/validation-multi", nil)
		rec := httptest.NewRecorder()

		// when
		router.ServeHTTP(rec, req)

		// then
		var body map[string]string
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		require.Equal(t, http.StatusBadRequest, rec.Code)
		require.Contains(t, body["error"], "Name is required")
		require.Contains(t, body["error"], "Email must be a valid email address")
	})

	t.Run("slice validation error unwraps inner ValidationErrors and returns 400", func(t *testing.T) {
		// given
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/slice-validation-error", nil)
		rec := httptest.NewRecorder()

		// when
		router.ServeHTTP(rec, req)

		// then
		var body map[string]string
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		require.Equal(t, http.StatusBadRequest, rec.Code)
		require.Equal(t, "Email must be a valid email address", body["error"])
	})

	t.Run("app error with description returns both error and error_description fields", func(t *testing.T) {
		// given
		mockHandler.lastEvent = ""
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/app-error-desc-400", nil)
		rec := httptest.NewRecorder()

		// when
		router.ServeHTTP(rec, req)

		// then
		var body map[string]string
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		require.Equal(t, http.StatusBadRequest, rec.Code)
		require.Equal(t, "Bad thing", body["error"])
		require.Equal(t, "more details", body["error_description"])
		require.Empty(t, mockHandler.lastEvent)
	})

	t.Run("app error with description and 403 status logs security event", func(t *testing.T) {
		// given
		mockHandler.lastEvent = ""
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/app-error-desc-403", nil)
		rec := httptest.NewRecorder()

		// when
		router.ServeHTTP(rec, req)

		// then
		var body map[string]string
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		require.Equal(t, http.StatusForbidden, rec.Code)
		require.Equal(t, "Forbidden thing", body["error"])
		require.Equal(t, "access denied details", body["error_description"])
		require.Equal(t, "Security event", mockHandler.lastEvent)
	})

	t.Run("app error with 403 status logs security event", func(t *testing.T) {
		// given
		mockHandler.lastEvent = ""
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/app-error-403", nil)
		rec := httptest.NewRecorder()

		// when
		router.ServeHTTP(rec, req)

		// then
		var body map[string]string
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		require.Equal(t, http.StatusForbidden, rec.Code)
		require.Equal(t, "Security event", mockHandler.lastEvent)
	})

	t.Run("webauthn policy_restriction error maps to 403 and logs security event", func(t *testing.T) {
		// given
		mockHandler.lastEvent = ""
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/protocol-policy-restriction", nil)
		rec := httptest.NewRecorder()

		// when
		router.ServeHTTP(rec, req)

		// then
		var body map[string]string
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		require.Equal(t, http.StatusForbidden, rec.Code)
		require.Equal(t, "Something went wrong. Please try again later", body["error"])
		require.Equal(t, "Security event", mockHandler.lastEvent)
	})

	t.Run("webauthn spec_unimplemented error maps to 501 and logs security event", func(t *testing.T) {
		// given
		mockHandler.lastEvent = ""
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/protocol-spec-unimplemented", nil)
		rec := httptest.NewRecorder()

		// when
		router.ServeHTTP(rec, req)

		// then
		var body map[string]string
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		require.Equal(t, http.StatusNotImplemented, rec.Code)
		require.Equal(t, "Something went wrong. Please try again later", body["error"])
		require.Equal(t, "Security event", mockHandler.lastEvent)
	})

	t.Run("webauthn unknown error type defaults to 400 and logs security event", func(t *testing.T) {
		// given
		mockHandler.lastEvent = ""
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/protocol-unknown-type", nil)
		rec := httptest.NewRecorder()

		// when
		router.ServeHTTP(rec, req)

		// then
		var body map[string]string
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		require.Equal(t, http.StatusBadRequest, rec.Code)
		require.Equal(t, "Something went wrong. Please try again later", body["error"])
		require.Equal(t, "Security event", mockHandler.lastEvent)
	})

	t.Run("unhandled error returns 500 with generic message", func(t *testing.T) {
		// given
		mockHandler.lastEvent = ""
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/unhandled-error", nil)
		rec := httptest.NewRecorder()

		// when
		router.ServeHTTP(rec, req)

		// then
		var body map[string]string
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		require.Equal(t, http.StatusInternalServerError, rec.Code)
		require.Equal(t, "Something went wrong", body["error"])
		require.Empty(t, mockHandler.lastEvent)
	})
}
