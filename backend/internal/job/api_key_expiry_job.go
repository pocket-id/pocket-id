package job

import (
	"context"
	"log"

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

	return s.registerJob(ctx, "ExpiredApiKeyEmailJob", "0 0 * * *", jobs.checkAndNotifyExpiringApiKeys, false)
}

func (j *ApiKeyEmailJobs) checkAndNotifyExpiringApiKeys(ctx context.Context) error {
	// Skip if the feature is disabled
	if !j.appConfigService.GetDbConfig().EmailApiKeyExpirationEnabled.IsTrue() {
		return nil
	}

	apiKeys, err := j.apiKeyService.ListExpiringApiKeys(ctx, 7)
	if err != nil {
		log.Printf("Failed to list expiring API keys: %v", err)
		return err
	}

	for _, key := range apiKeys {
		if key.User.Email == "" {
			continue
		}
		if err := j.apiKeyService.SendApiKeyExpiringSoonEmail(ctx, key); err != nil {
			log.Printf("Failed to send email for key %s: %v", key.ID, err)
		}
	}
	return nil
}
