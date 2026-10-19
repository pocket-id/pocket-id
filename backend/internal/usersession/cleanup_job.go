package usersession

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/italypaleale/francis/builtin/cronjob"
)

const (
	cleanupJobInterval = 24 * time.Hour
	cleanupJobJitter   = 5 * time.Minute
)

func newCleanupJob(service *Service) (*cronjob.CronJob, error) {
	cronActor, err := cronjob.New(
		"ClearUserSessions",
		cronjob.WithJob(func(ctx context.Context) error {
			count, err := service.cleanupExpired(ctx)
			if err != nil {
				return fmt.Errorf("failed to clean expired user sessions: %w", err)
			}
			slog.InfoContext(ctx, "Cleaned expired user sessions", slog.Int64("count", count))
			return nil
		}),
		cronjob.WithInterval(cleanupJobInterval),
		cronjob.WithJitter(cleanupJobJitter),
		// Run at startup too, to remove rows that expired while Pocket ID wasn't running
		cronjob.WithImmediate(),
		cronjob.WithLogger(slog.Default()),
	)
	if err != nil {
		return nil, fmt.Errorf("error creating ClearUserSessions cron job: %w", err)
	}

	return cronActor, nil
}
