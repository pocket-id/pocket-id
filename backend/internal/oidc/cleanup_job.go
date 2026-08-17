package oidc

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/italypaleale/francis/builtin/cronjob"
	"gorm.io/gorm"
)

const (
	// cleanupJobInterval is how often each of the OIDC cleanup jobs runs
	cleanupJobInterval = 24 * time.Hour
	// cleanupJobJitter spreads each occurrence around its scheduled time, so the cleanup jobs don't all hit the database at once
	cleanupJobJitter = 5 * time.Minute
)

type cleanupJobs struct {
	db *gorm.DB
}

// newCleanupJobs returns the cron job actors that delete expired OIDC rows from the database
func newCleanupJobs(db *gorm.DB) ([]*cronjob.CronJob, error) {
	jobs := &cleanupJobs{db: db}

	// Create the built-in actor for the ClearOAuth2Sessions job
	clearOAuth2Sessions, err := newCleanupJob("ClearOAuth2Sessions", jobs.clearOAuth2Sessions)
	if err != nil {
		return nil, err
	}

	// Create the built-in actor for the ClearOAuth2JTIs job
	clearOAuth2JTIs, err := newCleanupJob("ClearOAuth2JTIs", jobs.clearOAuth2JTIs)
	if err != nil {
		return nil, err
	}

	// Create the built-in actor for the ClearInteractionSessions job
	clearInteractionSessions, err := newCleanupJob("ClearInteractionSessions", jobs.clearInteractionSessions)
	if err != nil {
		return nil, err
	}

	return []*cronjob.CronJob{clearOAuth2Sessions, clearOAuth2JTIs, clearInteractionSessions}, nil
}

// newCleanupJob creates a cron job actor that runs one of the OIDC cleanups on a daily schedule
func newCleanupJob(name string, fn func(ctx context.Context) error) (*cronjob.CronJob, error) {
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

// clearOAuth2Sessions deletes expired OAuth2 sessions.
// Invalidated sessions are kept until their original expiry: see cleanupExpiredOAuth2Sessions.
func (j *cleanupJobs) clearOAuth2Sessions(ctx context.Context) error {
	count, err := cleanupExpiredOAuth2Sessions(ctx, j.db)
	if err != nil {
		return fmt.Errorf("failed to clean OAuth2 sessions: %w", err)
	}

	slog.InfoContext(ctx, "Cleaned OAuth2 sessions", slog.Int64("count", count))

	return nil
}

// clearOAuth2JTIs deletes expired JWT IDs used for client assertion replay protection.
func (j *cleanupJobs) clearOAuth2JTIs(ctx context.Context) error {
	count, err := cleanupExpiredClientAssertionJTIs(ctx, j.db)
	if err != nil {
		return fmt.Errorf("failed to clean OAuth2 client assertion JTIs: %w", err)
	}

	slog.InfoContext(ctx, "Cleaned OAuth2 client assertion JTIs", slog.Int64("count", count))

	return nil
}

// clearInteractionSessions deletes abandoned OIDC interaction sessions.
func (j *cleanupJobs) clearInteractionSessions(ctx context.Context) error {
	count, err := cleanupAbandonedInteractionSessions(ctx, j.db)
	if err != nil {
		return fmt.Errorf("failed to clean interaction sessions: %w", err)
	}

	slog.InfoContext(ctx, "Cleaned interaction sessions", slog.Int64("count", count))

	return nil
}
