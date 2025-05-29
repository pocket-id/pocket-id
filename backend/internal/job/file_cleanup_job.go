package job

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-co-op/gocron/v2"
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/model"
)

func (s *Scheduler) RegisterFileCleanupJobs(ctx context.Context, db *gorm.DB) error {
	jobs := &FileCleanupJobs{db: db}

	// Run every 24 hours
	return s.registerJob(ctx, "ClearUnusedDefaultProfilePictures", gocron.DurationJob(24*time.Hour), jobs.clearUnusedDefaultProfilePictures, false)
}

type FileCleanupJobs struct {
	db *gorm.DB
}

// ClearUnusedDefaultProfilePictures deletes default profile pictures that don't match any user's initials
func (j *FileCleanupJobs) clearUnusedDefaultProfilePictures(ctx context.Context) error {
	var users []model.User
	err := j.db.
		WithContext(ctx).
		Find(&users).
		Error
	if err != nil {
		return fmt.Errorf("failed to fetch users: %w", err)
	}

	// Create a map to track which initials are in use
	initialsInUse := make(map[string]struct{})
	for _, user := range users {
		initialsInUse[user.Initials()] = struct{}{}
	}

	defaultPicturesDir := common.EnvConfig.UploadPath + "/profile-pictures/defaults"
	if _, err := os.Stat(defaultPicturesDir); os.IsNotExist(err) {
		return nil
	}

	files, err := os.ReadDir(defaultPicturesDir)
	if err != nil {
		return fmt.Errorf("failed to read default profile pictures directory: %w", err)
	}

	filesDeleted := 0
	for _, file := range files {
		if file.IsDir() {
			continue // Skip directories
		}

		filename := file.Name()
		initials := strings.TrimSuffix(filename, ".png")

		// If these initials aren't used by any user, delete the file
		if _, ok := initialsInUse[initials]; !ok {
			filePath := filepath.Join(defaultPicturesDir, filename)
			if err := os.Remove(filePath); err != nil {
				slog.ErrorContext(ctx, "Failed to delete unused default profile picture", slog.String("path", filePath), slog.Any("error", err))
			} else {
				filesDeleted++
			}
		}
	}

	slog.Info("Done deleting unused default profile pictures", slog.Int("count", filesDeleted))
	return nil
}
