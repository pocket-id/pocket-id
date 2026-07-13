package appconfig

import (
	"context"
	"errors"
	"sync"

	"github.com/gin-gonic/gin"
)

// appConfigCtxKey is the context key used to store the AppConfigResolver in the http.Request's context
type appConfigCtxKey struct{}

type appConfigResolver func(ctx context.Context) (*AppConfigModel, error)

// AppConfigMiddleware is a Gin middleware that makes the application configuration available to all downstream handlers through the request's context
type AppConfigMiddleware struct {
	appConfigService *AppConfigService
}

func NewAppConfigMiddleware(appConfigService *AppConfigService) *AppConfigMiddleware {
	return &AppConfigMiddleware{
		appConfigService: appConfigService,
	}
}

// Add returns a Gin middleware that stores an AppConfigResolver in the http.Request's context
// The resolver loads the application configuration lazily on the first call and caches it for the duration of the request
func (m *AppConfigMiddleware) Add() gin.HandlerFunc {
	return func(c *gin.Context) {
		reqCtx := c.Request.Context()

		// Create a cache for each request in the middleware's scope, so it's unique per each request
		var (
			once sync.Once
			cfg  *AppConfigModel
			err  error
		)
		// Note: the resolver accepts a context argument, it doesn't use the request's own
		// This can be used for example for tracing
		resolver := appConfigResolver(func(ctx context.Context) (*AppConfigModel, error) {
			once.Do(func() {
				cfg, err = m.appConfigService.GetConfig(ctx)
			})
			return cfg, err
		})

		// Store the resolver in the request's context
		c.Request = c.Request.WithContext(context.WithValue(reqCtx, appConfigCtxKey{}, resolver))

		c.Next()
	}
}

// FromCtx retrieves the app config from the context
func FromCtx(ctx context.Context) (*AppConfigModel, error) {
	resolver, ok := ctx.Value(appConfigCtxKey{}).(appConfigResolver)
	if !ok || resolver == nil {
		// Indicates a development-time error
		return nil, errors.New("middleware AppConfigMiddleware was not registered for the handler")
	}

	return resolver(ctx)
}
