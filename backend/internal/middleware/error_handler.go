package middleware

import (
	"errors"
	"log/slog"
	"net/http"
	"reflect"
	"strconv"
	"time"
	"uuid"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/trace"

	"github.com/pocket-id/pocket-id/backend/internal/apperror"
	"github.com/pocket-id/pocket-id/backend/internal/authz"
	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/dto"
	"github.com/pocket-id/pocket-id/backend/internal/httpserver"
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
		classified := httpserver.ClassifyError(err)
		c.Set(requestErrorCodeContextKey{}, classified.Code)
		logRequestError(c, err, classified, requestID)

		if c.Writer.Written() {
			return
		}

		if classified.RetryAfter > 0 {
			c.Header("Retry-After", formatRetryAfter(classified.RetryAfter))
		}
		// MCP clients follow the challenge to the endpoint's protected resource metadata to find out where to get a token (RFC 9728)
		if classified.Status == http.StatusUnauthorized && c.Request.URL.Path == authz.MCPPath {
			c.Header("WWW-Authenticate", `Bearer resource_metadata="`+common.EnvConfig.AppURL+authz.MCPResourceMetadataPath+`"`)
		}
		writeErrorResponse(c, classified, requestID)
	}
}

func writeErrorResponse(c *gin.Context, classified httpserver.ClassifiedError, requestID string) {
	details := make(map[string]any, len(classified.Details)+1)
	for key, value := range classified.Details {
		details[key] = value
	}
	if len(classified.Fields) > 0 {
		details["fields"] = classified.Fields
	}
	if len(details) == 0 {
		details = nil
	}

	response := dto.ErrorDto{
		Error:     classified.Message,
		Code:      classified.Code,
		Details:   details,
		RequestID: requestID,
	}

	c.JSON(classified.Status, response)
}

func logRequestError(c *gin.Context, err error, classified httpserver.ClassifiedError, requestID string) {
	if classified.Status < http.StatusInternalServerError {
		cause := errors.Unwrap(err)
		if cause == nil {
			return
		}

		slog.DebugContext(c.Request.Context(), "Request rejected",
			slog.String("error_code", string(classified.Code)),
			slog.String("error_type", errorTypeName(err)),
			slog.String("cause_type", errorTypeName(cause)),
			slog.Int("http_status", classified.Status),
			slog.String("request_id", requestID),
			slog.String("http_method", c.Request.Method),
			slog.String("http_path", c.Request.URL.Path),
			slog.Any("error", err),
			slog.Any("cause", cause),
		)
		return
	}

	attrs := []any{
		slog.String("error_code", string(classified.Code)),
		slog.String("error_type", errorTypeName(err)),
		slog.Int("http_status", classified.Status),
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

func formatRetryAfter(retryAfter time.Duration) string {
	seconds := int(retryAfter / time.Second)
	if retryAfter%time.Second != 0 {
		seconds++
	}
	if seconds < 1 {
		seconds = 1
	}

	return strconv.Itoa(seconds)
}
