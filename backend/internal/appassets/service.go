package appassets

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"path"
	"strings"
	"sync"

	"github.com/pocket-id/pocket-id/backend/internal/apperror"
	"github.com/pocket-id/pocket-id/backend/internal/storage"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
	imageutil "github.com/pocket-id/pocket-id/backend/internal/utils/image"
	"github.com/pocket-id/pocket-id/backend/resources"
)

type Service struct {
	mu         sync.RWMutex
	extensions map[string]string
	storage    storage.FileStorage
}

func (s *Service) GetAsset(ctx context.Context, name string) (io.ReadCloser, int64, string, error) {
	ext, err := s.getExtension(name)
	if err != nil {
		return nil, 0, "", err
	}

	mimeType := fileMimeType(name, ext)
	if mimeType == "" {
		return nil, 0, "", fmt.Errorf("unsupported image type '%s'", ext)
	}

	assetPath := path.Join(applicationAssetsPath, name+"."+ext)
	reader, size, err := s.storage.Open(ctx, assetPath)
	if err != nil {
		if storage.IsNotExist(err) {
			return nil, 0, "", apperror.ImageNotFound()
		}
		return nil, 0, "", err
	}
	return reader, size, mimeType, nil
}

// GetAssetWithDefault behaves like GetAsset, but falls back to the image embedded in the binary if no custom image has been uploaded
func (s *Service) GetAssetWithDefault(ctx context.Context, name string) (f io.ReadCloser, size int64, mimeType string, err error) {
	f, size, mimeType, err = s.GetAsset(ctx, name)
	if err == nil || !apperror.IsCode(err, apperror.CodeImageNotFound) {
		return f, size, mimeType, err
	}

	return getDefaultAsset(name)
}

func getDefaultAsset(name string) (io.ReadCloser, int64, string, error) {
	// Map an image name to an image embedded in the binary
	var assetPath string
	switch name {
	case "logoLight":
		assetPath = "default-images/logoLight.svg"
	case "logoDark":
		assetPath = "default-images/logoDark.svg"
	default:
		return nil, 0, "", apperror.ImageNotFound()
	}

	file, err := resources.FS.Open(assetPath)
	if err != nil {
		return nil, 0, "", fmt.Errorf("failed to open default image '%s': %w", name, err)
	}

	stat, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, 0, "", fmt.Errorf("failed to get size of default image '%s': %w", name, err)
	}

	return file, stat.Size(), utils.GetImageMimeType(utils.GetFileExtension(assetPath)), nil
}

func (s *Service) UpdateAsset(ctx context.Context, file *multipart.FileHeader, assetName string) error {
	fileType := strings.ToLower(utils.GetFileExtension(file.Filename))
	if fileMimeType(assetName, fileType) == "" {
		return apperror.UnsupportedFileType("")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	currentExt := s.extensions[assetName]

	assetPath := path.Join(applicationAssetsPath, assetName+"."+fileType)
	fileReader, err := file.Open()
	if err != nil {
		return err
	}
	defer fileReader.Close()

	strippedReader, err := imageutil.StripMetadata(fileReader, fileType)
	if errors.Is(err, imageutil.ErrInvalidImage) {
		return apperror.InvalidImage(err)
	}
	if err != nil {
		return err
	}

	if err := s.storage.Save(ctx, assetPath, strippedReader); err != nil {
		return err
	}

	if currentExt != "" && currentExt != fileType {
		oldAssetPath := path.Join(applicationAssetsPath, assetName+"."+currentExt)
		if err := s.storage.Delete(ctx, oldAssetPath); err != nil {
			return err
		}
	}
	s.extensions[assetName] = fileType

	if err := s.storage.Delete(ctx, deletedApplicationAssetPath(assetName)); err != nil {
		return err
	}

	return nil
}

func (s *Service) DeleteAsset(ctx context.Context, assetName string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	ext, ok := s.extensions[assetName]
	if !ok || ext == "" {
		return apperror.ImageNotFound()
	}

	if err := s.storage.Save(ctx, deletedApplicationAssetPath(assetName), strings.NewReader("")); err != nil {
		return err
	}

	assetPath := path.Join(applicationAssetsPath, assetName+"."+ext)
	if err := s.storage.Delete(ctx, assetPath); err != nil {
		return err
	}

	delete(s.extensions, assetName)
	return nil
}

// The title font is the one application file that isn't an image
func fileMimeType(name, ext string) string {
	if name == "font" {
		return utils.GetFontMimeType(ext)
	}
	return utils.GetImageMimeType(ext)
}

func (s *Service) IsDefaultProfilePictureSet() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	_, ok := s.extensions["default-profile-picture"]
	return ok
}

func (s *Service) getExtension(name string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ext, ok := s.extensions[name]
	if !ok || ext == "" {
		return "", apperror.ImageNotFound()
	}

	return strings.ToLower(ext), nil
}
