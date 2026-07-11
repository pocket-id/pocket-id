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
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/common"
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

// webAuthnErrorToHTTPStatus maps WebAuthn protocol error types to HTTP status codes
func webAuthnErrorToHTTPStatus(errorType string) int {
	switch errorType {
	case "invalid_request":
		return http.StatusBadRequest
	case "policy_restriction":
		return http.StatusForbidden
	case "challenge_mismatch":
		return http.StatusBadRequest
	case "parse_error":
		return http.StatusBadRequest
	case "auth_data":
		return http.StatusBadRequest
	case "verification_error":
		return http.StatusBadRequest
	case "attestation_error":
		return http.StatusBadRequest
	case "invalid_attestation":
		return http.StatusBadRequest
	case "invalid_metadata":
		return http.StatusBadRequest
	case "invalid_certificate":
		return http.StatusBadRequest
	case "invalid_signature":
		return http.StatusBadRequest
	case "invalid_key_type":
		return http.StatusBadRequest
	case "unsupported_key_algorithm":
		return http.StatusBadRequest
	case "spec_unimplemented":
		return http.StatusNotImplemented
	case "not_implemented":
		return http.StatusNotImplemented
	default:
		return http.StatusBadRequest
	}
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
		case "resource_uri":
			errorMessage = fmt.Sprintf("%s must be an absolute URI without whitespace or a fragment", fieldName)
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
