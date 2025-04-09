package service

import (
	"sync/atomic"

	"github.com/pocket-id/pocket-id/backend/internal/model"
)

// NewTestAppConfigService is a function used by tests to create AppConfigService objects with pre-defined configuration values
func NewTestAppConfigService(config *model.AppConfig) *AppConfigService {
	service := &AppConfigService{
		dbConfig: atomic.Pointer[model.AppConfig]{},
	}
	service.dbConfig.Store(config)

	return service
}
