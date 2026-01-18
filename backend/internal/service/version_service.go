package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
)

const (
	versionTTL      = 15 * time.Minute
	versionCheckURL = "https://api.github.com/repos/pocket-id/pocket-id/releases/latest"
)

type VersionService struct {
	httpClient *http.Client
	cache      *utils.Cache[string]
}

func NewVersionService(httpClient *http.Client) *VersionService {
	return &VersionService{
		httpClient: httpClient,
		cache:      utils.New[string](versionTTL),
	}
}

func (s *VersionService) GetLatestVersion(ctx context.Context) (string, error) {
	if common.EnvConfig.VersionCheckDisabled {
		return "", nil
	}

	version, err := s.cache.GetOrFetch(ctx, func(ctx context.Context) (string, error) {
		reqCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, versionCheckURL, nil)
		if err != nil {
			return "", fmt.Errorf("create GitHub request: %w", err)
		}

		resp, err := s.httpClient.Do(req)
		if err != nil {
			return "", fmt.Errorf("get latest tag: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return "", fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
		}

		var payload struct {
			TagName string `json:"tag_name"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
			return "", fmt.Errorf("decode payload: %w", err)
		}

		if payload.TagName == "" {
			return "", errors.New("GitHub API returned empty tag name")
		}

		return strings.TrimPrefix(payload.TagName, "v"), nil
	})

	var staleErr *utils.ErrStale
	if errors.As(err, &staleErr) {
		slog.Warn("Failed to fetch latest version, returning stale cache", "error", staleErr.Err)
		return version, nil
	}

	return version, err
}
