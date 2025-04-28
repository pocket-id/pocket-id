package bootstrap

import (
	"context"

	_ "github.com/golang-migrate/migrate/v4/source/file"

	"github.com/pocket-id/pocket-id/backend/internal/service"
	"github.com/pocket-id/pocket-id/backend/internal/utils/signals"
)

func Bootstrap() {
	// Get a context that is canceled when the application is stopping
	ctx := signals.SignalContext(context.Background())

	initApplicationImages()

	migrateConfigDBConnstring()

	db := newDatabase()
	appConfigService := service.NewAppConfigService(ctx, db)

	migrateKey()

	initRouter(ctx, db, appConfigService)
}
