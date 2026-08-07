package appconfig

import (
	"context"
)

// AppConfigResolver loads the current application configuration, so handlers can pass it explicitly to the service methods that need it
type AppConfigResolver interface {
	GetConfig(ctx context.Context) (*AppConfigModel, error)
}
