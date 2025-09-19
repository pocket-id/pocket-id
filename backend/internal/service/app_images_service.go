package service

import (
	"fmt"
	"mime/multipart"
	"os"
	"path"
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

	imagePath := path.Join(common.EnvConfig.UploadPath, "application-images", fmt.Sprintf("%s.%s", name, ext))
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
		return fmt.Errorf("unknown application image '%s'", imageName)
	}

	imagePath := path.Join(common.EnvConfig.UploadPath, "application-images", fmt.Sprintf("%s.%s", imageName, fileType))

	if err := utils.SaveFile(file, imagePath); err != nil {
		return err
	}

	if currentExt != "" && currentExt != fileType {
		oldImagePath := path.Join(common.EnvConfig.UploadPath, "application-images", fmt.Sprintf("%s.%s", imageName, currentExt))
		if err := os.Remove(oldImagePath); err != nil && !os.IsNotExist(err) {
			return err
		}
	}

	s.extensions[imageName] = fileType

	return nil
}

func (s *AppImagesService) getExtension(name string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ext, ok := s.extensions[name]
	if !ok || ext == "" {
		return "", fmt.Errorf("unknown application image '%s'", name)
	}

	return strings.ToLower(ext), nil
}
