package appassets

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"path"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/pocket-id/pocket-id/backend/internal/apperror"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	"github.com/pocket-id/pocket-id/backend/internal/storage"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
	imageutil "github.com/pocket-id/pocket-id/backend/internal/utils/image"
	"github.com/pocket-id/pocket-id/backend/resources"
)

// Service keeps the file of each asset in the database rather than in memory, so every replica serves the latest upload
type Service struct {
	db      *gorm.DB
	storage storage.FileStorage
}

func (s *Service) GetAsset(ctx context.Context, name string) (io.ReadCloser, int64, string, error) {
	var row model.KV
	err := s.db.WithContext(ctx).Where("key = ?", assetKeyPrefix+name).Limit(1).Find(&row).Error
	if err != nil {
		return nil, 0, "", fmt.Errorf("failed to load application asset '%s': %w", name, err)
	}
	if row.Value == nil || *row.Value == "" {
		return nil, 0, "", apperror.ImageNotFound()
	}

	ext := utils.GetFileExtension(*row.Value)
	mimeType := fileMimeType(name, strings.ToLower(ext))
	if mimeType == "" {
		return nil, 0, "", fmt.Errorf("unsupported image type '%s'", ext)
	}

	reader, size, err := s.storage.Open(ctx, path.Join(applicationAssetsPath, *row.Value))
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

	// Every upload gets its own file, so a concurrent upload never overwrites a file that another one is about to delete
	suffix, err := utils.GenerateRandomAlphanumericString(16)
	if err != nil {
		return err
	}
	assetFile := path.Join(assetName, suffix+"."+fileType)
	if err := s.storage.Save(ctx, path.Join(applicationAssetsPath, assetFile), strippedReader); err != nil {
		return err
	}

	// Switch to the new file before removing the previous one
	previousFile, err := s.swapFile(ctx, assetName, assetFile)
	if err != nil {
		return err
	}
	if previousFile != "" {
		return s.storage.Delete(ctx, path.Join(applicationAssetsPath, previousFile))
	}

	return nil
}

func (s *Service) DeleteAsset(ctx context.Context, assetName string) error {
	// The empty value keeps Init from copying the bundled image back on the next start
	previousFile, err := s.swapFile(ctx, assetName, "")
	if err != nil {
		return err
	}
	if previousFile == "" {
		return apperror.ImageNotFound()
	}

	return s.storage.Delete(ctx, path.Join(applicationAssetsPath, previousFile))
}

// swapFile records the current file of an asset and returns the previous one
// The row lock serializes concurrent uploads and deletions, so each one removes exactly the file it replaced
func (s *Service) swapFile(ctx context.Context, name, file string) (previousFile string, err error) {
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Create the row first, since a missing row can't be locked
		row := model.KV{Key: assetKeyPrefix + name, Value: new("")}
		err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&row).Error
		if err != nil {
			return err
		}

		err = tx.Clauses(clause.Locking{Strength: clause.LockingStrengthUpdate}).First(&row, "key = ?", row.Key).Error
		if err != nil {
			return err
		}
		if row.Value != nil {
			previousFile = *row.Value
		}

		return tx.Model(&model.KV{}).Where("key = ?", row.Key).Update("value", file).Error
	})
	if err != nil {
		return "", fmt.Errorf("failed to record application asset '%s': %w", name, err)
	}
	return previousFile, nil
}

// The title font is the one application file that isn't an image
func fileMimeType(name, ext string) string {
	if name == "font" {
		return utils.GetFontMimeType(ext)
	}
	return utils.GetImageMimeType(ext)
}
