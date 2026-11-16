package appassets

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path"
	"strings"

	"github.com/pocket-id/pocket-id/backend/internal/storage"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
	"github.com/pocket-id/pocket-id/backend/resources"
)

const (
	applicationAssetsPath        = "application-assets"
	deletedApplicationAssetsPath = applicationAssetsPath + "/.deleted"
)

// initApplicationAssets copies embedded images to storage and returns the detected file extensions
func initApplicationAssets(ctx context.Context, fileStorage storage.FileStorage) (map[string]string, error) {
	sourceFiles, err := resources.FS.ReadDir("images")
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	destinationFiles, err := fileStorage.List(ctx, applicationAssetsPath)
	if err != nil {
		if storage.IsNotExist(err) {
			destinationFiles = []storage.ObjectInfo{}
		} else {
			return nil, fmt.Errorf("failed to list application images: %w", err)
		}

	}
	dstNameToExt := make(map[string]string, len(destinationFiles))
	for _, f := range destinationFiles {
		// Skip the deleted markers that recursive storage backends may include in the listing
		if strings.HasPrefix(f.Path, deletedApplicationAssetsPath+"/") {
			continue
		}

		// Skip directory entries returned by storage backends
		_, name := path.Split(f.Path)
		if name == "" {
			continue
		}
		nameWithoutExt, ext := utils.SplitFileName(name)
		dstNameToExt[nameWithoutExt] = ext
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
		deleted, err := storageObjectExists(ctx, fileStorage, deletedApplicationAssetPath(nameWithoutExt))
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
		if err := fileStorage.Save(ctx, path.Join(applicationAssetsPath, name), srcFile); err != nil {
			srcFile.Close()
			return nil, fmt.Errorf("failed to store application image '%s': %w", name, err)
		}
		srcFile.Close()
		dstNameToExt[nameWithoutExt] = ext
	}

	return dstNameToExt, nil
}

func deletedApplicationAssetPath(name string) string {
	return path.Join(deletedApplicationAssetsPath, name)
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
