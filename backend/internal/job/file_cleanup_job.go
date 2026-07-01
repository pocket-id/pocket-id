package job

import (
	"context"
	"fmt"
	"log/slog"
	"path"
	"strings"
	"time"

	"github.com/italypaleale/francis/builtin/cronjob"
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/model"
	"github.com/pocket-id/pocket-id/backend/internal/storage"
)

// GetFileCleanupJobs returns the CronJob actors
func GetFileCleanupJobs(db *gorm.DB, fileStorage storage.FileStorage) (cjs []*cronjob.CronJob, err error) {
	job := &FileCleanupJobs{
		db:          db,
		fileStorage: fileStorage,
	}

	// Create the built-in actor for the ClearUnusedDefaultProfilePictures job
	cj, err := cronjob.New(
		"ClearUnusedDefaultProfilePictures",
		cronjob.WithJob(job.clearUnusedDefaultProfilePictures),
		// Run every 24 hours
		cronjob.WithInterval(24*time.Hour),
		cronjob.WithLogger(slog.Default()),
	)
	if err != nil {
		return nil, fmt.Errorf("error creating ClearUnusedDefaultProfilePictures job: %w", err)
	}
	cjs = append(cjs, cj)

	// Create the built-in actor for the ClearOrphanedTempFiles job
	// Only necessary for file system storage
	if fileStorage.Type() == storage.TypeFileSystem {
		cj, err := cronjob.New(
			"ClearOrphanedTempFiles",
			cronjob.WithJob(job.clearOrphanedTempFiles),
			// Run every 12 hours
			cronjob.WithInterval(12*time.Hour),
			cronjob.WithLogger(slog.Default()),
		)
		if err != nil {
			return nil, fmt.Errorf("error creating ClearUnusedDefaultProfilePictures job: %w", err)
		}
		cjs = append(cjs, cj)
	}

	return cjs, nil
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
			err = j.fileStorage.Delete(ctx, filePath)
			if err != nil {
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

	deleted := 0
	err := j.fileStorage.Walk(ctx, "/", func(p storage.ObjectInfo) error {
		// Only temp files
		if !strings.HasSuffix(p.Path, "-tmp") {
			return nil
		}

		if time.Since(p.ModTime) < minAge {
			return nil
		}

		rErr := j.fileStorage.Delete(ctx, p.Path)
		if rErr != nil {
			slog.ErrorContext(ctx, "Failed to delete temp file", slog.String("path", p.Path), slog.Any("error", rErr))
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
