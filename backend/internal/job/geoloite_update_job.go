package job

import (
	"context"

	"github.com/pocket-id/pocket-id/backend/internal/service"
)

type GeoLiteUpdateJobs struct {
	geoLiteService *service.GeoLiteService
}

func (s *Scheduler) RegisterGeoLiteUpdateJobs(ctx context.Context, geoLiteService *service.GeoLiteService) error {
	// Check if the service needs periodic updating
	if geoLiteService.DisableUpdater() {
		// Nothing to do
		return nil
	}

	jobs := &GeoLiteUpdateJobs{geoLiteService: geoLiteService}

	// Register the job to run every day, at 5 minutes past midnight
	return s.registerJob(ctx, "UpdateGeoLiteDB", "5 * */1 * *", jobs.updateGoeLiteDB, true)
}

func (j *GeoLiteUpdateJobs) updateGoeLiteDB(ctx context.Context) error {
	return j.geoLiteService.UpdateDatabase(ctx)
}
