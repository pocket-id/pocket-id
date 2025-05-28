package bootstrap

import (
	"context"
	"fmt"
	"net/http"

	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/job"
)

func registerScheduledJobs(ctx context.Context, db *gorm.DB, svc *services, httpClient *http.Client, scheduler *job.Scheduler) error {
	err := scheduler.RegisterLdapJobs(ctx, svc.ldapService, svc.appConfigService)
	if err != nil {
		return fmt.Errorf("failed to register LDAP jobs in scheduler: %w", err)
	}
	err = scheduler.RegisterGeoLiteUpdateJobs(ctx, svc.geoLiteService)
	if err != nil {
		return fmt.Errorf("failed to register GeoLite DB update service: %w", err)
	}
	err = scheduler.RegisterDbCleanupJobs(ctx, db)
	if err != nil {
		return fmt.Errorf("failed to register DB cleanup jobs in scheduler: %w", err)
	}
	err = scheduler.RegisterFileCleanupJobs(ctx, db)
	if err != nil {
		return fmt.Errorf("failed to register file cleanup jobs in scheduler: %w", err)
	}
	err = scheduler.RegisterApiKeyExpiryJob(ctx, svc.apiKeyService, svc.appConfigService)
	if err != nil {
		return fmt.Errorf("failed to register API key expiration jobs in scheduler: %w", err)
	}
	err = scheduler.RegisterAnalyticsJob(ctx, svc.appConfigService, httpClient)
	if err != nil {
		return fmt.Errorf("failed to register analytics job in scheduler: %w", err)
	}

	return nil
}
