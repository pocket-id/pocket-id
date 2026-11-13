//go:build e2etest

package bootstrap

import (
	"log/slog"
	"os"

	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/authz"
	"github.com/pocket-id/pocket-id/backend/internal/e2etest"
)

// When building for E2E tests, add the e2etest module
func init() {
	registerTestControllers = []func(apiRouter *authz.Router, db *gorm.DB, svc *services){
		func(apiRouter *authz.Router, db *gorm.DB, svc *services) {
			testModule, err := e2etest.New(e2etest.Dependencies{
				DB:          db,
				Actors:      svc.actors,
				AppConfig:   svc.appConfigService,
				SigningKey:  svc.signingKey,
				BrowserKey:  svc.auditLogsModule,
				LdapSyncer:  svc.ldapSyncModule,
				FileStorage: svc.fileStorage,
			})
			if err != nil {
				slog.Error("Failed to initialize test module", slog.Any("error", err))
				os.Exit(1)
				return
			}

			testModule.RegisterRoutes(apiRouter.Public())
		},
	}
}
