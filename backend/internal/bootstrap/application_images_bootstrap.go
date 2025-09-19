package bootstrap

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path"
	"strings"

	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
	"github.com/pocket-id/pocket-id/backend/resources"
)

// initApplicationImages copies the images from the images directory to the application-images directory
func initApplicationImages() error {
	// Images that are built into the Pocket ID binary
	builtInImageHashes := getBuiltInImageHashes()

	// Previous versions of images
	// If these are found, they are deleted
	legacyImageHashes := imageHashMap{
		"background.jpg": mustDecodeHex("138d510030ed845d1d74de34658acabff562d306476454369a60ab8ade31933f"),
	}

	dirPath := common.EnvConfig.UploadPath + "/application-images"

	sourceFiles, err := resources.FS.ReadDir("images")
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to read directory: %w", err)
	}

	destinationFiles, err := os.ReadDir(dirPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to read directory: %w", err)
	}
	destinationFilesMap := make(map[string]bool, len(destinationFiles))
	for _, f := range destinationFiles {
		name := f.Name()
		destFilePath := path.Join(dirPath, name)

		h, err := utils.CreateSha256FileHash(destFilePath)
		if err != nil {
			return fmt.Errorf("failed to get hash for file '%s': %w", name, err)
		}

		// Check if the file is a legacy one - if so, delete it
		if legacyImageHashes.Contains(h) {
			slog.Info("Found legacy application image that will be removed", slog.String("name", name))
			err = os.Remove(destFilePath)
			if err != nil {
				return fmt.Errorf("failed to remove legacy file '%s': %w", name, err)
			}
			continue
		}

		// Check if the file is a built-in one and save it in the map
		destinationFilesMap[getImageNameWithoutExtension(name)] = builtInImageHashes.Contains(h)
	}

	// Copy images from the images directory to the application-images directory if they don't already exist
	for _, sourceFile := range sourceFiles {
		// Skip if it's a directory
		if sourceFile.IsDir() {
			continue
		}

		name := sourceFile.Name()
		srcFilePath := path.Join("images", name)
		destFilePath := path.Join(dirPath, name)

		// Skip if there's already an image at the path
		// We do not check the extension because users could have uploaded a different one
		if imageAlreadyExists(sourceFile, destinationFilesMap) {
			continue
		}

		slog.Info("Writing new application image", slog.String("name", name))
		err := utils.CopyEmbeddedFileToDisk(srcFilePath, destFilePath)
		if err != nil {
			return fmt.Errorf("failed to copy file: %w", err)
		}
	}

	return nil
}

func getBuiltInImageHashes() imageHashMap {
	return imageHashMap{
		"background.webp": mustDecodeHex("3fc436a66d6b872b01d96a4e75046c46b5c3e2daccd51e98ecdf98fd445599ab"),
		"favicon.ico":     mustDecodeHex("70f9c4b6bd4781ade5fc96958b1267511751e91957f83c2354fb880b35ec890a"),
		"logo.svg":        mustDecodeHex("f1e60707df9784152ce0847e3eb59cb68b9015f918ff160376c27ebff1eda796"),
		"logoDark.svg":    mustDecodeHex("0421a8d93714bacf54c78430f1db378fd0d29565f6de59b6a89090d44a82eb16"),
		"logoLight.svg":   mustDecodeHex("6d42c88cf6668f7e57c4f2a505e71ecc8a1e0a27534632aa6adec87b812d0bb0"),
	}
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

func imageAlreadyExists(sourceFile fs.DirEntry, destinationFiles map[string]bool) bool {
	sourceFileWithoutExtension := getImageNameWithoutExtension(sourceFile.Name())
	_, ok := destinationFiles[sourceFileWithoutExtension]
	return ok
}

func getImageNameWithoutExtension(fileName string) string {
	idx := strings.LastIndexByte(fileName, '.')
	if idx < 1 {
		// No dot found, or fileName starts with a dot
		return fileName
	}
	return fileName[:idx]
}

func mustDecodeHex(str string) []byte {
	b, err := hex.DecodeString(str)
	if err != nil {
		panic(err)
	}
	return b
}
