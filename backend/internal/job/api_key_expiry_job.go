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

	if err := jobs.checkAndNotifyExpiringApiKeys(ctx); err != nil {
		log.Printf("Failed to send expired api key emails: %v", err)
	}

	scheduler.Start()
}

func (j *ApiKeyEmailJobs) checkAndNotifyExpiringApiKeys(ctx context.Context) error {
	log.Printf("Running API key expiry check...")

	apiKeys, err := j.apiKeyService.ListExpiringApiKeys(ctx, 7)
	if err != nil {
		log.Printf("ExpiredApiKeyEmailJob: query failed: %v", err)
		return err
	}

	log.Printf("ExpiredApiKeyEmailJob: found %d keys expiring in the next 7 days", len(apiKeys))
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
