package job

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/go-co-op/gocron/v2"

	"github.com/pocket-id/pocket-id/backend/internal/apikey"
	"github.com/pocket-id/pocket-id/backend/internal/appconfig"
	"github.com/pocket-id/pocket-id/backend/internal/service"
	"github.com/pocket-id/pocket-id/backend/internal/utils/email"
)

type ApiKeyEmailJobs struct {
	apiKeyModule     *apikey.Module
	appConfigService *appconfig.AppConfigService
	emailService     *service.EmailService
}

func (s *Scheduler) RegisterApiKeyExpiryJob(ctx context.Context, apiKeyModule *apikey.Module, appConfigService *appconfig.AppConfigService, emailService *service.EmailService) error {
	jobs := &ApiKeyEmailJobs{
		apiKeyModule:     apiKeyModule,
		appConfigService: appConfigService,
		emailService:     emailService,
	}

	// Send every day at midnight
	return s.RegisterJob(ctx, "ExpiredApiKeyEmailJob", gocron.CronJob("0 0 * * *", false), jobs.checkAndNotifyExpiringApiKeys, service.RegisterJobOpts{})
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

		err = service.SendEmail(ctx, j.emailService, email.Address{
			Name:  key.User.FullName(),
			Email: *key.User.Email,
		}, service.ApiKeyExpiringSoonTemplate, &service.ApiKeyExpiringSoonTemplateData{
			Name:       key.User.FirstName,
			ApiKeyName: key.Name,
			ExpiresAt:  key.ExpiresAt.ToTime(),
		})
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
