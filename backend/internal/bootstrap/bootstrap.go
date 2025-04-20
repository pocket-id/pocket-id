package bootstrap

import (
	"context"

	_ "github.com/golang-migrate/migrate/v4/source/file"

	"github.com/pocket-id/pocket-id/backend/internal/service"
)

// Test comment to force run of actions.

func Bootstrap() {
	ctx := context.TODO()

	initApplicationImages()

	migrateConfigDBConnstring()

	db := newDatabase()
	appConfigService := service.NewAppConfigService(ctx, db)

	migrateKey()

	initRouter(ctx, db, appConfigService)
}
