package bootstrap

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	_ "github.com/golang-migrate/migrate/v4/source/file"
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/job"
	"github.com/pocket-id/pocket-id/backend/internal/storage"
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

	db, err := NewDatabase()
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}

	fileStorage, err := InitStorage(ctx, db)
	if err != nil {
		return fmt.Errorf("failed to initialize file storage (backend: %s): %w", common.EnvConfig.FileBackend, err)
	}

	imageExtensions, err := initApplicationImages(ctx, fileStorage)
	if err != nil {
		return fmt.Errorf("failed to initialize application images: %w", err)
	}

	scheduler, err := job.NewScheduler()
	if err != nil {
		return fmt.Errorf("failed to create job scheduler: %w", err)
	}

	// Create all services
	svc, err := initServices(ctx, db, httpClient, imageExtensions, fileStorage, scheduler)
	if err != nil {
		return fmt.Errorf("failed to initialize services: %w", err)
	}

	waitUntil, err := svc.appLockService.Acquire(ctx, false)
	if err != nil {
		return fmt.Errorf("failed to acquire application lock: %w", err)
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(time.Until(waitUntil)):
	}

	shutdownFn := func(shutdownCtx context.Context) error {
		sErr := svc.appLockService.Release(shutdownCtx)
		if sErr != nil {
			return fmt.Errorf("failed to release application lock: %w", sErr)
		}
		return nil
	}
	shutdownFns = append(shutdownFns, shutdownFn)

	// Register scheduled jobs
	err = registerScheduledJobs(ctx, db, svc, httpClient, scheduler)
	if err != nil {
		return fmt.Errorf("failed to register scheduled jobs: %w", err)
	}

	// Init the router
	router, err := initRouter(db, svc)
	if err != nil {
		return fmt.Errorf("failed to initialize router: %w", err)
	}

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

func InitStorage(ctx context.Context, db *gorm.DB) (fileStorage storage.FileStorage, err error) {
	switch common.EnvConfig.FileBackend {
	case storage.TypeFileSystem:
		fileStorage, err = storage.NewFilesystemStorage(common.EnvConfig.UploadPath)
	case storage.TypeDatabase:
		fileStorage, err = storage.NewDatabaseStorage(db)
	case storage.TypeS3:
		s3Cfg := storage.S3Config{
			Bucket:                        common.EnvConfig.S3Bucket,
			Region:                        common.EnvConfig.S3Region,
			Endpoint:                      common.EnvConfig.S3Endpoint,
			AccessKeyID:                   common.EnvConfig.S3AccessKeyID,
			SecretAccessKey:               common.EnvConfig.S3SecretAccessKey,
			ForcePathStyle:                common.EnvConfig.S3ForcePathStyle,
			DisableDefaultIntegrityChecks: common.EnvConfig.S3DisableDefaultIntegrityChecks,
			Root:                          common.EnvConfig.UploadPath,
		}
		fileStorage, err = storage.NewS3Storage(ctx, s3Cfg)
	default:
		err = fmt.Errorf("unknown file storage backend: %s", common.EnvConfig.FileBackend)
	}
	if err != nil {
		return fileStorage, err
	}

	return fileStorage, nil
}
