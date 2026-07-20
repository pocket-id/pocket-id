//go:build unit

// This file contains utils for unit tests and it's only built when the "unit" tag is set
package appconfig

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

// NewTestConfig returns an application configuration for use in tests, falling back to the default configuration when none is provided
func NewTestConfig(config *AppConfigModel) *AppConfigModel {
	if config == nil {
		config = getDefaultConfig()
	}

	return config
}
