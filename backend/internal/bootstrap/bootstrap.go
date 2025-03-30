package bootstrap

import (
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/pocket-id/pocket-id/backend/internal/service"
)

func Bootstrap() {
	initApplicationImages()

	migrateConfigDBConnstring()

	db := newDatabase()
	appConfigService := service.NewAppConfigService(db)

	migrateKey()

	migrateProfilePictures(db)

	initRouter(db, appConfigService)
}
