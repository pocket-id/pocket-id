package job

import (
	"log"

	"github.com/go-co-op/gocron/v2"
	"github.com/pocket-id/pocket-id/backend/internal/service"
)

type LdapJobs struct {
	ldapService      *service.LdapService
	appConfigService *service.AppConfigService
}

func RegisterLdapJobs(ldapService *service.LdapService, appConfigService *service.AppConfigService) {
	jobs := &LdapJobs{ldapService: ldapService, appConfigService: appConfigService}

	scheduler, err := gocron.NewScheduler()
	if err != nil {
		log.Fatalf("Failed to create a new scheduler: %s", err)
	}

	// Register the job to run every hour
	registerJob(scheduler, "SyncLdap", "0 * * * *", jobs.syncLdap)

	// Run the job immediately on startup
	if err := jobs.syncLdap(); err != nil {
		log.Printf("Failed to sync LDAP: %s", err)
	}

	scheduler.Start()
}

func (j *LdapJobs) syncLdap() error {
	if j.appConfigService.DbConfig.LdapEnabled.IsTrue() {
		return j.ldapService.SyncAll()
	}
	return nil
}
