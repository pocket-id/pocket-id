package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync/atomic"
	"time"
)

const (
	versionCacheTTL = 15 * time.Minute
	versionCheckURL = "https://api.github.com/repos/pocket-id/pocket-id/releases/latest"
)

type cacheEntry struct {
	version     string
	lastFetched time.Time
}

type VersionService struct {
	httpClient *http.Client
	cache      atomic.Value
}

func NewVersionService(httpClient *http.Client) *VersionService {
	s := &VersionService{httpClient: httpClient}
	s.cache.Store((*cacheEntry)(nil))
	return s
}

// GetLatestVersion returns the latest available version from GitHub.
// It caches the result for a short duration to avoid excessive API calls.
func (s *VersionService) GetLatestVersion(ctx context.Context) (string, error) {
	// Serve from cache if fresh
	if entry, ok := s.cache.Load().(*cacheEntry); ok && entry != nil {
		if time.Since(entry.lastFetched) < versionCacheTTL {
			return entry.version, nil
		}
	}

	// Fetch from GitHub
	reqCtx, cancel := context.WithTimeout(ctx, 5 * time.Second)
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
		return "", err
	}

	if payload.TagName == "" {
		return "", fmt.Errorf("GitHub API returned empty tag name")
	}

	version := strings.TrimPrefix(payload.TagName, "v")

	s.cache.Store(&cacheEntry{
		version:     version,
		lastFetched: time.Now(),
	})

	return version, nil
}
