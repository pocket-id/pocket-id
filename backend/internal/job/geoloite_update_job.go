package job

import (
	"context"
	"log"
	"time"

	"github.com/pocket-id/pocket-id/backend/internal/service"
)

type GeoLiteUpdateJobs struct {
	geoLiteService *service.GeoLiteService
}

func (s *Scheduler) RegisterGeoLiteUpdateJobs(ctx context.Context, geoLiteService *service.GeoLiteService) error {
	// Check if the service needs periodic updating
	if !geoLiteService.HasBackgroundService() {
		// Nothing to do
		return nil
	}

	jobs := &GeoLiteUpdateJobs{geoLiteService: geoLiteService}

	// Register the job to run every day, at 5 minutes past midnight
	err := s.registerJob(ctx, "UpdateGeoLiteDB", "5 * */1 * *", jobs.updateGoeLiteDB)
	if err != nil {
		return err
	}

	// Run the job immediately on startup, with a 1s delay
	go func() {
		time.Sleep(time.Second)
		err = jobs.updateGoeLiteDB(ctx)
		if err != nil {
			// Log the error only, but don't return it
			log.Printf("Failed to Update GeoLite database: %v", err)
		}
	}()

	return nil
}

func (j *GeoLiteUpdateJobs) updateGoeLiteDB(ctx context.Context) error {
	return j.geoLiteService.UpdateDatabase(ctx)
}
