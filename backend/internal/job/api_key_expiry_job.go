package job

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/go-co-op/gocron/v2"

	"github.com/pocket-id/pocket-id/backend/internal/apikey"
	"github.com/pocket-id/pocket-id/backend/internal/appconfig"
)

type APIKeyExpiryEmailSender interface {
	SendAPIKeyExpiringSoon(ctx context.Context, dbConfig *appconfig.AppConfigModel, userFullName, userEmail, firstName, apiKeyName string, expiresAt time.Time) error
}

type ApiKeyEmailJobs struct {
	apiKeyModule     *apikey.Module
	appConfigService *appconfig.AppConfigService
	emailSender      APIKeyExpiryEmailSender
}

func (s *Scheduler) RegisterApiKeyExpiryJob(ctx context.Context, apiKeyModule *apikey.Module, appConfigService *appconfig.AppConfigService, emailSender APIKeyExpiryEmailSender) error {
	jobs := &ApiKeyEmailJobs{
		apiKeyModule:     apiKeyModule,
		appConfigService: appConfigService,
		emailSender:      emailSender,
	}

	// Send every day at midnight
	return s.RegisterJob(ctx, "ExpiredApiKeyEmailJob", gocron.CronJob("0 0 * * *", false), jobs.checkAndNotifyExpiringApiKeys, RegisterJobOpts{})
}

func (j *ApiKeyEmailJobs) checkAndNotifyExpiringApiKeys(ctx context.Context) error {
	dbConfig, err := j.appConfigService.GetConfig(ctx)
	if err != nil {
		return fmt.Errorf("error load app config: %w", err)
	}

	// Skip if the feature is disabled
	if !dbConfig.EmailApiKeyExpirationEnabled.IsTrue() {
		return nil
	}

	apiKeys, err := j.apiKeyModule.ListExpiringApiKeys(ctx, 7)
	if err != nil {
		return fmt.Errorf("failed to list expiring API keys: %w", err)
	}

	for _, key := range apiKeys {
		if key.User.Email == nil {
			continue
		}

		err = j.emailSender.SendAPIKeyExpiringSoon(
			ctx,
			dbConfig,
			key.User.FullName(),
			*key.User.Email,
			key.User.FirstName,
			key.Name,
			key.ExpiresAt.ToTime(),
		)
		if err != nil {
			slog.ErrorContext(ctx, "Failed to send expiring API key notification email",
				slog.String("key", key.ID),
				slog.String("user", key.User.ID),
				slog.Any("error", err),
			)
			continue
		}

		if err = j.apiKeyModule.MarkExpirationEmailSent(ctx, key.ID); err != nil {
			slog.ErrorContext(ctx, "Failed to record that the expiration email was sent",
				slog.String("key", key.ID),
				slog.Any("error", err),
			)
		}
	}
	return nil
}
