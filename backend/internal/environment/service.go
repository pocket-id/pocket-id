package environment

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
)

const (
	versionTTL      = 15 * time.Minute
	versionCheckURL = "https://api.github.com/repos/pocket-id/pocket-id/releases/latest"
)

// warning identifies a problem with this instance that admins should fix, and the frontend shows the message for it
type warning string

const (
	// warningSQLiteOnNetworkedFilesystem means the SQLite database is on a networked filesystem, which is unsupported and can corrupt it
	warningSQLiteOnNetworkedFilesystem warning = "sqlite-on-networked-filesystem"

	// warningDatabaseMigratedByNewerVersion means a newer Pocket ID version migrated the database, so this one runs on a newer schema
	warningDatabaseMigratedByNewerVersion warning = "database-migrated-by-newer-version"
)

type Service struct {
	db         *gorm.DB
	httpClient *http.Client
	cache      *utils.Cache[string]

	sqliteOnNetworkedFilesystem bool
}

func newService(deps Dependencies) *Service {
	return &Service{
		db:                          deps.DB,
		httpClient:                  deps.HTTPClient,
		cache:                       utils.New[string](versionTTL),
		sqliteOnNetworkedFilesystem: deps.SQLiteOnNetworkedFilesystem,
	}
}

func (s *Service) GetLatestVersion(ctx context.Context) (string, error) {
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

	if staleErr, ok := errors.AsType[*utils.ErrStale](err); ok {
		slog.Warn("Failed to fetch latest version, returning stale cache", "error", staleErr.Err)
		return version, nil
	}

	return version, err
}

// Warnings returns the problems with this instance that admins should fix
// They are checked on every call, since a newer replica can migrate the database while this one keeps running
func (s *Service) Warnings(ctx context.Context) ([]warning, error) {
	warnings := []warning{}
	if s.showSQLiteStorageWarning() {
		warnings = append(warnings, warningSQLiteOnNetworkedFilesystem)
	}

	sqlDb, err := s.db.DB()
	if err != nil {
		return nil, err
	}
	newer, err := utils.HasNewerMigrations(ctx, sqlDb, common.EnvConfig.DbProvider)
	if err != nil {
		return nil, fmt.Errorf("failed to check for migrations of a newer version: %w", err)
	}
	if newer {
		warnings = append(warnings, warningDatabaseMigratedByNewerVersion)
	}

	return warnings, nil
}

// showSQLiteStorageWarning reports whether admins should be warned that the SQLite database is on a networked filesystem
func (s *Service) showSQLiteStorageWarning() bool {
	return s.sqliteOnNetworkedFilesystem && !bool(common.EnvConfig.DismissSQLiteStorageWarning)
}
