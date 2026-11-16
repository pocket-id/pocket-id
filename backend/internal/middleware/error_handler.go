package middleware

import (
	"net/http"
	"strconv"
	"time"
	"uuid"

	"github.com/gin-gonic/gin"

	"github.com/pocket-id/pocket-id/backend/internal/apperror"
	"github.com/pocket-id/pocket-id/backend/internal/authz"
	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/dto"
	"github.com/pocket-id/pocket-id/backend/internal/httpserver"
)

const requestIDHeader = "X-Request-ID"

type requestIDContextKey struct{}
type requestErrorCodeContextKey struct{}
type requestErrorContextKey struct{}

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

// RequestError returns the error handled for the current request, so the request log can record it
func RequestError(c *gin.Context) error {
	if c == nil {
		return nil
	}

	value, exists := c.Get(requestErrorContextKey{})
	if !exists {
		return nil
	}

	err, _ := value.(error)
	return err
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
		c.Set(requestErrorContextKey{}, err)

		if c.Writer.Written() {
			return
		}

		if classified.RetryAfter > 0 {
			c.Header("Retry-After", formatRetryAfter(classified.RetryAfter))
		}
		// MCP clients follow the challenge to the endpoint's protected resource metadata to find out where to get a token (RFC 9728)
		if classified.Status == http.StatusUnauthorized && c.Request.URL.Path == authz.MCPEndpointPath {
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
