package job

import (
	"context"
	"fmt"
	"time"

	"github.com/pocket-id/pocket-id/backend/internal/appconfig"
	"github.com/pocket-id/pocket-id/backend/internal/service"
)

type LdapJobs struct {
	ldapService      *service.LdapService
	appConfigService *appconfig.AppConfigService
}

func (s *Scheduler) RegisterLdapJobs(ctx context.Context, ldapService *service.LdapService, appConfigService *appconfig.AppConfigService) error {
	jobs := &LdapJobs{ldapService: ldapService, appConfigService: appConfigService}

	// Register the job to run every hour (with some jitter)
	return s.RegisterJob(ctx, "SyncLdap", jobDefWithJitter(time.Hour), jobs.syncLdap, service.RegisterJobOpts{RunImmediately: true})
}

func (j *LdapJobs) syncLdap(ctx context.Context) error {
	dbConfig, err := j.appConfigService.GetConfig(ctx)
	if err != nil {
		return fmt.Errorf("error load app config: %w", err)
	}

	if !dbConfig.LdapEnabled.IsTrue() {
		return nil
	}

	return j.ldapService.SyncAll(ctx)
}
