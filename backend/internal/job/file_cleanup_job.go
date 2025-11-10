package job

import (
	"context"
	"errors"
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

	err := s.registerJob(ctx, "ClearUnusedDefaultProfilePictures", gocron.DurationJob(24*time.Hour), jobs.clearUnusedDefaultProfilePictures, false)

	// Only necessary for file system storage
	if fileStorage.Type() == storage.TypeFileSystem {
		err = errors.Join(err, s.registerJob(ctx, "ClearOrphanedTempFiles", gocron.DurationJob(12*time.Hour), jobs.clearOrphanedTempFiles, true))
	}

	return err
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

// clearOrphanedTempFiles deletes temporary files that are produced by failed atomic writes
func (j *FileCleanupJobs) clearOrphanedTempFiles(ctx context.Context) error {
	const minAge = 10 * time.Minute

	var deleted int
	err := j.fileStorage.Walk(ctx, "/", func(p storage.ObjectInfo) error {
		// Only temp files
		if !strings.HasSuffix(p.Path, "-tmp") {
			return nil
		}

		if time.Since(p.ModTime) < minAge {
			return nil
		}

		if err := j.fileStorage.Delete(ctx, p.Path); err != nil {
			slog.ErrorContext(ctx, "Failed to delete temp file", slog.String("path", p.Path), slog.Any("error", err))
			return nil
		}
		deleted++
		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to scan storage: %w", err)
	}

	slog.Info("Done cleaning orphaned temp files", slog.Int("count", deleted))
	return nil
}
