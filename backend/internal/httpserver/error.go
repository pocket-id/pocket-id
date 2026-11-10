package httpserver

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"
	"unicode"

	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"

	"github.com/pocket-id/pocket-id/backend/internal/apperror"
	"github.com/pocket-id/pocket-id/backend/internal/dto"
)

// ClassifiedError is an error reduced to what is safe to show its caller
type ClassifiedError struct {
	Code       apperror.Code
	Status     int
	Message    string
	Details    map[string]string
	Fields     []apperror.FieldError
	RetryAfter time.Duration
}

// ClassifyError maps an error to the code, status and message shown to the caller, hiding unexpected errors behind a generic internal error
func ClassifyError(err error) ClassifiedError {
	var structuredErr *apperror.Error
	if errors.As(err, &structuredErr) && structuredErr != nil {
		return ClassifiedError{
			Code:       structuredErr.Code(),
			Status:     normalizeStatus(structuredErr.HTTPStatus()),
			Message:    capitalizeFirst(structuredErr.ClientMessage()),
			Details:    structuredErr.Details(),
			Fields:     structuredErr.Fields(),
			RetryAfter: structuredErr.RetryAfter(),
		}
	}

	if errors.Is(err, context.DeadlineExceeded) {
		return ClassifiedError{
			Code:    apperror.CodeRequestTimeout,
			Status:  http.StatusGatewayTimeout,
			Message: "Request timed out",
		}
	}

	var validationErrors validator.ValidationErrors
	if errors.As(err, &validationErrors) && len(validationErrors) > 0 {
		return classifiedValidationError(validationErrors)
	}

	var sliceValidationErrors binding.SliceValidationError
	if errors.As(err, &sliceValidationErrors) && len(sliceValidationErrors) > 0 {
		if errors.As(sliceValidationErrors[0], &validationErrors) {
			return classifiedValidationError(validationErrors)
		}
	}

	return ClassifiedError{
		Code:    apperror.CodeInternal,
		Status:  http.StatusInternalServerError,
		Message: "Something went wrong",
	}
}

func classifiedValidationError(validationErrors validator.ValidationErrors) ClassifiedError {
	fields := make([]apperror.FieldError, 0, len(validationErrors))
	messages := make([]string, 0, len(validationErrors))

	for _, validationError := range validationErrors {
		fieldName := validationError.Field()
		code, message := dto.ValidationErrorDetails(validationError)
		fields = append(fields, apperror.FieldError{
			Field:   fieldName,
			Code:    code,
			Message: message,
		})
		messages = append(messages, fieldName+" "+message)
	}

	return ClassifiedError{
		Code:    apperror.CodeValidationFailed,
		Status:  http.StatusBadRequest,
		Message: capitalizeFirst(strings.Join(messages, ", ")),
		Fields:  fields,
	}
}

func normalizeStatus(status int) int {
	if status < http.StatusBadRequest || status > 599 {
		return http.StatusInternalServerError
	}

	return status
}

func capitalizeFirst(message string) string {
	runes := []rune(message)
	if len(runes) == 0 {
		return message
	}

	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}
