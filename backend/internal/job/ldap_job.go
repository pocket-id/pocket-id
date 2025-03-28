package job

import (
	"context"
	"log"

	"github.com/go-co-op/gocron/v2"
	"github.com/pocket-id/pocket-id/backend/internal/service"
)

type LdapJobs struct {
	ldapService      *service.LdapService
	appConfigService *service.AppConfigService
}

func RegisterLdapJobs(ctx context.Context, ldapService *service.LdapService, appConfigService *service.AppConfigService) {
	jobs := &LdapJobs{ldapService: ldapService, appConfigService: appConfigService}

	scheduler, err := gocron.NewScheduler()
	if err != nil {
		log.Fatalf("Failed to create a new scheduler: %v", err)
	}

	// Register the job to run every hour
	registerJob(ctx, scheduler, "SyncLdap", "0 * * * *", jobs.syncLdap)

	// Run the job immediately on startup
	err = jobs.syncLdap(ctx)
	if err != nil {
		log.Printf("Failed to sync LDAP: %v", err)
	}

	scheduler.Start()
}

func (j *LdapJobs) syncLdap(ctx context.Context) error {
	if !j.appConfigService.DbConfig.LdapEnabled.IsTrue() {
		return nil
	}

	return j.ldapService.SyncAll(ctx)
}
