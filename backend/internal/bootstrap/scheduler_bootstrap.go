package bootstrap

import (
	"context"
	"fmt"

	"github.com/pocket-id/pocket-id/backend/internal/job"
)

func registerScheduledJobs(ctx context.Context, svc *services, scheduler *job.Scheduler) error {
	err := scheduler.RegisterScimJobs(ctx, svc.scimService)
	if err != nil {
		return fmt.Errorf("failed to register SCIM scheduler job: %w", err)
	}

	return nil
}
