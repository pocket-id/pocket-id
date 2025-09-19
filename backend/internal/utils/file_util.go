package utils

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
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

// SplitFileName splits a full file name into name and extension.
func SplitFileName(fullName string) (name, ext string) {
	dot := strings.LastIndexByte(fullName, '.')
	if dot == -1 || dot == 0 {
		return fullName, "" // no extension or hidden file like .gitignore
	}
	return fullName[:dot], fullName[dot+1:]
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
	case "webp":
		return "image/webp"
	case "avif":
		return "image/avif"
	case "heic":
		return "image/heic"
	default:
		return ""
	}
}

func CopyEmbeddedFileToDisk(srcFilePath, destFilePath string) error {
	srcFile, err := resources.FS.Open(srcFilePath)
	if err != nil {
		return fmt.Errorf("failed to open embedded file: %w", err)
	}
	defer srcFile.Close()

	err = os.MkdirAll(filepath.Dir(destFilePath), os.ModePerm)
	if err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	destFile, err := os.Create(destFilePath)
	if err != nil {
		return fmt.Errorf("failed to open destination file: %w", err)
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, srcFile)
	if err != nil {
		return fmt.Errorf("failed to write to destination file: %w", err)
	}

	return nil
}

func EmbeddedFileSha256(filePath string) ([]byte, error) {
	f, err := resources.FS.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open embedded file: %w", err)
	}
	defer f.Close()

	h := sha256.New()
	_, err = io.Copy(h, f)
	if err != nil {
		return nil, fmt.Errorf("failed to read embedded file: %w", err)
	}

	return h.Sum(nil), nil
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
	if os.IsNotExist(err) {
		return false, nil
	} else if err != nil {
		return false, fmt.Errorf("failed to stat '%s': %w", dir, err)
	}
	if !info.IsDir() {
		return false, nil
	}

	// Generate a random suffix for the test file to avoid conflicts
	randomBytes := make([]byte, 8)
	_, err = io.ReadFull(rand.Reader, randomBytes)
	if err != nil {
		return false, fmt.Errorf("failed to generate random bytes: %w", err)
	}

	// Check if directory is writable by trying to create a temporary file
	testFile := filepath.Join(dir, ".pocketid_test_write_"+hex.EncodeToString(randomBytes))
	defer os.Remove(testFile)

	file, err := os.Create(testFile)
	if err != nil {
		if os.IsPermission(err) || errors.Is(err, syscall.EROFS) {
			return false, nil
		}

		return false, fmt.Errorf("failed to create test file: %w", err)
	}

	_ = file.Close()

	return true, nil
}
