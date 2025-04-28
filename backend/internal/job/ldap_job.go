package job

import (
	"context"
	"log"

	"github.com/pocket-id/pocket-id/backend/internal/service"
)

type LdapJobs struct {
	ldapService      *service.LdapService
	appConfigService *service.AppConfigService
}

func (s *Scheduler) RegisterLdapJobs(ctx context.Context, ldapService *service.LdapService, appConfigService *service.AppConfigService) error {
	jobs := &LdapJobs{ldapService: ldapService, appConfigService: appConfigService}

	// Register the job to run every hour
	err := s.registerJob(ctx, "SyncLdap", "0 * * * *", jobs.syncLdap)
	if err != nil {
		return err
	}

	// Run the job immediately on startup
	err = jobs.syncLdap(ctx)
	if err != nil {
		// Log the error only, but don't return it
		log.Printf("Failed to sync LDAP: %v", err)
	}

	return nil
}

func (j *LdapJobs) syncLdap(ctx context.Context) error {
	if !j.appConfigService.GetDbConfig().LdapEnabled.IsTrue() {
		return nil
	}

	return j.ldapService.SyncAll(ctx)
}
