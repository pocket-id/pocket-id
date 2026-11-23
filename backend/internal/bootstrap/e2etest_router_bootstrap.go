//go:build e2etest

package bootstrap

import (
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/e2etest"
)

// When building for E2E tests, add the e2etest module
func init() {
	newTestModule = func(db *gorm.DB, svc *services) (testModule, error) {
		return e2etest.New(e2etest.Dependencies{
			DB:          db,
			Actors:      svc.actors,
			AppConfig:   svc.appConfigService,
			SigningKey:  svc.signingKey,
			BrowserKey:  svc.auditLogsModule,
			LdapSyncer:  svc.ldapSyncModule,
			FileStorage: svc.fileStorage,
		})
	}
}
