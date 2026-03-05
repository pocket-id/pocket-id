package job

import (
	"context"
	"time"

	"github.com/pocket-id/pocket-id/backend/internal/service"
)

type LdapJobs struct {
	ldapService      *service.LdapService
	appConfigService *service.AppConfigService
}

func (s *Scheduler) RegisterLdapJobs(ctx context.Context, ldapService *service.LdapService, appConfigService *service.AppConfigService) error {
	jobs := &LdapJobs{ldapService: ldapService, appConfigService: appConfigService}

	// Register the job to run every hour (with some jitter)
	return s.RegisterJob(ctx, "SyncLdap", jobDefWithJitter(time.Hour), jobs.syncLdap, true)
}

func (j *LdapJobs) syncLdap(ctx context.Context) error {
	if !j.appConfigService.GetDbConfig().LdapEnabled.IsTrue() {
		return nil
	}

	return j.ldapService.SyncAll(ctx)
}
