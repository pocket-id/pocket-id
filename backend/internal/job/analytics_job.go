package job

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	backoff "github.com/cenkalti/backoff/v5"
	"github.com/go-co-op/gocron/v2"

	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/service"
)

const heartbeatUrl = "https://analytics.pocket-id.org/heartbeat"

func (s *Scheduler) RegisterAnalyticsJob(ctx context.Context, appConfig *service.AppConfigService, httpClient *http.Client) error {
	// Skip if analytics are disabled or not in production environment
	if common.EnvConfig.AnalyticsDisabled || !common.EnvConfig.AppEnv.IsProduction() {
		return nil
	}

	// Send every 24 hours
	jobs := &AnalyticsJob{
		appConfig:  appConfig,
		httpClient: httpClient,
	}
	return s.registerJob(ctx, "SendHeartbeat", gocron.DurationJob(24*time.Hour), jobs.sendHeartbeat, true)
}

type AnalyticsJob struct {
	appConfig  *service.AppConfigService
	httpClient *http.Client
}

// sendHeartbeat sends a heartbeat to the analytics service
func (j *AnalyticsJob) sendHeartbeat(parentCtx context.Context) error {
	// Skip if analytics are disabled or not in production environment
	if common.EnvConfig.AnalyticsDisabled || !common.EnvConfig.AppEnv.IsProduction() {
		return nil
	}

	body, err := json.Marshal(struct {
		Version    string `json:"version"`
		InstanceID string `json:"instance_id"`
	}{
		Version:    common.Version,
		InstanceID: j.appConfig.GetDbConfig().InstanceID.Value,
	})
	if err != nil {
		return fmt.Errorf("failed to marshal heartbeat body: %w", err)
	}

	_, err = backoff.Retry(
		parentCtx,
		func() (struct{}, error) {
			ctx, cancel := context.WithTimeout(parentCtx, 20*time.Second)
			defer cancel()
			req, err := http.NewRequestWithContext(ctx, http.MethodPost, heartbeatUrl, bytes.NewReader(body))
			if err != nil {
				return struct{}{}, fmt.Errorf("failed to create request: %w", err)
			}
			req.Header.Set("Content-Type", "application/json")
			resp, err := j.httpClient.Do(req)
			if err != nil {
				return struct{}{}, fmt.Errorf("failed to send request: %w", err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				return struct{}{}, fmt.Errorf("request failed with status code: %d", resp.StatusCode)
			}
			return struct{}{}, nil
		},
		backoff.WithBackOff(backoff.NewExponentialBackOff()),
		backoff.WithMaxTries(3),
	)

	if err != nil {
		return fmt.Errorf("heartbeat request failed: %w", err)
	}

	return nil
}
