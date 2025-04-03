package job

import (
	"fmt"
	"github.com/go-co-op/gocron/v2"
	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	"gorm.io/gorm"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func RegisterFileCleanupJobs(db *gorm.DB) {
	scheduler, err := gocron.NewScheduler()
	if err != nil {
		log.Fatalf("Failed to create a new scheduler: %s", err)
	}

	jobs := &FileCleanupJobs{db: db}

	registerJob(scheduler, "ClearUnusedDefaultProfilePictures", "0 2 * * 0", jobs.clearUnusedDefaultProfilePictures)

	scheduler.Start()
}

type FileCleanupJobs struct {
	db *gorm.DB
}

// ClearUnusedDefaultProfilePictures deletes default profile pictures that don't match any user's initials
func (j *FileCleanupJobs) clearUnusedDefaultProfilePictures() error {
	var users []model.User
	if err := j.db.Find(&users).Error; err != nil {
		return fmt.Errorf("failed to fetch users: %w", err)
	}

	// Create a map to track which initials are in use
	initialsInUse := make(map[string]bool)
	for _, user := range users {
		initialsInUse[user.Initials()] = true
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
		if !initialsInUse[initials] {
			filePath := filepath.Join(defaultPicturesDir, filename)
			if err := os.Remove(filePath); err != nil {
				log.Printf("Failed to delete unused default profile picture %s: %v", filePath, err)
			} else {
				filesDeleted++
			}
		}
	}

	log.Printf("Deleted %d unused default profile pictures", filesDeleted)
	return nil
}
