package job

import (
	"context"
	"log"

	"github.com/go-co-op/gocron/v2"
	"github.com/pocket-id/pocket-id/backend/internal/service"
)

type ApiKeyEmailJobs struct {
	apiKeyService    *service.ApiKeyService
	appConfigService *service.AppConfigService
}

func RegisterApiKeyExpiryJob(ctx context.Context, apiKeyService *service.ApiKeyService, appConfigService *service.AppConfigService) {
	jobs := &ApiKeyEmailJobs{
		apiKeyService:    apiKeyService,
		appConfigService: appConfigService,
	}

	scheduler, err := gocron.NewScheduler()
	if err != nil {
		log.Fatalf("Failed to create a new scheduler: %v", err)
	}

	registerJob(ctx, scheduler, "ExpiredApiKeyEmailJob", "0 0 * * *", jobs.checkAndNotifyExpiringApiKeys)

	scheduler.Start()
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
