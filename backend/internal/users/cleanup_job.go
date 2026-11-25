package users

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"path"
	"time"

	"github.com/pocket-id/pocket-id/backend/internal/model"
	profilepicture "github.com/pocket-id/pocket-id/backend/internal/utils/image"
)

const (
	// defaultPicturesDir is the folder of the generated default profile pictures in the cache storage
	defaultPicturesDir = "profile-pictures"
	// defaultPicturesCleanupInterval is how often the default profile pictures no user needs anymore are deleted
	defaultPicturesCleanupInterval = 24 * time.Hour
)

// Run deletes the unused default profile pictures at startup and then once a day, until the context is canceled
// It is a loop in every process instead of a cron job actor, which runs on one replica only, because the cache folder may be local to each replica
// It satisfies servicerunner.Service
func (m *Module) Run(ctx context.Context) error {
	ticker := time.NewTicker(defaultPicturesCleanupInterval)
	defer ticker.Stop()

	for {
		err := m.service.clearUnusedDefaultProfilePictures(ctx)
		if err != nil {
			slog.ErrorContext(ctx, "Failed to delete unused default profile pictures", slog.Any("error", err))
		}

		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
	}
}

// clearUnusedDefaultProfilePictures deletes default profile pictures that don't match any user's initials or were rendered by an earlier version
func (s *Service) clearUnusedDefaultProfilePictures(ctx context.Context) error {
	// Nothing was cached yet when the folder doesn't exist
	files, err := s.cacheStorage.List(ctx, defaultPicturesDir)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to list default profile pictures: %w", err)
	}

	// Only the fields the initials are made of are loaded
	var users []model.User
	err = s.db.
		WithContext(ctx).
		Select("first_name", "last_name", "username").
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

	filesDeleted := 0
	for _, file := range files {
		// If no user needs this picture, delete the file
		_, ok := filesInUse[path.Base(file.Path)]
		if ok {
			continue
		}
		err = s.cacheStorage.Delete(ctx, file.Path)
		if err != nil {
			slog.ErrorContext(ctx, "Failed to delete unused default profile picture", slog.String("path", file.Path), slog.Any("error", err))
		} else {
			filesDeleted++
		}
	}

	slog.InfoContext(ctx, "Done deleting unused default profile pictures", slog.Int("count", filesDeleted))
	return nil
}
