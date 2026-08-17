package webauthn

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/italypaleale/francis/builtin/cronjob"
	"gorm.io/gorm"
)

const (
	// cleanupJobInterval is how often each of the WebAuthn cleanup jobs runs
	cleanupJobInterval = 24 * time.Hour
	// cleanupJobJitter spreads each occurrence around its scheduled time, so the cleanup jobs don't all hit the database at once
	cleanupJobJitter = 5 * time.Minute
)

type cleanupJobs struct {
	db *gorm.DB
}

// newCleanupJobs returns the cron job actors that delete expired WebAuthn rows from the database
func newCleanupJobs(db *gorm.DB) ([]*cronjob.CronJob, error) {
	jobs := &cleanupJobs{db: db}

	// Create the built-in actor for the ClearWebauthnSessions job
	clearWebauthnSessions, err := newCleanupJob("ClearWebauthnSessions", jobs.clearWebauthnSessions)
	if err != nil {
		return nil, err
	}

	// Create the built-in actor for the ClearReauthenticationTokens job
	clearReauthenticationTokens, err := newCleanupJob("ClearReauthenticationTokens", jobs.clearReauthenticationTokens)
	if err != nil {
		return nil, err
	}

	return []*cronjob.CronJob{clearWebauthnSessions, clearReauthenticationTokens}, nil
}

// newCleanupJob creates a cron job actor that runs one of the WebAuthn cleanups on a daily schedule
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

// clearWebauthnSessions deletes expired WebAuthn challenge sessions.
func (j *cleanupJobs) clearWebauthnSessions(ctx context.Context) error {
	count, err := cleanupExpiredSessions(ctx, j.db)
	if err != nil {
		return fmt.Errorf("failed to clean expired WebAuthn sessions: %w", err)
	}

	slog.InfoContext(ctx, "Cleaned expired WebAuthn sessions", slog.Int64("count", count))

	return nil
}

// clearReauthenticationTokens deletes expired reauthentication tokens.
func (j *cleanupJobs) clearReauthenticationTokens(ctx context.Context) error {
	count, err := cleanupExpiredReauthenticationTokens(ctx, j.db)
	if err != nil {
		return fmt.Errorf("failed to clean expired reauthentication tokens: %w", err)
	}

	slog.InfoContext(ctx, "Cleaned expired reauthentication tokens", slog.Int64("count", count))

	return nil
}
