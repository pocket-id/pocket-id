package bootstrap

import (
	"context"
	"fmt"

	"github.com/pocket-id/pocket-id/backend/internal/job"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
	"gorm.io/gorm"
)

func initScheduler(ctx context.Context, db *gorm.DB, svc *services) (utils.Service, error) {
	scheduler, err := job.NewScheduler()
	if err != nil {
		return nil, fmt.Errorf("failed to create job scheduler: %w", err)
	}

	err = scheduler.RegisterLdapJobs(ctx, svc.ldapService, svc.appConfigService)
	if err != nil {
		return nil, fmt.Errorf("failed to register LDAP jobs in scheduler: %w", err)
	}
	err = scheduler.RegisterDbCleanupJobs(ctx, db)
	if err != nil {
		return nil, fmt.Errorf("failed to register DB cleanup jobs in scheduler: %w", err)
	}
	err = scheduler.RegisterFileCleanupJobs(ctx, db)
	if err != nil {
		return nil, fmt.Errorf("failed to register file cleanup jobs in scheduler: %w", err)
	}
	err = scheduler.RegisterApiKeyExpiryJob(ctx, svc.apiKeyService, svc.appConfigService)
	if err != nil {
		return nil, fmt.Errorf("failed to register API key expiration jobs in scheduler: %w", err)
	}

	// Return the service function
	return scheduler.Run, nil
}
