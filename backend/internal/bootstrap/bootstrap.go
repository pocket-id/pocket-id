package bootstrap

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	_ "github.com/golang-migrate/migrate/v4/source/file"

	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/job"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
)

func Bootstrap(ctx context.Context) error {
	// Initialize the observability stack, including the logger, distributed tracing, and metrics
	shutdownFns, httpClient, err := initObservability(ctx, common.EnvConfig.MetricsEnabled, common.EnvConfig.TracingEnabled)
	if err != nil {
		return fmt.Errorf("failed to initialize OpenTelemetry: %w", err)
	}
	slog.InfoContext(ctx, "Pocket ID is starting")

	err = initApplicationImages()
	if err != nil {
		return fmt.Errorf("failed to initialize application images: %w", err)
	}

	// Connect to the database
	db := NewDatabase()

	// Create all services
	svc, err := initServices(ctx, db, httpClient)
	if err != nil {
		return fmt.Errorf("failed to initialize services: %w", err)
	}

	// Init the job scheduler
	scheduler, err := job.NewScheduler()
	if err != nil {
		return fmt.Errorf("failed to create job scheduler: %w", err)
	}
	err = registerScheduledJobs(ctx, db, svc, httpClient, scheduler)
	if err != nil {
		return fmt.Errorf("failed to register scheduled jobs: %w", err)
	}

	// Init the router
	router := initRouter(db, svc)

	// Run all background services
	// This call blocks until the context is canceled
	err = utils.
		NewServiceRunner(router, scheduler.Run).
		Run(ctx)
	if err != nil {
		return fmt.Errorf("failed to run services: %w", err)
	}

	// Invoke all shutdown functions
	// We give these a timeout of 5s
	// Note: we use a background context because the run context has been canceled already
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	err = utils.
		NewServiceRunner(shutdownFns...).
		Run(shutdownCtx) //nolint:contextcheck
	if err != nil {
		slog.Error("Error shutting down services", slog.Any("error", err))
	}

	return nil
}
