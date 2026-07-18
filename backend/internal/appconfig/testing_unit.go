//go:build unit

// This file contains utils for unit tests and it's only built when the "unit" tag is set
package appconfig

import (
	"context"
)

// NewTestAppConfigService is a function used by tests to create AppConfigService objects with pre-defined configuration values
func NewTestAppConfigService(config *AppConfigModel) *AppConfigService {
	if config == nil {
		// If there's no config, set the default one
		config = getDefaultConfig()
	}

	service := &AppConfigService{
		envConfig: config,
	}

	return service
}

// NewTestContext returns a context that resolves the provided application configuration
func NewTestContext(ctx context.Context, config *AppConfigModel) context.Context {
	if config == nil {
		config = getDefaultConfig()
	}

	resolver := appConfigResolver(func(context.Context) (*AppConfigModel, error) {
		return config, nil
	})

	return context.WithValue(ctx, appConfigCtxKey{}, resolver)
}
