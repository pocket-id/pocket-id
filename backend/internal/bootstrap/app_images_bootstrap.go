package bootstrap

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"log/slog"
	"os"
	"path"

	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
	"github.com/pocket-id/pocket-id/backend/resources"
)

// initApplicationImages copies the images from the images directory to the application-images directory
// and returns a map containing the detected file extensions in the application-images directory.
func initApplicationImages() (map[string]string, error) {
	// Previous versions of images
	// If these are found, they are deleted
	legacyImageHashes := imageHashMap{
		"background.jpg": mustDecodeHex("138d510030ed845d1d74de34658acabff562d306476454369a60ab8ade31933f"),
	}

	dirPath := common.EnvConfig.UploadPath + "/application-images"

	sourceFiles, err := resources.FS.ReadDir("images")
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	destinationFiles, err := os.ReadDir(dirPath)
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}
	dstNameToExt := make(map[string]string, len(destinationFiles))
	for _, f := range destinationFiles {
		if f.IsDir() {
			continue
		}
		name := f.Name()
		nameWithoutExt, ext := utils.SplitFileName(name)
		destFilePath := path.Join(dirPath, name)

		h, err := utils.CreateSha256FileHash(destFilePath)
		if err != nil {
			return nil, fmt.Errorf("failed to get hash for file '%s': %w", name, err)
		}

		// Check if the file is a legacy one - if so, delete it
		if legacyImageHashes.Contains(h) {
			slog.Info("Found legacy application image that will be removed", slog.String("name", name))
			err = os.Remove(destFilePath)
			if err != nil {
				return nil, fmt.Errorf("failed to remove legacy file '%s': %w", name, err)
			}
			continue
		}

		// Track existing files
		dstNameToExt[nameWithoutExt] = ext
	}

	// Copy images from the images directory to the application-images directory if they don't already exist
	for _, sourceFile := range sourceFiles {
		if sourceFile.IsDir() {
			continue
		}

		name := sourceFile.Name()
		nameWithoutExt, ext := utils.SplitFileName(name)
		srcFilePath := path.Join("images", name)
		destFilePath := path.Join(dirPath, name)

		// Skip if there's already an image at the path
		// We do not check the extension because users could have uploaded a different one
		if _, exists := dstNameToExt[nameWithoutExt]; exists {
			continue
		}

		slog.Info("Writing new application image", slog.String("name", name))
		err := utils.CopyEmbeddedFileToDisk(srcFilePath, destFilePath)
		if err != nil {
			return nil, fmt.Errorf("failed to copy file: %w", err)
		}

		// Track the newly copied file so it can be included in the extensions map later
		dstNameToExt[nameWithoutExt] = ext
	}

	return dstNameToExt, nil
}

type imageHashMap map[string][]byte

func (m imageHashMap) Contains(target []byte) bool {
	if len(target) == 0 {
		return false
	}
	for _, h := range m {
		if bytes.Equal(h, target) {
			return true
		}
	}
	return false
}

func mustDecodeHex(str string) []byte {
	b, err := hex.DecodeString(str)
	if err != nil {
		panic(err)
	}
	return b
}
