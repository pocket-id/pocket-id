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
	"github.com/pocket-id/pocket-id/backend/internal/appconfig"
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

func TestValidationResponseUsesSpecificCustomMessages(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(NewErrorHandlerMiddleware().Add())
	router.POST("/config", httpserver.Handle(func(c *gin.Context) error {
		var input struct {
			SignupDefaultCustomClaims string `json:"signupDefaultCustomClaims" binding:"json_custom_claims"`
		}
		return httpserver.BindJSON(c, &input)
	}))

	recorder := httptest.NewRecorder()
	request := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/config", strings.NewReader(`{"signupDefaultCustomClaims":"[\"immich_role\": \"user\"]"}`))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)

	var body struct {
		Error   string `json:"error"`
		Details struct {
			Fields []apperror.FieldError `json:"fields"`
		} `json:"details"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
	require.Equal(t, http.StatusBadRequest, recorder.Code)
	require.Equal(t, `SignupDefaultCustomClaims must be a JSON array of objects with string "key" and "value" properties`, body.Error)
	require.Equal(t, []apperror.FieldError{{
		Field:   "signupDefaultCustomClaims",
		Code:    "invalid_format",
		Message: `must be a JSON array of objects with string "key" and "value" properties`,
	}}, body.Details.Fields)
}

func TestValidationResponseUsesAppConfigTypeMessages(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(NewErrorHandlerMiddleware().Add())
	router.POST("/config", httpserver.Handle(func(c *gin.Context) error {
		var input dto.AppConfigUpdateDto
		return httpserver.BindJSON(c, &input)
	}))

	input := dto.AppConfigUpdateDto{
		AppName:                          "Pocket ID",
		SessionDuration:                  "60",
		HomePageURL:                      "/settings/account",
		EmailsVerified:                   "false",
		DisableAnimations:                "false",
		AllowOwnAccountEdit:              "true",
		AllowUserSignups:                 "disabled",
		RequireUserEmail:                 "hello",
		SmtpTls:                          "none",
		SmtpSkipCertVerify:               "false",
		LdapEnabled:                      "false",
		LdapSkipCertVerify:               "false",
		LdapSoftDeleteUsers:              "true",
		WebauthnUserVerification:         "preferred",
		WebauthnAllowSyncedPasskeys:      "true",
		WebauthnAuthenticatorAttachment:  "any",
		EmailOneTimeAccessAsAdminEnabled: "false",
		EmailOneTimeAccessAsUnauthenticatedEnabled: "false",
		EmailLoginNotificationMode:                 "disabled",
		EmailApiKeyExpirationEnabled:               "false",
		EmailVerification:                          "disabled",
		AutoCreateOIDCClientSecret:                 "true",
	}
	payload, err := json.Marshal(input)
	require.NoError(t, err)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/config", strings.NewReader(string(payload)))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)

	var body struct {
		Error   string `json:"error"`
		Details struct {
			Fields []apperror.FieldError `json:"fields"`
		} `json:"details"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
	require.Equal(t, http.StatusBadRequest, recorder.Code)
	require.Equal(t, "RequireUserEmail must be either true or false", body.Error)
	require.Equal(t, []apperror.FieldError{{
		Field:   "requireUserEmail",
		Code:    "invalid_format",
		Message: "must be either true or false",
	}}, body.Details.Fields)
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

func TestAppConfigBindingValidatesRequiredEmailVerification(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(NewErrorHandlerMiddleware().Add())
	router.PUT("/config", httpserver.Handle(func(c *gin.Context) error {
		var input dto.AppConfigUpdateDto
		return httpserver.BindJSON(c, &input)
	}))

	const (
		smtpHost = "smtp.example.com"
		smtpFrom = "pocket-id@example.com"
	)
	for _, tt := range []struct {
		name, mode, required, verified, host, from string
		status                                     int
	}{
		{"required with trusted emails", "required", "true", "true", smtpHost, smtpFrom, http.StatusBadRequest},
		{"required without email", "required", "false", "false", smtpHost, smtpFrom, http.StatusBadRequest},
		{"required without SMTP host", "required", "true", "false", "", smtpFrom, http.StatusBadRequest},
		{"required without sender address", "required", "true", "false", smtpHost, "", http.StatusBadRequest},
		{"valid required", "required", "true", "false", smtpHost, smtpFrom, http.StatusOK},
		{"optional with trusted emails", "optional", "false", "true", "", "", http.StatusOK},
		{"disabled with trusted emails", "disabled", "false", "true", "", "", http.StatusOK},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var input dto.AppConfigUpdateDto
			require.NoError(t, dto.MapStruct(appconfig.NewTestConfig(nil), &input))
			input.EmailVerification = tt.mode
			input.RequireUserEmail = tt.required
			input.EmailsVerified = tt.verified
			input.SmtpHost = tt.host
			input.SmtpFrom = tt.from
			payload, err := json.Marshal(input)
			require.NoError(t, err)
			request := httptest.NewRequestWithContext(t.Context(), http.MethodPut, "/config", strings.NewReader(string(payload)))
			request.Header.Set("Content-Type", "application/json")
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, request)

			require.Equal(t, tt.status, recorder.Code, recorder.Body.String())
			if tt.status == http.StatusBadRequest {
				var body struct {
					Details struct {
						Fields []apperror.FieldError `json:"fields"`
					} `json:"details"`
				}
				require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
				require.Equal(t, []apperror.FieldError{{Field: "emailVerification", Code: "invalid_configuration", Message: "requires user emails, an SMTP host and sender address, and forbids emails verified by default"}}, body.Details.Fields)
			}
		})
	}
}
