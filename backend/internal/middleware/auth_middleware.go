package middleware

import (
	"errors"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humagin"
	"github.com/gin-gonic/gin"
	"github.com/pocket-id/pocket-id/backend/internal/apikey"
	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/service"
	httpapi "github.com/pocket-id/pocket-id/backend/internal/utils/huma"
)

// AuthMiddleware is a wrapper middleware that delegates to either API key or JWT authentication
type AuthMiddleware struct {
	apiKeyMiddleware *ApiKeyAuthMiddleware
	jwtMiddleware    *JwtAuthMiddleware
	options          AuthOptions
}

type AuthOptions struct {
	AdminRequired   bool
	SuccessOptional bool
	AllowApiKeyAuth bool
}

func NewAuthMiddleware(
	apiKeyModule *apikey.Module,
	userService *service.UserService,
	jwtService *service.JwtService,
) *AuthMiddleware {
	return &AuthMiddleware{
		apiKeyMiddleware: NewApiKeyAuthMiddleware(apiKeyModule, jwtService),
		jwtMiddleware:    NewJwtAuthMiddleware(jwtService, userService),
		options: AuthOptions{
			AdminRequired:   true,
			SuccessOptional: false,
			AllowApiKeyAuth: true,
		},
	}
}

// WithAdminNotRequired allows the middleware to continue with the request even if the user is not an admin
func (m *AuthMiddleware) WithAdminNotRequired() *AuthMiddleware {
	// Create a new instance to avoid modifying the original
	clone := &AuthMiddleware{
		apiKeyMiddleware: m.apiKeyMiddleware,
		jwtMiddleware:    m.jwtMiddleware,
		options:          m.options,
	}
	clone.options.AdminRequired = false
	return clone
}

// WithSuccessOptional allows the middleware to continue with the request even if authentication fails
func (m *AuthMiddleware) WithSuccessOptional() *AuthMiddleware {
	// Create a new instance to avoid modifying the original
	clone := &AuthMiddleware{
		apiKeyMiddleware: m.apiKeyMiddleware,
		jwtMiddleware:    m.jwtMiddleware,
		options:          m.options,
	}
	clone.options.SuccessOptional = true
	return clone
}

// WithApiKeyAuthDisabled disables API key authentication fallback and requires JWT auth
func (m *AuthMiddleware) WithApiKeyAuthDisabled() *AuthMiddleware {
	clone := &AuthMiddleware{
		apiKeyMiddleware: m.apiKeyMiddleware,
		jwtMiddleware:    m.jwtMiddleware,
		options:          m.options,
	}
	clone.options.AllowApiKeyAuth = false
	return clone
}

func (m *AuthMiddleware) Add() gin.HandlerFunc {
	return func(c *gin.Context) {
		result, err := m.authenticate(c)
		if err != nil {
			c.Abort()
			_ = c.Error(err)
			return
		}
		if result.UserID != "" {
			setGinAuthentication(c, result)
		}
		c.Next()
	}
}

// Huma returns an operation decorator using the same authentication behavior as Add
func (m *AuthMiddleware) Huma(api huma.API) func(*huma.Operation) {
	return func(operation *huma.Operation) {
		operation.Security = m.securityRequirements()
		if m.options.AdminRequired {
			if operation.Extensions == nil {
				operation.Extensions = map[string]any{}
			}
			operation.Extensions["x-pocket-id-admin-required"] = true
		}
		operation.Middlewares = append(operation.Middlewares, func(ctx huma.Context, next func(huma.Context)) {
			result, err := m.authenticate(humagin.Unwrap(ctx))
			if err != nil {
				status := 500
				message := "Something went wrong"
				var appError common.AppError
				if errors.As(err, &appError) {
					status = appError.HttpStatusCode()
					message = appError.Error()
				}
				_ = huma.WriteErr(api, ctx, status, message)
				return
			}
			if result.UserID != "" {
				ctx = httpapi.WithAuthentication(ctx, result.UserID, result.IsAdmin, result.AuthenticationMethod, result.AuthenticationTime)
			}
			next(ctx)
		})
	}
}

type authenticationResult struct {
	UserID               string
	IsAdmin              bool
	AuthenticationMethod string
	AuthenticationTime   time.Time
}

func (m *AuthMiddleware) authenticate(c *gin.Context) (authenticationResult, error) {
	// Return immediately after successful JWT authentication
	userID, isAdmin, authenticationMethod, authenticationTime, err := m.jwtMiddleware.Verify(c, m.options.AdminRequired)
	if err == nil {
		return authenticationResult{userID, isAdmin, authenticationMethod, authenticationTime}, nil
	}

	// Fall back only when JWT verification reports that the request is not signed in
	if !errors.Is(err, &common.NotSignedInError{}) {
		return authenticationResult{}, err
	}

	// Handle API-key-disabled routes before considering API-key authentication
	if !m.options.AllowApiKeyAuth {
		if m.options.SuccessOptional {
			return authenticationResult{}, nil
		}
		if c.GetHeader("X-API-Key") != "" {
			return authenticationResult{}, &common.APIKeyAuthNotAllowedError{}
		}
		return authenticationResult{}, err
	}

	// Attempt API-key authentication after JWT reports that the request is not signed in
	userID, isAdmin, err = m.apiKeyMiddleware.Verify(c, m.options.AdminRequired)
	if err == nil {
		return authenticationResult{UserID: userID, IsAdmin: isAdmin}, nil
	}
	if m.options.SuccessOptional {
		return authenticationResult{}, nil
	}
	return authenticationResult{}, err
}

func (m *AuthMiddleware) securityRequirements() []map[string][]string {
	requirements := []map[string][]string{
		{"BearerAuth": {}},
		{"SessionCookie": {}},
	}
	if m.options.AllowApiKeyAuth {
		requirements = append(requirements, map[string][]string{"ApiKeyAuth": {}})
	}
	if m.options.SuccessOptional {
		requirements = append([]map[string][]string{{}}, requirements...)
	}
	return requirements
}

func setGinAuthentication(c *gin.Context, result authenticationResult) {
	c.Set("userID", result.UserID)
	c.Set("userIsAdmin", result.IsAdmin)
	c.Set("authenticationMethod", result.AuthenticationMethod)
	c.Set("authenticationTime", result.AuthenticationTime)
}
