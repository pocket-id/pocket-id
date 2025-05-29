package job

import (
	"context"
	"time"

	"github.com/go-co-op/gocron/v2"

	"github.com/pocket-id/pocket-id/backend/internal/service"
)

type LdapJobs struct {
	ldapService      *service.LdapService
	appConfigService *service.AppConfigService
}

func (s *Scheduler) RegisterLdapJobs(ctx context.Context, ldapService *service.LdapService, appConfigService *service.AppConfigService) error {
	jobs := &LdapJobs{ldapService: ldapService, appConfigService: appConfigService}

	// Register the job to run every hour
	return s.registerJob(ctx, "SyncLdap", gocron.DurationJob(time.Hour), jobs.syncLdap, true)
}

func (j *LdapJobs) syncLdap(ctx context.Context) error {
	if !j.appConfigService.GetDbConfig().LdapEnabled.IsTrue() {
		return nil
	}

	return j.ldapService.SyncAll(ctx)
}
