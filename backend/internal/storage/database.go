package storage

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pocket-id/pocket-id/backend/internal/model"
	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var TypeDatabase = "database"

type databaseStorage struct {
	db *gorm.DB
}

// NewDatabaseStorage creates a new database storage provider
func NewDatabaseStorage(db *gorm.DB) (FileStorage, error) {
	if db == nil {
		return nil, fmt.Errorf("database connection is required")
	}
	return &databaseStorage{db: db}, nil
}

func (s *databaseStorage) Type() string {
	return TypeDatabase
}

func (s *databaseStorage) Save(ctx context.Context, relativePath string, data io.Reader) error {
	// Normalize the path
	relativePath = filepath.ToSlash(filepath.Clean(relativePath))

	// Read all data into memory
	b, err := io.ReadAll(data)
	if err != nil {
		return fmt.Errorf("failed to read data: %w", err)
	}

	now := datatype.DateTime(time.Now())
	storage := model.Storage{
		Path:      relativePath,
		Data:      b,
		Size:      int64(len(b)),
		ModTime:   now,
		CreatedAt: now,
	}

	// Use upsert: insert or update on conflict
	result := s.db.
		WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "path"}},
			DoUpdates: clause.AssignmentColumns([]string{"data", "size", "mod_time"}),
		}).
		Create(&storage)

	if result.Error != nil {
		return fmt.Errorf("failed to save file to database: %w", result.Error)
	}

	return nil
}

func (s *databaseStorage) Open(ctx context.Context, relativePath string) (io.ReadCloser, int64, error) {
	relativePath = filepath.ToSlash(filepath.Clean(relativePath))

	var storage model.Storage
	result := s.db.
		WithContext(ctx).
		Where("path = ?", relativePath).
		First(&storage)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, 0, os.ErrNotExist
		}
		return nil, 0, fmt.Errorf("failed to read file from database: %w", result.Error)
	}

	reader := io.NopCloser(bytes.NewReader(storage.Data))
	return reader, storage.Size, nil
}

func (s *databaseStorage) Delete(ctx context.Context, relativePath string) error {
	relativePath = filepath.ToSlash(filepath.Clean(relativePath))

	result := s.db.
		WithContext(ctx).
		Where("path = ?", relativePath).
		Delete(&model.Storage{})
	if result.Error != nil {
		return fmt.Errorf("failed to delete file from database: %w", result.Error)
	}

	return nil
}

func (s *databaseStorage) DeleteAll(ctx context.Context, prefix string) error {
	prefix = filepath.ToSlash(filepath.Clean(prefix))

	// If empty prefix, delete all
	if prefix == "" || prefix == "/" || prefix == "." {
		result := s.db.
			WithContext(ctx).
			Where("1 = 1"). // Delete everything
			Delete(&model.Storage{})
		if result.Error != nil {
			return fmt.Errorf("failed to delete all files from database: %w", result.Error)
		}
		return nil
	}

	// Ensure prefix ends with / for proper prefix matching
	if !strings.HasSuffix(prefix, "/") {
		prefix = prefix + "/"
	}

	query := s.db.WithContext(ctx)
	query = addPathPrefixClause(s.db.Name(), query, prefix)
	result := query.Delete(&model.Storage{})
	if result.Error != nil {
		return fmt.Errorf("failed to delete files with prefix '%s' from database: %w", prefix, result.Error)
	}

	return nil
}

func (s *databaseStorage) List(ctx context.Context, prefix string) ([]ObjectInfo, error) {
	prefix = filepath.ToSlash(filepath.Clean(prefix))

	var storageItems []model.Storage
	query := s.db.WithContext(ctx)

	if prefix != "" && prefix != "/" && prefix != "." {
		// Ensure prefix matching
		if !strings.HasSuffix(prefix, "/") {
			prefix = prefix + "/"
		}
		query = addPathPrefixClause(s.db.Name(), query, prefix)
	}

	result := query.
		Select("path", "size", "mod_time").
		Find(&storageItems)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to list files from database: %w", result.Error)
	}

	objects := make([]ObjectInfo, 0, len(storageItems))
	for _, item := range storageItems {
		// Filter out directory-like paths (those that contain additional slashes after the prefix)
		relativePath := strings.TrimPrefix(item.Path, prefix)
		if strings.ContainsRune(relativePath, '/') {
			continue
		}

		objects = append(objects, ObjectInfo{
			Path:    item.Path,
			Size:    item.Size,
			ModTime: time.Time(item.ModTime),
		})
	}

	return objects, nil
}

func (s *databaseStorage) Walk(ctx context.Context, root string, fn func(ObjectInfo) error) error {
	root = filepath.ToSlash(filepath.Clean(root))

	var storageItems []model.Storage
	query := s.db.WithContext(ctx)

	if root != "" && root != "/" && root != "." {
		// Ensure root matching
		if !strings.HasSuffix(root, "/") {
			root = root + "/"
		}
		query = addPathPrefixClause(s.db.Name(), query, root)
	}

	result := query.
		Select("path", "size", "mod_time").
		Find(&storageItems)
	if result.Error != nil {
		return fmt.Errorf("failed to walk files from database: %w", result.Error)
	}

	for _, item := range storageItems {
		err := fn(ObjectInfo{
			Path:    item.Path,
			Size:    item.Size,
			ModTime: time.Time(item.ModTime),
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func addPathPrefixClause(dialect string, query *gorm.DB, prefix string) *gorm.DB {
	// In SQLite, we use "GLOB" which can use the index
	switch dialect {
	case "sqlite":
		return query.Where("path GLOB ?", prefix+"*")
	case "postgres":
		return query.Where("path LIKE ?", prefix+"%")
	default:
		// Indicates a development-time error
		panic(fmt.Errorf("unsupported database dialect: %s", dialect))
	}
}
