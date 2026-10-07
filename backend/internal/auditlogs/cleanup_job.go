package auditlogs

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/italypaleale/francis/builtin/cronjob"
	"gorm.io/gorm"

	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
)

const (
	// cleanupJobInterval is how often each audit cleanup job runs
	cleanupJobInterval = 24 * time.Hour
	// cleanupJobJitter spreads each occurrence around its scheduled time, so the cleanup jobs don't all hit the database at once
	cleanupJobJitter = 5 * time.Minute
)

type cleanupJobs struct {
	db            *gorm.DB
	retentionDays int
}

// newCleanupJobs returns the cron job actors for audit retention
func newCleanupJobs(db *gorm.DB, retentionDays int) ([]*cronjob.CronJob, error) {
	jobs := &cleanupJobs{db: db, retentionDays: retentionDays}

	clearAuditLogs, err := newCleanupJob("ClearAuditLogs", jobs.clearAuditLogs)
	if err != nil {
		return nil, err
	}

	return []*cronjob.CronJob{clearAuditLogs}, nil
}

// newCleanupJob applies the shared daily schedule to each audit cleanup
func newCleanupJob(name string, fn func(context.Context) error) (*cronjob.CronJob, error) {
	cronActor, err := cronjob.New(
		name,
		cronjob.WithJob(fn),
		cronjob.WithInterval(cleanupJobInterval),
		cronjob.WithJitter(cleanupJobJitter),
		// Also run right after the job is first registered, so rows that expired while Pocket ID wasn't running are removed at startup
		cronjob.WithImmediate(),
		cronjob.WithLogger(slog.Default()),
	)
	if err != nil {
		return nil, fmt.Errorf("error creating %s cron job: %w", name, err)
	}

	return cronActor, nil
}

// clearAuditLogs deletes audit logs older than the configured retention window
func (j *cleanupJobs) clearAuditLogs(ctx context.Context) error {
	cutoff := time.Now().AddDate(0, 0, -j.retentionDays)

	st := j.db.
		WithContext(ctx).
		Delete(&AuditLog{}, "created_at < ?", datatype.DateTime(cutoff))
	if st.Error != nil {
		return fmt.Errorf("failed to delete old audit logs: %w", st.Error)
	}

	slog.InfoContext(ctx, "Deleted old audit logs", slog.Int64("count", st.RowsAffected))

	return nil
}
