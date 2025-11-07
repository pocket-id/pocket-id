package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/google/uuid"
)

type filesystemStorage struct {
	root *os.Root
}

func newFilesystemStorage(rootPath string) (FileStorage, error) {
	root, err := os.OpenRoot(rootPath)
	return &filesystemStorage{root: root}, err
}

func (s *filesystemStorage) Save(_ context.Context, path string, data io.Reader) error {
	path = filepath.FromSlash(path)

	// Our strategy is to save to a separate file and then rename it to override the original file
	tmpName := path + "." + uuid.NewString() + "-tmp"

	// Write to the temporary file
	tmpFile, err := s.root.Create(tmpName)
	if err != nil {
		return fmt.Errorf("failed to open file '%s' for writing: %w", tmpName, err)
	}

	_, err = io.Copy(tmpFile, data)
	if err != nil {
		tmpFile.Close()
		_ = s.root.Remove(tmpName)
		return fmt.Errorf("failed to write temporary file: %w", err)
	}

	if err = tmpFile.Close(); err != nil {
		_ = s.root.Remove(tmpName)
		return fmt.Errorf("failed to close temporary file: %w", err)
	}

	// Rename to the final file, which overrides existing files
	// This is an atomic operation
	if err = s.root.Rename(tmpName, path); err != nil {
		_ = s.root.Remove(tmpName)
		return fmt.Errorf("failed to move temporary file: %w", err)
	}

	return nil
}

func (s *filesystemStorage) Open(_ context.Context, path string) (io.ReadCloser, int64, error) {
	path = filepath.FromSlash(path)

	file, err := s.root.Open(path)
	if err != nil {
		return nil, 0, err
	}
	info, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, 0, err
	}
	return file, info.Size(), nil
}

func (s *filesystemStorage) Delete(_ context.Context, path string) error {
	path = filepath.FromSlash(path)

	err := s.root.Remove(path)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	return nil
}

func (s *filesystemStorage) DeleteAll(_ context.Context, path string) error {
	path = filepath.FromSlash(path)

	return s.root.RemoveAll(path)
}
func (s *filesystemStorage) List(_ context.Context, path string) ([]ObjectInfo, error) {
	path = filepath.FromSlash(path)

	dir, err := s.root.Open(path)
	if err != nil {
		return nil, err
	}
	defer dir.Close()

	entries, err := dir.ReadDir(-1)
	if err != nil {
		return nil, err
	}

	objects := make([]ObjectInfo, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			return nil, err
		}
		objects = append(objects, ObjectInfo{
			Path: filepath.Join(path, entry.Name()),
			Size: info.Size(),
		})
	}
	return objects, nil
}
