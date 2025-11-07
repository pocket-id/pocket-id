package job

import (
	"context"
	"fmt"
	"log/slog"
	"path"
	"strings"
	"time"

	"github.com/go-co-op/gocron/v2"
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/model"
	"github.com/pocket-id/pocket-id/backend/internal/storage"
)

func (s *Scheduler) RegisterFileCleanupJobs(ctx context.Context, db *gorm.DB, fileStorage storage.FileStorage) error {
	jobs := &FileCleanupJobs{db: db, fileStorage: fileStorage}

	// Run every 24 hours
	return s.registerJob(ctx, "ClearUnusedDefaultProfilePictures", gocron.DurationJob(24*time.Hour), jobs.clearUnusedDefaultProfilePictures, false)
}

type FileCleanupJobs struct {
	db          *gorm.DB
	fileStorage storage.FileStorage
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

	defaultPicturesDir := path.Join("profile-pictures", "defaults")
	files, err := j.fileStorage.List(ctx, defaultPicturesDir)
	if err != nil {
		return fmt.Errorf("failed to list default profile pictures: %w", err)
	}

	filesDeleted := 0
	for _, file := range files {
		_, filename := path.Split(file.Path)
		if filename == "" {
			continue
		}
		initials := strings.TrimSuffix(filename, ".png")

		// If these initials aren't used by any user, delete the file
		if _, ok := initialsInUse[initials]; !ok {
			filePath := path.Join(defaultPicturesDir, filename)
			if err := j.fileStorage.Delete(ctx, filePath); err != nil {
				slog.ErrorContext(ctx, "Failed to delete unused default profile picture", slog.String("path", filePath), slog.Any("error", err))
			} else {
				filesDeleted++
			}
		}
	}

	slog.Info("Done deleting unused default profile pictures", slog.Int("count", filesDeleted))
	return nil
}
