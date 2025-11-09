package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

type filesystemStorage struct {
	root             *os.Root
	absoluteRootPath string
}

func NewFilesystemStorage(rootPath string) (FileStorage, error) {
	if err := os.MkdirAll(rootPath, 0700); err != nil {
		return nil, fmt.Errorf("failed to create root directory '%s': %w", rootPath, err)
	}
	root, err := os.OpenRoot(rootPath)

	absoluteRootPath, err := filepath.Abs(rootPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path of root directory '%s': %w", rootPath, err)
	}

	return &filesystemStorage{root: root, absoluteRootPath: absoluteRootPath}, err
}

func (s *filesystemStorage) Type() string {
	return TypeFileSystem
}

func (s *filesystemStorage) Save(_ context.Context, path string, data io.Reader) error {
	path = filepath.FromSlash(path)

	if err := s.root.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return fmt.Errorf("failed to create directories for path '%s': %w", path, err)
	}

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

	// If "/", "." or "" is requested, we delete all contents of the root.
	if path == "" || path == "/" || path == "." {
		dir, err := s.root.Open(".")
		if err != nil {
			return fmt.Errorf("failed to open root directory: %w", err)
		}
		defer dir.Close()

		entries, err := dir.ReadDir(-1)
		if err != nil {
			return fmt.Errorf("failed to list root directory: %w", err)
		}
		for _, entry := range entries {
			if err := s.root.RemoveAll(entry.Name()); err != nil {
				return fmt.Errorf("failed to delete '%s': %w", entry.Name(), err)
			}
		}
		return nil
	}

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
			Path:    filepath.Join(path, entry.Name()),
			Size:    info.Size(),
			ModTime: info.ModTime(),
		})
	}
	return objects, nil
}
func (s *filesystemStorage) Walk(_ context.Context, root string, fn func(ObjectInfo) error) error {
	root = filepath.FromSlash(root)

	fullPath := filepath.Clean(filepath.Join(s.absoluteRootPath, root))

	// As we can't use os.Root here, we manually ensure that the fullPath is within the root directory
	sep := string(filepath.Separator)
	if !strings.HasPrefix(fullPath+sep, s.absoluteRootPath+sep) {
		return fmt.Errorf("invalid root path: %s", root)
	}

	return filepath.WalkDir(fullPath, func(full string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(s.absoluteRootPath, full)
		if err != nil {
			return err
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		return fn(ObjectInfo{
			Path:    filepath.ToSlash(rel),
			Size:    info.Size(),
			ModTime: info.ModTime(),
		})
	})
}
