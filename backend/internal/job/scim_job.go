package job

import (
	"context"
	"time"

	"github.com/go-co-op/gocron/v2"

	"github.com/pocket-id/pocket-id/backend/internal/service"
)

type ScimJobs struct {
	scimService *service.ScimService
}

func (s *Scheduler) RegisterScimJobs(ctx context.Context, scimService *service.ScimService) error {
	jobs := &ScimJobs{scimService: scimService}

	// Register the job to run every hour
	return s.RegisterJob(ctx, "SyncScim", gocron.DurationJob(time.Hour), jobs.SyncScim, true)
}

func (j *ScimJobs) SyncScim(ctx context.Context) error {
	return j.scimService.SyncAll(ctx)
}
