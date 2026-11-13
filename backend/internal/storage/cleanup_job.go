package storage

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/italypaleale/francis/builtin/cronjob"
)

// NewTempFileCleanupJob returns the cron job actor that deletes temporary files left behind by failed atomic writes of the file system storage
func NewTempFileCleanupJob(fileStorage FileStorage) (*cronjob.CronJob, error) {
	job := &tempFileCleanupJob{fileStorage: fileStorage}
	cj, err := cronjob.New(
		"ClearOrphanedTempFiles",
		cronjob.WithJob(job.clearOrphanedTempFiles),
		// Run every 12 hours
		cronjob.WithInterval(12*time.Hour),
		cronjob.WithLogger(slog.Default()),
		cronjob.WithJitter(5*time.Minute),
	)
	if err != nil {
		return nil, fmt.Errorf("error creating ClearOrphanedTempFiles job: %w", err)
	}
	return cj, nil
}

type tempFileCleanupJob struct {
	fileStorage FileStorage
}

// clearOrphanedTempFiles deletes temporary files that are produced by failed atomic writes
func (j *tempFileCleanupJob) clearOrphanedTempFiles(ctx context.Context) error {
	const minAge = 10 * time.Minute

	deleted := 0
	err := j.fileStorage.Walk(ctx, "/", func(p ObjectInfo) error {
		// Only temp files
		if !strings.HasSuffix(p.Path, "-tmp") {
			return nil
		}

		if time.Since(p.ModTime) < minAge {
			return nil
		}

		rErr := j.fileStorage.Delete(ctx, p.Path)
		if rErr != nil {
			slog.ErrorContext(ctx, "Failed to delete temp file", slog.String("path", p.Path), slog.Any("error", rErr))
			return nil
		}
		deleted++
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to scan storage: %w", err)
	}

	slog.Info("Done cleaning orphaned temp files", slog.Int("count", deleted))
	return nil
}
