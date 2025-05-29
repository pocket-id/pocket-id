package job

import (
	"context"
	"time"

	"github.com/go-co-op/gocron/v2"

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

	// Run every 24 hours (and right away)
	return s.registerJob(ctx, "UpdateGeoLiteDB", gocron.DurationJob(24*time.Hour), jobs.updateGoeLiteDB, true)
}

func (j *GeoLiteUpdateJobs) updateGoeLiteDB(ctx context.Context) error {
	return j.geoLiteService.UpdateDatabase(ctx)
}
