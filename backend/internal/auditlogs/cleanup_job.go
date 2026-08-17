package auditlogs

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/italypaleale/francis/builtin/cronjob"
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/model"
	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
)

const (
	// cleanupJobInterval is how often the audit log cleanup job runs
	cleanupJobInterval = 24 * time.Hour
	// cleanupJobJitter spreads each occurrence around its scheduled time, so the cleanup jobs don't all hit the database at once
	cleanupJobJitter = 5 * time.Minute
)

type cleanupJob struct {
	db            *gorm.DB
	retentionDays int
}

// newCleanupJob returns the cron job actor that deletes audit logs past the retention window
func newCleanupJob(db *gorm.DB, retentionDays int) (*cronjob.CronJob, error) {
	job := &cleanupJob{
		db:            db,
		retentionDays: retentionDays,
	}

	cronActor, err := cronjob.New(
		"ClearAuditLogs",
		cronjob.WithJob(job.clearAuditLogs),
		cronjob.WithInterval(cleanupJobInterval),
		cronjob.WithJitter(cleanupJobJitter),
		// Also run right after the job is first registered, so rows that aged out while Pocket ID wasn't running are removed at startup
		cronjob.WithImmediate(),
		cronjob.WithLogger(slog.Default()),
	)
	if err != nil {
		return nil, fmt.Errorf("error creating audit log cleanup cron job: %w", err)
	}

	return cronActor, nil
}

// clearAuditLogs deletes audit logs older than the configured retention window
func (j *cleanupJob) clearAuditLogs(ctx context.Context) error {
	cutoff := time.Now().AddDate(0, 0, -j.retentionDays)

	st := j.db.
		WithContext(ctx).
		Delete(&model.AuditLog{}, "created_at < ?", datatype.DateTime(cutoff))
	if st.Error != nil {
		return fmt.Errorf("failed to delete old audit logs: %w", st.Error)
	}

	slog.InfoContext(ctx, "Deleted old audit logs", slog.Int64("count", st.RowsAffected))

	return nil
}
