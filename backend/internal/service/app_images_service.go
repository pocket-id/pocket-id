package service

import (
	"fmt"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
)

type AppImagesService struct {
	mu         sync.RWMutex
	extensions map[string]string
}

func NewAppImagesService(extensions map[string]string) *AppImagesService {
	return &AppImagesService{extensions: extensions}
}

func (s *AppImagesService) GetImage(name string) (string, string, error) {
	ext, err := s.getExtension(name)
	if err != nil {
		return "", "", err
	}

	mimeType := utils.GetImageMimeType(ext)
	if mimeType == "" {
		return "", "", fmt.Errorf("unsupported image type '%s'", ext)
	}

	imagePath := filepath.Join(common.EnvConfig.UploadPath, "application-images", fmt.Sprintf("%s.%s", name, ext))
	return imagePath, mimeType, nil
}

func (s *AppImagesService) UpdateImage(file *multipart.FileHeader, imageName string) error {
	fileType := strings.ToLower(utils.GetFileExtension(file.Filename))
	mimeType := utils.GetImageMimeType(fileType)
	if mimeType == "" {
		return &common.FileTypeNotSupportedError{}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	currentExt, ok := s.extensions[imageName]
	if !ok {
		s.extensions[imageName] = fileType
	}

	imagePath := filepath.Join(common.EnvConfig.UploadPath, "application-images", fmt.Sprintf("%s.%s", imageName, fileType))

	if err := utils.SaveFile(file, imagePath); err != nil {
		return err
	}

	if currentExt != "" && currentExt != fileType {
		oldImagePath := filepath.Join(common.EnvConfig.UploadPath, "application-images", fmt.Sprintf("%s.%s", imageName, currentExt))
		if err := os.Remove(oldImagePath); err != nil && !os.IsNotExist(err) {
			return err
		}
	}

	s.extensions[imageName] = fileType

	return nil
}

func (s *AppImagesService) DeleteImage(imageName string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	ext, ok := s.extensions[imageName]
	if !ok || ext == "" {
		return &common.ImageNotFoundError{}
	}

	imagePath := filepath.Join(common.EnvConfig.UploadPath, "application-images", imageName+"."+ext)
	if err := os.Remove(imagePath); err != nil && !os.IsNotExist(err) {
		return err
	}

	delete(s.extensions, imageName)
	return nil
}

func (s *AppImagesService) IsDefaultProfilePictureSet() bool {
	_, ok := s.extensions["default-profile-picture"]
	return ok
}

func (s *AppImagesService) getExtension(name string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ext, ok := s.extensions[name]
	if !ok || ext == "" {
		return "", &common.ImageNotFoundError{}
	}

	return strings.ToLower(ext), nil
}
