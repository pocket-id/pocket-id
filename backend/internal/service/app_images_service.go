package service

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"path"
	"strings"
	"sync"

	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/storage"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
)

type AppImagesService struct {
	mu         sync.RWMutex
	extensions map[string]string
	storage    storage.FileStorage
}

func NewAppImagesService(extensions map[string]string, storage storage.FileStorage) *AppImagesService {
	return &AppImagesService{extensions: extensions, storage: storage}
}

func (s *AppImagesService) GetImage(ctx context.Context, name string) (io.ReadCloser, int64, string, error) {
	ext, err := s.getExtension(name)
	if err != nil {
		return nil, 0, "", err
	}

	mimeType := utils.GetImageMimeType(ext)
	if mimeType == "" {
		return nil, 0, "", fmt.Errorf("unsupported image type '%s'", ext)
	}

	imagePath := path.Join("application-images", name+"."+ext)
	reader, size, err := s.storage.Open(ctx, imagePath)
	if err != nil {
		if storage.IsNotExist(err) {
			return nil, 0, "", &common.ImageNotFoundError{}
		}
		return nil, 0, "", err
	}
	return reader, size, mimeType, nil
}

func (s *AppImagesService) UpdateImage(ctx context.Context, file *multipart.FileHeader, imageName string) error {
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

	imagePath := path.Join("application-images", imageName+"."+fileType)
	fileReader, err := file.Open()
	if err != nil {
		return err
	}
	defer fileReader.Close()

	if err := s.storage.Save(ctx, imagePath, fileReader); err != nil {
		return err
	}

	if currentExt != "" && currentExt != fileType {
		oldImagePath := path.Join("application-images", imageName + "." + currentExt)
		if err := s.storage.Delete(ctx, oldImagePath); err != nil {
			return err
		}
	}

	s.extensions[imageName] = fileType

	return nil
}

func (s *AppImagesService) DeleteImage(ctx context.Context, imageName string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	ext, ok := s.extensions[imageName]
	if !ok || ext == "" {
		return &common.ImageNotFoundError{}
	}

	imagePath := path.Join("application-images", imageName+"."+ext)
	if err := s.storage.Delete(ctx, imagePath); err != nil {
		return err
	}

	delete(s.extensions, imageName)
	return nil
}

func (s *AppImagesService) IsDefaultProfilePictureSet() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

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
