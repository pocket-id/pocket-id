package bootstrap

import (
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/stonith404/pocket-id/backend/internal/service"
)

func Bootstrap() {
	initApplicationImages()

	db := newDatabase()
	appConfigService := service.NewAppConfigService(db)

	initRouter(db, appConfigService)
}
