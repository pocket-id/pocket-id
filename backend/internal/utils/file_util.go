package utils

import (
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"syscall"

	"github.com/google/uuid"
	"github.com/pocket-id/pocket-id/backend/resources"
)

func GetFileExtension(filename string) string {
	ext := filepath.Ext(filename)
	if len(ext) > 0 && ext[0] == '.' {
		return ext[1:]
	}
	return filename
}

func GetImageMimeType(ext string) string {
	switch ext {
	case "jpg", "jpeg":
		return "image/jpeg"
	case "png":
		return "image/png"
	case "svg":
		return "image/svg+xml"
	case "ico":
		return "image/x-icon"
	case "gif":
		return "image/gif"
	default:
		return ""
	}
}

func CopyEmbeddedFileToDisk(srcFilePath, destFilePath string) error {
	srcFile, err := resources.FS.Open(srcFilePath)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	err = os.MkdirAll(filepath.Dir(destFilePath), os.ModePerm)
	if err != nil {
		return err
	}

	destFile, err := os.Create(destFilePath)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, srcFile)
	if err != nil {
		return err
	}

	return nil
}

func SaveFile(file *multipart.FileHeader, dst string) error {
	src, err := file.Open()
	if err != nil {
		return err
	}
	defer src.Close()

	if err = os.MkdirAll(filepath.Dir(dst), 0o750); err != nil {
		return err
	}

	return SaveFileStream(src, dst)
}

// SaveFileStream saves a stream to a file.
func SaveFileStream(r io.Reader, dstFileName string) error {
	// Our strategy is to save to a separate file and then rename it to override the original file
	tmpFileName := dstFileName + "." + uuid.NewString() + "-tmp"

	// Write to the temporary file
	tmpFile, err := os.Create(tmpFileName)
	if err != nil {
		return fmt.Errorf("failed to open file '%s' for writing: %w", tmpFileName, err)
	}

	n, err := io.Copy(tmpFile, r)
	if err != nil {
		// Delete the temporary file; we ignore errors here
		_ = tmpFile.Close()
		_ = os.Remove(tmpFileName)

		return fmt.Errorf("failed to write to file '%s': %w", tmpFileName, err)
	}

	err = tmpFile.Close()
	if err != nil {
		// Delete the temporary file; we ignore errors here
		_ = os.Remove(tmpFileName)

		return fmt.Errorf("failed to close stream to file '%s': %w", tmpFileName, err)
	}

	if n == 0 {
		// Delete the temporary file; we ignore errors here
		_ = os.Remove(tmpFileName)

		return errors.New("no data written")
	}

	// Rename to the final file, which overrides existing files
	// This is an atomic operation
	err = os.Rename(tmpFileName, dstFileName)
	if err != nil {
		// Delete the temporary file; we ignore errors here
		_ = os.Remove(tmpFileName)

		return fmt.Errorf("failed to rename file '%s': %w", dstFileName, err)
	}

	return nil
}

// FileExists returns true if a file exists on disk and is a regular file
func FileExists(path string) (bool, error) {
	s, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			err = nil
		}
		return false, err
	}
	return !s.IsDir(), nil
}

// IsWritableDir checks if a directory exists and is writable
func IsWritableDir(dir string) (bool, error) {
	// Check if directory exists and it's actually a directory
	info, err := os.Stat(dir)
	if err != nil {
		return false, nil
	}
	if !info.IsDir() {
		return false, nil
	}

	// Generate a random suffix for the test file to avoid conflicts
	randomBytes := make([]byte, 8)
	if err != nil {
		return false, fmt.Errorf("failed to generate random bytes: %w", err)
	}

	// Check if directory is writable by trying to create a temporary file
	testFile := dir + "/.pocketid_test_write_" + hex.EncodeToString(randomBytes)
	defer os.Remove(testFile)

	file, err := os.Create(testFile)
	if err != nil {
		if os.IsPermission(err) || errors.Is(err, syscall.EROFS) {
			return false, nil
		}

		return false, fmt.Errorf("failed to create test file: %w", err)
	}

	file.Close()

	return true, nil
}
