package bootstrap

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/job"
)

func registerScheduledJobs(ctx context.Context, db *gorm.DB, svc *services, scheduler *job.Scheduler) error {
	err := scheduler.RegisterDbCleanupJobs(ctx, db)
	if err != nil {
		return fmt.Errorf("failed to register DB cleanup jobs in scheduler: %w", err)
	}
	return nil
}
