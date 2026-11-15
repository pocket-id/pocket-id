package users

import (
	"context"
	"fmt"
	"log/slog"
	"path"
	"time"

	"github.com/italypaleale/francis/builtin/cronjob"
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/model"
	"github.com/pocket-id/pocket-id/backend/internal/storage"
	profilepicture "github.com/pocket-id/pocket-id/backend/internal/utils/image"
)

// newCleanupJob returns the cron job actor that deletes the generated default profile pictures no user needs anymore
func newCleanupJob(db *gorm.DB, fileStorage storage.FileStorage) (*cronjob.CronJob, error) {
	job := &cleanupJob{
		db:          db,
		fileStorage: fileStorage,
	}

	cj, err := cronjob.New(
		"ClearUnusedDefaultProfilePictures",
		cronjob.WithJob(job.clearUnusedDefaultProfilePictures),
		// Run every 24 hours
		cronjob.WithInterval(24*time.Hour),
		cronjob.WithLogger(slog.Default()),
		cronjob.WithJitter(5*time.Minute),
	)
	if err != nil {
		return nil, fmt.Errorf("error creating ClearUnusedDefaultProfilePictures job: %w", err)
	}
	return cj, nil
}

type cleanupJob struct {
	db          *gorm.DB
	fileStorage storage.FileStorage
}

// ClearUnusedDefaultProfilePictures deletes default profile pictures that don't match any user's initials or were rendered by an earlier version
func (j *cleanupJob) clearUnusedDefaultProfilePictures(ctx context.Context) error {
	var users []model.User
	err := j.db.
		WithContext(ctx).
		Find(&users).
		Error
	if err != nil {
		return fmt.Errorf("failed to fetch users: %w", err)
	}

	// Create a map to track which pictures are in use
	filesInUse := make(map[string]struct{})
	for _, user := range users {
		filesInUse[profilepicture.DefaultPictureFile(profilepicture.Initials(user.FirstName, user.LastName, user.Username))] = struct{}{}
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
		// If no user needs this picture, delete the file
		if _, ok := filesInUse[filename]; !ok {
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
