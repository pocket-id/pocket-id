package webauthn

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/go-webauthn/webauthn/metadata"
	"github.com/go-webauthn/webauthn/metadata/providers/cached"
	"github.com/go-webauthn/webauthn/metadata/providers/memory"
)

// fidoMetadataPath is where the FIDO Metadata Service blob is cached across restarts
const fidoMetadataPath = "data/fido-mds.jwt"

// fidoMetadataRetryInterval spaces out downloads after a failure, because the FIDO Metadata Service rate-limits them
const fidoMetadataRetryInterval = time.Hour

// fidoMetadata loads the FIDO Metadata Service blob that attestations are verified against
// It is loaded on first use, so instances that don't require attestation never download it
type fidoMetadata struct {
	httpClient *http.Client
	// url is where the blob is downloaded from, and empty never downloads it
	url  string
	path string

	mu       sync.Mutex
	provider metadata.Provider
	// refreshAt is when the loaded blob is due for its next update, or when a failed download may be retried
	refreshAt time.Time
	// refreshing is set while a background refresh downloads a new blob
	refreshing bool
	// downloadErr explains why metadata is missing while a failed download waits to be retried
	downloadErr error
}

func (m *fidoMetadata) get() (metadata.Provider, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Serve loaded metadata right away and refresh it in the background once it is due, so no registration waits for a download
	refreshDue := m.url != "" && !time.Now().Before(m.refreshAt)
	if m.provider != nil {
		if refreshDue && !m.refreshing {
			m.refreshing = true
			go m.refresh()
		}
		return m.provider, nil
	}

	// Without metadata in memory the registration has to wait for the cached blob, which downloads a new one when it is missing or outdated
	if refreshDue {
		m.finishRefresh(m.load(true))
		if m.provider != nil {
			return m.provider, nil
		}
	}

	// Without metadata in memory, fall back to an outdated blob on disk rather than refusing every registration
	// The provider downloads even with updates disabled when the file is missing, so that case is ruled out first
	if _, err := os.Stat(m.path); err != nil {
		if m.url == "" {
			return nil, fmt.Errorf("FIDO metadata is unavailable because downloads are disabled and it wasn't placed at %s", m.path)
		}
		return nil, fmt.Errorf("FIDO metadata is unavailable: %w", m.downloadErr)
	}
	provider, _, err := m.load(false)
	if err != nil {
		// The provider reads the cached blob before it would download, so an unreadable one, e.g. after its signing certificate expired, would block every refresh
		// Removing it lets the next registration download a new blob, and no download was attempted for this one since the read failed first
		m.refreshAt = time.Time{}
		return nil, fmt.Errorf("failed to load the cached FIDO metadata: %w", errors.Join(err, os.Remove(m.path)))
	}
	m.provider = provider
	return provider, nil
}

// refresh downloads a new blob while registrations keep using the loaded one
func (m *fidoMetadata) refresh() {
	provider, nextUpdate, err := m.load(true)

	m.mu.Lock()
	defer m.mu.Unlock()
	m.refreshing = false
	m.finishRefresh(provider, nextUpdate, err)
}

// finishRefresh keeps a loaded blob until its next update, or waits before retrying after a failed one
func (m *fidoMetadata) finishRefresh(provider metadata.Provider, nextUpdate time.Time, err error) {
	now := time.Now()
	if err != nil {
		m.refreshAt, m.downloadErr = now.Add(fidoMetadataRetryInterval), err
		slog.Warn("Failed to refresh the FIDO metadata", slog.Any("error", err))
		return
	}

	// The service may still serve the previous blob past its announced next update, so refreshes stay spaced out regardless
	m.provider, m.refreshAt = provider, nextUpdate
	if retryAt := now.Add(fidoMetadataRetryInterval); m.refreshAt.Before(retryAt) {
		m.refreshAt = retryAt
	}
}

func (m *fidoMetadata) load(update bool) (provider metadata.Provider, nextUpdate time.Time, err error) {
	err = os.MkdirAll(filepath.Dir(m.path), 0700)
	if err != nil {
		return nil, time.Time{}, err
	}

	// The memory provider's defaults require an entry for the AAGUID, a chain to its roots, an undesired-free status and a listed attestation type
	opts := []cached.Option{
		cached.WithPath(m.path),
		cached.WithUpdate(update),
		cached.WithClient(m.httpClient),
		cached.WithNew(func(mds *metadata.Metadata) (metadata.Provider, error) {
			nextUpdate = mds.Parsed.NextUpdate
			return memory.New(memory.WithMetadata(mds.ToMap()))
		}),
	}
	if m.url != "" {
		opts = append(opts, cached.WithMetadataURL(m.url))
	}

	provider, err = cached.New(opts...)
	return provider, nextUpdate, err
}
