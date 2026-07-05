package middleware

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/pocket-id/pocket-id/backend/internal/common"
	"gorm.io/gorm"
)

type ErrorHandlerMiddleware struct{}

func NewErrorHandlerMiddleware() *ErrorHandlerMiddleware {
	return &ErrorHandlerMiddleware{}
}

//nolint:gocognit
func (m *ErrorHandlerMiddleware) Add() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		for _, err := range c.Errors {
			// Check for record not found errors
			if errors.Is(err, gorm.ErrRecordNotFound) {
				errorResponse(c, http.StatusNotFound, "Record not found")
				return
			}

			// Check for validation errors
			var validationErrors validator.ValidationErrors
			if errors.As(err, &validationErrors) {
				message := handleValidationError(validationErrors)
				errorResponse(c, http.StatusBadRequest, message)
				return
			}

			// Check for slice validation errors
			svErr, ok := errors.AsType[binding.SliceValidationError](err)
			if ok {
				if errors.As(svErr[0], &validationErrors) {
					message := handleValidationError(validationErrors)
					errorResponse(c, http.StatusBadRequest, message)
					return
				}
			}

			// AppError with description
			appDescErr, ok := errors.AsType[common.AppErrorDescription](err)
			if ok {
				statusCode := appDescErr.HttpStatusCode()
				if isSecurityError(statusCode) {
					logSecurityEvent(c, appDescErr.Error())
				}
				errorResponseWithDescription(c, statusCode, appDescErr.Error(), appDescErr.Description())
				return
			}

			// AppError (without description)
			appErr, ok := errors.AsType[common.AppError](err)
			if ok {
				statusCode := appErr.HttpStatusCode()
				if isSecurityError(statusCode) {
					logSecurityEvent(c, appErr.Error())
				}
				errorResponse(c, statusCode, appErr.Error())
				return
			}

			protocolErr, ok := errors.AsType[*protocol.Error](err)
			if ok {
				statusCode := webAuthnErrorToHTTPStatus(protocolErr.Type)
				logSecurityEvent(c, protocolErr.Error())
				errorResponse(c, statusCode, "Something went wrong. Please try again later")
				return
			}

			c.JSON(http.StatusInternalServerError, errorResponseBody{
				Error: "Something went wrong",
			})
		}
	}
}

// webAuthnProtocolErrorHTTPStatus maps WebAuthn protocol error types to HTTP status codes.
var webAuthnProtocolErrorHTTPStatus = map[string]int{
	"invalid_request":           http.StatusBadRequest,
	"policy_restriction":        http.StatusForbidden,
	"challenge_mismatch":        http.StatusBadRequest,
	"parse_error":               http.StatusBadRequest,
	"auth_data":                 http.StatusBadRequest,
	"verification_error":        http.StatusBadRequest,
	"attestation_error":         http.StatusBadRequest,
	"invalid_attestation":       http.StatusBadRequest,
	"invalid_metadata":          http.StatusBadRequest,
	"invalid_certificate":       http.StatusBadRequest,
	"invalid_signature":         http.StatusBadRequest,
	"invalid_key_type":          http.StatusBadRequest,
	"unsupported_key_algorithm": http.StatusBadRequest,
	"spec_unimplemented":        http.StatusNotImplemented,
	"not_implemented":           http.StatusNotImplemented,
}

func webAuthnErrorToHTTPStatus(errorType string) int {
	if code, ok := webAuthnProtocolErrorHTTPStatus[errorType]; ok {
		return code
	}
	return http.StatusBadRequest
}

func isSecurityError(statusCode int) bool {
	return statusCode == http.StatusUnauthorized || statusCode == http.StatusForbidden
}

func logSecurityEvent(c *gin.Context, message string) {
	slog.WarnContext(c.Request.Context(), "Security event",
		slog.String("event", "auth_failure"),
		slog.String("error", message),
		slog.String("ip", c.ClientIP()),
		slog.String("user_agent", c.Request.UserAgent()),
		slog.String("path", c.Request.URL.Path),
	)
}

type errorResponseBody struct {
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description,omitempty"`
}

func errorResponse(c *gin.Context, statusCode int, message string) {
	// Capitalize the first letter of the message
	message = strings.ToUpper(message[:1]) + message[1:]
	c.JSON(statusCode, errorResponseBody{
		Error: message,
	})
}

func errorResponseWithDescription(c *gin.Context, statusCode int, message string, description string) {
	// Capitalize the first letter of the message
	message = strings.ToUpper(message[:1]) + message[1:]
	c.JSON(statusCode, errorResponseBody{
		Error:            message,
		ErrorDescription: description,
	})
}

func handleValidationError(validationErrors validator.ValidationErrors) string {
	var errorMessages []string

	for _, ve := range validationErrors {
		fieldName := ve.Field()
		var errorMessage string
		switch ve.Tag() {
		case "required":
			errorMessage = fmt.Sprintf("%s is required", fieldName)
		case "email":
			errorMessage = fmt.Sprintf("%s must be a valid email address", fieldName)
		case "username":
			errorMessage = fmt.Sprintf("%s must only contain letters, numbers, underscores, dots, hyphens, and '@' symbols and not start or end with a special character", fieldName)
		case "url":
			errorMessage = fmt.Sprintf("%s must be a valid URL", fieldName)
		case "min":
			errorMessage = fmt.Sprintf("%s must be at least %s characters long", fieldName, ve.Param())
		case "max":
			errorMessage = fmt.Sprintf("%s must be at most %s characters long", fieldName, ve.Param())
		default:
			errorMessage = fmt.Sprintf("%s is invalid", fieldName)
		}

		errorMessages = append(errorMessages, errorMessage)
	}

	// Join all the error messages into a single string
	combinedErrors := strings.Join(errorMessages, ", ")

	return combinedErrors
}
