package bootstrap

import (
	"log"
	"os"
	"path"
	"strings"

	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
	"github.com/pocket-id/pocket-id/backend/resources"
)

// initApplicationImages copies the images from the images directory to the application-images directory
func initApplicationImages() {
	dirPath := common.EnvConfig.UploadPath + "/application-images"

	sourceFiles, err := resources.FS.ReadDir("images")
	if err != nil && !os.IsNotExist(err) {
		log.Fatalf("Error reading directory: %v", err)
	}

	destinationFiles, err := os.ReadDir(dirPath)
	if err != nil && !os.IsNotExist(err) {
		log.Fatalf("Error reading directory: %v", err)
	}

	// Copy images from the images directory to the application-images directory if they don't already exist
	for _, sourceFile := range sourceFiles {
		if sourceFile.IsDir() || imageAlreadyExists(sourceFile.Name(), destinationFiles) {
			continue
		}
		srcFilePath := path.Join("images", sourceFile.Name())
		destFilePath := path.Join(dirPath, sourceFile.Name())

		err := utils.CopyEmbeddedFileToDisk(srcFilePath, destFilePath)
		if err != nil {
			log.Fatalf("Error copying file: %v", err)
		}
	}
}

func imageAlreadyExists(fileName string, destinationFiles []os.DirEntry) bool {
	for _, destinationFile := range destinationFiles {
		sourceFileWithoutExtension := getImageNameWithoutExtension(fileName)
		destinationFileWithoutExtension := getImageNameWithoutExtension(destinationFile.Name())

		if sourceFileWithoutExtension == destinationFileWithoutExtension {
			return true
		}
	}

	return false
}

func getImageNameWithoutExtension(fileName string) string {
	idx := strings.LastIndexByte(fileName, '.')
	if idx < 1 {
		// No dot found, or fileName starts with a dot
		return fileName
	}

	return fileName[:idx]
}
