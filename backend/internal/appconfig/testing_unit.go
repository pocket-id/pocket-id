//go:build unit

// This file contains utils for unit tests and it's only built when the "unit" tag is set
package appconfig

import (
	"sync/atomic"

	"github.com/pocket-id/pocket-id/backend/internal/model"
)

// NewTestAppConfigService is a function used by tests to create AppConfigService objects with pre-defined configuration values
func NewTestAppConfigService(config *model.AppConfig) *AppConfigService {
	if config == nil {
		// If there's no config, set the default one
		config = getDefaultDbConfig()
	}

	service := &AppConfigService{
		dbConfig: atomic.Pointer[model.AppConfig]{},
	}
	service.dbConfig.Store(config)

	return service
}
