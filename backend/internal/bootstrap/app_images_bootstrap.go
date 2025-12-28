package bootstrap

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path"

	"github.com/pocket-id/pocket-id/backend/internal/storage"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
	"github.com/pocket-id/pocket-id/backend/resources"
)

// initApplicationImages copies the images from the embedded directory to the storage backend
// and returns a map containing the detected file extensions in the application-images directory.
func initApplicationImages(ctx context.Context, fileStorage storage.FileStorage) (map[string]string, error) {
	// Previous versions of images
	// If these are found, they are deleted
	legacyImageHashes := imageHashMap{
		"background.jpg":  mustDecodeHex("138d510030ed845d1d74de34658acabff562d306476454369a60ab8ade31933f"),
		"background.webp": mustDecodeHex("3fc436a66d6b872b01d96a4e75046c46b5c3e2daccd51e98ecdf98fd445599ab"),
	}

	sourceFiles, err := resources.FS.ReadDir("images")
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	destinationFiles, err := fileStorage.List(ctx, "application-images")
	if err != nil {
		if storage.IsNotExist(err) {
			destinationFiles = []storage.ObjectInfo{}
		} else {
			return nil, fmt.Errorf("failed to list application images: %w", err)
		}

	}
	dstNameToExt := make(map[string]string, len(destinationFiles))
	for _, f := range destinationFiles {
		// Skip directories
		_, name := path.Split(f.Path)
		if name == "" {
			continue
		}
		nameWithoutExt, ext := utils.SplitFileName(name)
		reader, _, err := fileStorage.Open(ctx, f.Path)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				continue
			}
			slog.Warn("Failed to open application image for hashing", slog.String("name", name), slog.Any("error", err))
			continue
		}
		hash, err := hashStream(reader)
		reader.Close()
		if err != nil {
			slog.Warn("Failed to hash application image", slog.String("name", name), slog.Any("error", err))
			continue
		}

		// Check if the file is a legacy one - if so, delete it
		if legacyImageHashes.Contains(hash) {
			slog.Info("Found legacy application image that will be removed", slog.String("name", name))
			if err := fileStorage.Delete(ctx, f.Path); err != nil {
				return nil, fmt.Errorf("failed to remove legacy file '%s': %w", name, err)
			}
			continue
		}
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

		if _, exists := dstNameToExt[nameWithoutExt]; exists {
			continue
		}

		slog.Info("Writing new application image", slog.String("name", name))
		srcFile, err := resources.FS.Open(srcFilePath)
		if err != nil {
			return nil, fmt.Errorf("failed to open embedded file '%s': %w", name, err)
		}
		if err := fileStorage.Save(ctx, path.Join("application-images", name), srcFile); err != nil {
			srcFile.Close()
			return nil, fmt.Errorf("failed to store application image '%s': %w", name, err)
		}
		srcFile.Close()
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

func hashStream(r io.Reader) ([]byte, error) {
	h := sha256.New()
	if _, err := io.Copy(h, r); err != nil {
		return nil, err
	}
	return h.Sum(nil), nil
}
