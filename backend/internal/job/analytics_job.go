package job

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	backoff "github.com/cenkalti/backoff/v5"
	"github.com/italypaleale/francis/builtin/cronjob"

	"github.com/pocket-id/pocket-id/backend/internal/common"
)

const heartbeatUrl = "https://analytics.pocket-id.org/heartbeat"

// GetAnalyticsJob returns the CronJob actor
func GetAnalyticsJob(httpClient *http.Client, instanceID string) (*cronjob.CronJob, error) {
	// Skip if analytics are disabled or not in production environment
	if common.EnvConfig.AnalyticsDisabled || !common.EnvConfig.AppEnv.IsProduction() {
		return nil, nil
	}

	job := &AnalyticsJob{
		httpClient: httpClient,
	}
	err := job.createBody(instanceID)
	if err != nil {
		return nil, fmt.Errorf("error pre-computing request body: %w", err)
	}

	// Create the built-in actor
	cj, err := cronjob.New(
		"Analytics",
		cronjob.WithJob(job.sendHeartbeat),
		// Run every 24 hours
		cronjob.WithInterval(24*time.Hour),
		// Run immediately upon registration too
		cronjob.WithImmediate(),
		cronjob.WithLogger(slog.Default()),
	)
	if err != nil {
		return nil, fmt.Errorf("error creating Analytics job: %w", err)
	}

	return cj, nil
}

type AnalyticsJob struct {
	httpClient *http.Client
	body       []byte
}

// createBody pre-computes the body for all requests
func (j *AnalyticsJob) createBody(instanceID string) error {
	body, err := json.Marshal(struct {
		Version    string `json:"version"`
		InstanceID string `json:"instance_id"`
	}{
		Version:    common.Version,
		InstanceID: instanceID,
	})
	if err != nil {
		return fmt.Errorf("failed to marshal heartbeat body: %w", err)
	}

	// Set the body in the object
	j.body = body

	return nil
}

// sendHeartbeat sends a heartbeat to the analytics service
func (j *AnalyticsJob) sendHeartbeat(parentCtx context.Context) error {
	// Skip if analytics are disabled or not in production environment
	if common.EnvConfig.AnalyticsDisabled || !common.EnvConfig.AppEnv.IsProduction() {
		return nil
	}

	// Use a backoff to retry
	_, err := backoff.Retry(
		parentCtx,
		func() (struct{}, error) {
			ctx, cancel := context.WithTimeout(parentCtx, 20*time.Second)
			defer cancel()
			req, err := http.NewRequestWithContext(ctx, http.MethodPost, heartbeatUrl, bytes.NewReader(j.body))
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
