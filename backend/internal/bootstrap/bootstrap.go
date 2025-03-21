package bootstrap

import (
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/pocket-id/pocket-id/backend/internal/service"
)

func Bootstrap() {
	initApplicationImages()

	db := newDatabase()
	appConfigService := service.NewAppConfigService(db)

	migrateKey()

	initRouter(db, appConfigService)
}
