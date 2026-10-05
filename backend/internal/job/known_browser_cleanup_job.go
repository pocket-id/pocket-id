package job

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/italypaleale/francis/builtin/cronjob"
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/model"
)

// GetKnownBrowserCleanupJob removes expired browser records even for users who no longer sign in
func GetKnownBrowserCleanupJob(db *gorm.DB) (*cronjob.CronJob, error) {
	job := &knownBrowserCleanupJob{db: db}
	return cronjob.New(
		"ClearKnownBrowsers",
		cronjob.WithJob(job.clearKnownBrowsers),
		cronjob.WithInterval(24*time.Hour),
		cronjob.WithJitter(5*time.Minute),
		cronjob.WithImmediate(),
		cronjob.WithLogger(slog.Default()),
	)
}

type knownBrowserCleanupJob struct {
	db *gorm.DB
}

func (j *knownBrowserCleanupJob) clearKnownBrowsers(ctx context.Context) error {
	// Use the same expiry boundary as sign-in recognition so valid browser cookies remain untouched
	result := j.db.WithContext(ctx).Delete(&model.KnownBrowser{}, "expires_at <= ?", time.Now().Unix())
	if result.Error != nil {
		return fmt.Errorf("failed to delete expired known browsers: %w", result.Error)
	}

	slog.InfoContext(ctx, "Deleted expired known browsers", slog.Int64("count", result.RowsAffected))
	return nil
}
