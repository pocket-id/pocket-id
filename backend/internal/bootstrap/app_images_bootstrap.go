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
	"strings"

	"github.com/pocket-id/pocket-id/backend/internal/storage"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
	"github.com/pocket-id/pocket-id/backend/resources"
)

const (
	applicationImagesPath             = "application-images"
	deletedApplicationImagesPath      = applicationImagesPath + "/.deleted"
	legacyApplicationImagesInitedPath = applicationImagesPath + "/.inited"
	deletableBundledApplicationImage  = "background"
)

// initApplicationImages copies embedded images to storage and returns the detected file extensions
//
//nolint:gocognit
func initApplicationImages(ctx context.Context, fileStorage storage.FileStorage) (map[string]string, error) {
	// Previous versions of images
	// If these are found, they are deleted
	legacyImageHashes := imageHashMap{
		"logoLight.svg": {
			mustDecodeHex("6d42c88cf6668f7e57c4f2a505e71ecc8a1e0a27534632aa6adec87b812d0bb0"),
		},
		"logoDark.svg": {
			mustDecodeHex("0421a8d93714bacf54c78430f1db378fd0d29565f6de59b6a89090d44a82eb16"),
		},
		"background.jpg": {
			mustDecodeHex("138d510030ed845d1d74de34658acabff562d306476454369a60ab8ade31933f"),
		},
		"background.webp": {
			mustDecodeHex("3fc436a66d6b872b01d96a4e75046c46b5c3e2daccd51e98ecdf98fd445599ab"),
		},
	}

	sourceFiles, err := resources.FS.ReadDir("images")
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	destinationFiles, err := fileStorage.List(ctx, applicationImagesPath)
	if err != nil {
		if storage.IsNotExist(err) {
			destinationFiles = []storage.ObjectInfo{}
		} else {
			return nil, fmt.Errorf("failed to list application images: %w", err)
		}

	}
	dstNameToExt := make(map[string]string, len(destinationFiles))
	listedImageNames := make(map[string]struct{}, len(destinationFiles))
	for _, f := range destinationFiles {
		// Skip bootstrap state that recursive storage backends may include in the listing
		if f.Path == legacyApplicationImagesInitedPath || strings.HasPrefix(f.Path, deletedApplicationImagesPath+"/") {
			continue
		}

		// Skip directory entries returned by storage backends
		_, name := path.Split(f.Path)
		if name == "" {
			continue
		}
		nameWithoutExt, ext := utils.SplitFileName(name)
		listedImageNames[nameWithoutExt] = struct{}{}
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

		// Remove bundled legacy images so their current versions can be restored
		if legacyImageHashes.Matches(name, hash) {
			slog.Info("Found legacy application image that will be removed", slog.String("name", name))
			if err := fileStorage.Delete(ctx, f.Path); err != nil {
				return nil, fmt.Errorf("failed to remove legacy file '%s': %w", name, err)
			}
			continue
		}
		dstNameToExt[nameWithoutExt] = ext
	}

	// Preserve an intentionally deleted background when replacing the legacy global initialization marker
	legacyInited, err := storageObjectExists(ctx, fileStorage, legacyApplicationImagesInitedPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read legacy application images marker: %w", err)
	}
	if legacyInited {
		_, backgroundWasPresent := listedImageNames[deletableBundledApplicationImage]
		if !backgroundWasPresent {
			deletedPath := deletedApplicationImagePath(deletableBundledApplicationImage)
			if err := fileStorage.Save(ctx, deletedPath, strings.NewReader("")); err != nil {
				return nil, fmt.Errorf("failed to store deleted application image marker '%s': %w", deletableBundledApplicationImage, err)
			}
		}
		if err := fileStorage.Delete(ctx, legacyApplicationImagesInitedPath); err != nil {
			return nil, fmt.Errorf("failed to remove legacy application images marker: %w", err)
		}
	}

	// Copy missing bundled images unless an administrator intentionally deleted them
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
		deleted, err := storageObjectExists(ctx, fileStorage, deletedApplicationImagePath(nameWithoutExt))
		if err != nil {
			return nil, fmt.Errorf("failed to read deleted application image marker '%s': %w", nameWithoutExt, err)
		}
		if deleted {
			continue
		}

		slog.Info("Writing new application image", slog.String("name", name))
		srcFile, err := resources.FS.Open(srcFilePath)
		if err != nil {
			return nil, fmt.Errorf("failed to open embedded file '%s': %w", name, err)
		}
		if err := fileStorage.Save(ctx, path.Join(applicationImagesPath, name), srcFile); err != nil {
			srcFile.Close()
			return nil, fmt.Errorf("failed to store application image '%s': %w", name, err)
		}
		srcFile.Close()
		dstNameToExt[nameWithoutExt] = ext
	}

	return dstNameToExt, nil
}

type imageHashMap map[string][][]byte

func (m imageHashMap) Matches(name string, target []byte) bool {
	if len(target) == 0 {
		return false
	}
	for _, hash := range m[name] {
		if bytes.Equal(hash, target) {
			return true
		}
	}
	return false
}

func deletedApplicationImagePath(name string) string {
	return path.Join(deletedApplicationImagesPath, name)
}

func storageObjectExists(ctx context.Context, fileStorage storage.FileStorage, objectPath string) (bool, error) {
	reader, _, err := fileStorage.Open(ctx, objectPath)
	if err == nil {
		reader.Close()
		return true, nil
	}
	if storage.IsNotExist(err) {
		return false, nil
	}
	return false, err
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
