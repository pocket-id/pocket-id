package apikey

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/italypaleale/francis/builtin/cronjob"

	"github.com/pocket-id/pocket-id/backend/internal/appconfig"
)

type APIKeyExpiryEmailSender interface {
	SendAPIKeyExpiringSoon(ctx context.Context, dbConfig *appconfig.AppConfigModel, userFullName, userEmail, firstName, apiKeyName string, expiresAt time.Time) error
}

type expiryJob struct {
	service   *Service
	appConfig appconfig.AppConfigResolver
	email     APIKeyExpiryEmailSender
}

func newExpiryJob(service *Service, appConfig appconfig.AppConfigResolver, email APIKeyExpiryEmailSender) (*cronjob.CronJob, error) {
	job := &expiryJob{
		service:   service,
		appConfig: appConfig,
		email:     email,
	}

	cronActor, err := cronjob.New(
		"ExpiredApiKeyEmailJob",
		cronjob.WithJob(job.checkAndNotifyExpiringAPIKeys),
		// Run at midnight (± 2 minutes)
		// We want a consistent time because we are emailing users
		cronjob.WithCron("0 0 * * *"),
		cronjob.WithLogger(slog.Default()),
		cronjob.WithJitter(2*time.Minute),
	)
	if err != nil {
		return nil, fmt.Errorf("error creating API key expiration cron job: %w", err)
	}

	return cronActor, nil
}

func (j *expiryJob) checkAndNotifyExpiringAPIKeys(ctx context.Context) error {
	dbConfig, err := j.appConfig.GetConfig(ctx)
	if err != nil {
		return fmt.Errorf("error loading app config: %w", err)
	}

	// Skip the database query when expiration notifications are disabled
	if !dbConfig.EmailApiKeyExpirationEnabled.IsTrue() {
		return nil
	}

	// Load API keys that entered the notification window since the last occurrence
	apiKeys, err := j.service.ListExpiringApiKeys(ctx, 7)
	if err != nil {
		return fmt.Errorf("failed to list expiring API keys: %w", err)
	}

	for _, key := range apiKeys {
		if key.User.Email == nil {
			continue
		}

		// Continue processing other recipients when one delivery fails
		err = j.email.SendAPIKeyExpiringSoon(
			ctx,
			dbConfig,
			key.User.FullName(),
			*key.User.Email,
			key.User.FirstName,
			key.Name,
			key.ExpiresAt.ToTime(),
		)
		if err != nil {
			slog.ErrorContext(ctx,
				"Failed to send expiring API key notification email",
				slog.String("key", key.ID),
				slog.String("user", key.User.ID),
				slog.Any("error", err),
			)
			continue
		}

		// Mark successful deliveries so future cron occurrences do not resend them
		err = j.service.MarkExpirationEmailSent(ctx, key.ID)
		if err != nil {
			slog.ErrorContext(ctx,
				"Failed to record that the expiration email was sent",
				slog.String("key", key.ID),
				slog.Any("error", err),
			)
		}
	}

	return nil
}
