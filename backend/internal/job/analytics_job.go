package job

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/service"
	"net/http"
)

const heartbeatUrl = "https://analytics.pocket-id.org/heartbeat"

func (s *Scheduler) RegisterAnalyticsJob(ctx context.Context, appConfig *service.AppConfigService, httpClient *http.Client) error {
	jobs := &AnalyticsJob{appConfig: appConfig, httpClient: httpClient}
	return s.registerJob(ctx, "SendHeartbeat", "0 0 * * *", jobs.sendHeartbeat, true)
}

type AnalyticsJob struct {
	appConfig  *service.AppConfigService
	httpClient *http.Client
}

// sendHeartbeat sends a heartbeat to the analytics service
func (j *AnalyticsJob) sendHeartbeat(ctx context.Context) error {
	// Skip if analytics are disabled or not in production environment
	if common.EnvConfig.AnalyticsDisabled || common.EnvConfig.AppEnv != "production" {
		return nil
	}

	body := struct {
		Version    string `json:"version"`
		InstanceID string `json:"instance_id"`
	}{
		Version:    common.Version,
		InstanceID: j.appConfig.GetDbConfig().InstanceID.Value,
	}
	bodyBytes, err := json.Marshal(body)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, heartbeatUrl, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return fmt.Errorf("failed to create heartbeat request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := j.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send heartbeat request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("heartbeat request failed with status code: %d", resp.StatusCode)
	}

	return nil

}
