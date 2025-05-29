package job

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/go-co-op/gocron/v2"

	"github.com/pocket-id/pocket-id/backend/internal/service"
)

type ApiKeyEmailJobs struct {
	apiKeyService    *service.ApiKeyService
	appConfigService *service.AppConfigService
}

func (s *Scheduler) RegisterApiKeyExpiryJob(ctx context.Context, apiKeyService *service.ApiKeyService, appConfigService *service.AppConfigService) error {
	jobs := &ApiKeyEmailJobs{
		apiKeyService:    apiKeyService,
		appConfigService: appConfigService,
	}

	// Send every day at midnight
	return s.registerJob(ctx, "ExpiredApiKeyEmailJob", gocron.CronJob("0 0 * * *", false), jobs.checkAndNotifyExpiringApiKeys, false)
}

func (j *ApiKeyEmailJobs) checkAndNotifyExpiringApiKeys(ctx context.Context) error {
	// Skip if the feature is disabled
	if !j.appConfigService.GetDbConfig().EmailApiKeyExpirationEnabled.IsTrue() {
		return nil
	}

	apiKeys, err := j.apiKeyService.ListExpiringApiKeys(ctx, 7)
	if err != nil {
		return fmt.Errorf("failed to list expiring API keys: %w", err)
	}

	for _, key := range apiKeys {
		if key.User.Email == "" {
			continue
		}
		err = j.apiKeyService.SendApiKeyExpiringSoonEmail(ctx, key)
		if err != nil {
			slog.ErrorContext(ctx, "Failed to send expiring API key notification email", slog.String("key", key.ID), slog.Any("error", err))
		}
	}
	return nil
}
