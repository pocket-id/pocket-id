package middleware

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"reflect"
	"strings"
	"time"
	"unicode"
	"uuid"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	"go.opentelemetry.io/otel/trace"

	"github.com/pocket-id/pocket-id/backend/internal/apperror"
	"github.com/pocket-id/pocket-id/backend/internal/dto"
)

const requestIDHeader = "X-Request-ID"

type requestIDContextKey struct{}
type requestErrorCodeContextKey struct{}

// RequestID returns the identifier assigned to the current request
func RequestID(c *gin.Context) string {
	if c == nil {
		return ""
	}

	requestID, exists := c.Get(requestIDContextKey{})
	if !exists {
		return ""
	}

	value, ok := requestID.(string)
	if !ok {
		return ""
	}

	return value
}

// RequestErrorCode returns the stable code of the error handled for the current request
func RequestErrorCode(c *gin.Context) apperror.Code {
	if c == nil {
		return ""
	}

	code, exists := c.Get(requestErrorCodeContextKey{})
	if !exists {
		return ""
	}

	value, ok := code.(apperror.Code)
	if !ok {
		return ""
	}

	return value
}

type ErrorHandlerMiddleware struct{}

func NewErrorHandlerMiddleware() *ErrorHandlerMiddleware {
	return &ErrorHandlerMiddleware{}
}

type classifiedError struct {
	code       apperror.Code
	status     int
	message    string
	details    map[string]string
	fields     []apperror.FieldError
	retryAfter time.Duration
}

// Add records a request ID before executing the request and serializes the first returned error afterward
func (m *ErrorHandlerMiddleware) Add() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := uuid.NewV4().String()
		c.Set(requestIDContextKey{}, requestID)
		c.Header(requestIDHeader, requestID)

		c.Next()
		if len(c.Errors) == 0 {
			return
		}

		err := c.Errors[0].Err
		classified := classifyError(err)
		c.Set(requestErrorCodeContextKey{}, classified.code)
		logRequestError(c, err, classified, requestID)

		if c.Writer.Written() {
			return
		}

		if classified.retryAfter > 0 {
			c.Header("Retry-After", formatRetryAfter(classified.retryAfter))
		}
		writeErrorResponse(c, classified, requestID)
	}
}

func classifyError(err error) classifiedError {
	var structuredErr *apperror.Error
	if errors.As(err, &structuredErr) && structuredErr != nil {
		return classifiedError{
			code:       structuredErr.Code(),
			status:     normalizeStatus(structuredErr.HTTPStatus()),
			message:    capitalizeFirst(structuredErr.ClientMessage()),
			details:    structuredErr.Details(),
			fields:     structuredErr.Fields(),
			retryAfter: structuredErr.RetryAfter(),
		}
	}

	if errors.Is(err, context.DeadlineExceeded) {
		return classifiedError{
			code:    apperror.CodeRequestTimeout,
			status:  http.StatusGatewayTimeout,
			message: "Request timed out",
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

	return classifiedError{
		code:    apperror.CodeInternal,
		status:  http.StatusInternalServerError,
		message: "Something went wrong",
	}
}

func classifiedValidationError(validationErrors validator.ValidationErrors) classifiedError {
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

	return classifiedError{
		code:    apperror.CodeValidationFailed,
		status:  http.StatusBadRequest,
		message: capitalizeFirst(strings.Join(messages, ", ")),
		fields:  fields,
	}
}

func writeErrorResponse(c *gin.Context, classified classifiedError, requestID string) {
	details := make(map[string]any, len(classified.details)+1)
	for key, value := range classified.details {
		details[key] = value
	}
	if len(classified.fields) > 0 {
		details["fields"] = classified.fields
	}
	if len(details) == 0 {
		details = nil
	}

	response := dto.ErrorDto{
		Error:     classified.message,
		Code:      classified.code,
		Details:   details,
		RequestID: requestID,
	}

	c.JSON(classified.status, response)
}

func logRequestError(c *gin.Context, err error, classified classifiedError, requestID string) {
	if classified.status < http.StatusInternalServerError {
		return
	}

	attrs := []any{
		slog.String("error_code", string(classified.code)),
		slog.String("error_type", errorTypeName(err)),
		slog.Int("http_status", classified.status),
		slog.String("request_id", requestID),
		slog.String("http_method", c.Request.Method),
		slog.String("http_path", c.Request.URL.Path),
		slog.Any("error", err),
	}
	if spanContext := trace.SpanFromContext(c.Request.Context()).SpanContext(); spanContext.IsValid() {
		attrs = append(attrs, slog.String("trace_id", spanContext.TraceID().String()))
	}

	slog.ErrorContext(c.Request.Context(), "Request failed", attrs...)
}

func errorTypeName(err error) string {
	if err == nil {
		return "<nil>"
	}

	return reflect.TypeOf(err).String()
}

func normalizeStatus(status int) int {
	if status < http.StatusBadRequest || status > 599 {
		return http.StatusInternalServerError
	}

	return status
}

func formatRetryAfter(retryAfter time.Duration) string {
	seconds := int(retryAfter / time.Second)
	if retryAfter%time.Second != 0 {
		seconds++
	}
	if seconds < 1 {
		seconds = 1
	}

	return fmt.Sprintf("%d", seconds)
}

func capitalizeFirst(message string) string {
	runes := []rune(message)
	if len(runes) == 0 {
		return message
	}

	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}
