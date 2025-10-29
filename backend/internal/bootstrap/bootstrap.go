package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/pocket-id/pocket-id/backend/internal/service"

	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/job"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
)

func Bootstrap(ctx context.Context) error {
	var shutdownFns []utils.Service
	defer func() { //nolint:contextcheck
		// Invoke all shutdown functions on exit
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := utils.NewServiceRunner(shutdownFns...).Run(shutdownCtx); err != nil {
			slog.Error("Error during graceful shutdown", "error", err)
		}
	}()

	// Initialize the observability stack, including the logger, distributed tracing, and metrics
	shutdownFns, httpClient, err := initObservability(ctx, common.EnvConfig.MetricsEnabled, common.EnvConfig.TracingEnabled)
	if err != nil {
		return fmt.Errorf("failed to initialize OpenTelemetry: %w", err)
	}
	slog.InfoContext(ctx, "Pocket ID is starting")

	imageExtensions, err := initApplicationImages()
	if err != nil {
		return fmt.Errorf("failed to initialize application images: %w", err)
	}

	// Connect to the database
	db, err := NewDatabase()
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}

	// Create all services
	svc, err := initServices(ctx, db, httpClient, imageExtensions)
	if err != nil {
		return fmt.Errorf("failed to initialize services: %w", err)
	}

	opCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	if err := svc.appLockService.Acquire(opCtx, common.EnvConfig.ForcefullyAcquireLock); err != nil {
		if errors.Is(err, service.ErrLockUnavailable) {
			return fmt.Errorf("only one Pocket ID instance can run at the same time. Set FORCEFULLY_ACQUIRE_LOCK=true to kill the other instance and proceed")
		}
		return fmt.Errorf("failed to acquire application lock: %w", err)
	}

	shutdownFn := func(shutdownCtx context.Context) error {
		if err := svc.appLockService.Release(shutdownCtx); err != nil {
			return fmt.Errorf("failed to release application lock: %w", err)
		}
		return nil
	}
	shutdownFns = append(shutdownFns, shutdownFn)

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
	services := []utils.Service{svc.appLockService.RunRenewal, router}

	if common.EnvConfig.AppEnv != "test" {
		services = append(services, scheduler.Run)
	}

	err = utils.NewServiceRunner(services...).Run(ctx)
	if err != nil {
		return fmt.Errorf("failed to run services: %w", err)
	}

	return nil
}
