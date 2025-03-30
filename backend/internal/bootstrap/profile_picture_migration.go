package bootstrap

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
	profilepicture "github.com/pocket-id/pocket-id/backend/internal/utils/image"
	"gorm.io/gorm"
)

func migrateProfilePictures(db *gorm.DB) {
	err := migrateProfilePicturesPrivate(db)
	if err != nil {
		log.Fatalf("failed to perform migration of profile pictures: %v", err)
	}
}

// MigrateDefaultProfilePictures generates initials-based default profile pictures
// for users who don't have custom profile pictures
func migrateProfilePicturesPrivate(db *gorm.DB) error {
	uploadPath := common.EnvConfig.UploadPath
	var users []model.User
	if err := db.Find(&users).Error; err != nil {
		return fmt.Errorf("failed to fetch users: %w", err)
	}

	// Create defaults directory if it doesn't exist
	defaultsDir := uploadPath + "/profile-pictures/defaults"
	if err := os.MkdirAll(defaultsDir, os.ModePerm); err != nil {
		return fmt.Errorf("failed to create defaults directory: %w", err)
	}

	processed := 0
	skipped := 0

	for _, user := range users {
		initials := profilepicture.GetUserInitials(user.FirstName, user.LastName)

		// Process each user using a switch statement for cleaner flow control
		switch {
		// Case 1: User has no initials
		case initials == "":
			skipped++

		// Case 2: Initials-based picture already exists
		case fileExists(filepath.Join(defaultsDir, initials+".png")):
			skipped++

		// Case 3: User has a custom profile picture
		case fileExists(filepath.Join(uploadPath, "profile-pictures", user.ID+".png")):
			skipped++

		// Case 4: Need to generate a default picture
		default:
			newPath := filepath.Join(defaultsDir, initials+".png")

			// Generate and save default picture
			defaultPicture, err := profilepicture.CreateDefaultProfilePicture(user.FirstName, user.LastName)
			if err != nil {
				log.Printf("Failed to create default picture for user %s: %v", user.ID, err)
				continue
			}

			if err := utils.SaveFileStream(bytes.NewBuffer(defaultPicture.Bytes()), newPath); err != nil {
				log.Printf("Failed to save default picture for initials %s: %v", initials, err)
				continue
			}

			processed++
			log.Printf("Created default picture for initials %s", initials)
		}
	}

	return nil
}

// Helper function to check if a file exists
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
