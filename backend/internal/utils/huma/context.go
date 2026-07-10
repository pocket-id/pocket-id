package humautils

import (
	"context"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humagin"
)

type contextKey uint8

const (
	requestContextKey contextKey = iota
	clientIPContextKey
	userIDContextKey
	userIsAdminContextKey
	authenticationMethodContextKey
	authenticationTimeContextKey
)

// CaptureRequestContext exposes trusted Gin request metadata to typed handlers
func CaptureRequestContext(ctx huma.Context, next func(huma.Context)) {
	ginCtx := humagin.Unwrap(ctx)
	ctx = huma.WithValue(ctx, requestContextKey, ginCtx.Request)
	ctx = huma.WithValue(ctx, clientIPContextKey, ginCtx.ClientIP())
	next(ctx)
}

// WithAuthentication adds the authenticated identity to a Huma request context
func WithAuthentication(ctx huma.Context, userID string, isAdmin bool, method string, authenticationTime time.Time) huma.Context {
	ctx = huma.WithValue(ctx, userIDContextKey, userID)
	ctx = huma.WithValue(ctx, userIsAdminContextKey, isAdmin)
	ctx = huma.WithValue(ctx, authenticationMethodContextKey, method)
	return huma.WithValue(ctx, authenticationTimeContextKey, authenticationTime)
}

// Request returns the underlying HTTP request for protocol handlers that require it
func Request(ctx context.Context) *http.Request {
	request, _ := ctx.Value(requestContextKey).(*http.Request)
	return request
}

// ClientIP returns the trusted client IP calculated by Gin
func ClientIP(ctx context.Context) string {
	value, _ := ctx.Value(clientIPContextKey).(string)
	return value
}

// UserAgent returns the request user agent
func UserAgent(ctx context.Context) string {
	request := Request(ctx)
	if request == nil {
		return ""
	}
	return request.UserAgent()
}

// Cookie returns a dynamically named request cookie
func Cookie(ctx context.Context, name string) (*http.Cookie, error) {
	request := Request(ctx)
	if request == nil {
		return nil, http.ErrNoCookie
	}
	return request.Cookie(name)
}

// QueryPresent reports whether a query key was present regardless of its value
func QueryPresent(ctx context.Context, name string) bool {
	request := Request(ctx)
	if request == nil {
		return false
	}
	_, ok := request.URL.Query()[name]
	return ok
}

// UserID returns the authenticated user ID
func UserID(ctx context.Context) string {
	value, _ := ctx.Value(userIDContextKey).(string)
	return value
}

// IsAdmin reports whether the authenticated user is an administrator
func IsAdmin(ctx context.Context) bool {
	value, _ := ctx.Value(userIsAdminContextKey).(bool)
	return value
}

// AuthenticationMethod returns the session authentication method
func AuthenticationMethod(ctx context.Context) string {
	value, _ := ctx.Value(authenticationMethodContextKey).(string)
	return value
}

// AuthenticationTime returns the session authentication time
func AuthenticationTime(ctx context.Context) time.Time {
	value, _ := ctx.Value(authenticationTimeContextKey).(time.Time)
	return value
}
