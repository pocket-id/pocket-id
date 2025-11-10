package utils

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"mime"
	"os"
	"path/filepath"
	"strings"
	"syscall"
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
	dot := strings.LastIndex(fullName, ".")
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

func GetImageExtensionFromMimeType(mimeType string) string {
	// Normalize and strip parameters like `; charset=utf-8`
	mt := strings.TrimSpace(strings.ToLower(mimeType))
	if v, _, err := mime.ParseMediaType(mt); err == nil {
		mt = v
	}
	switch mt {
	case "image/jpeg", "image/jpg":
		return "jpg"
	case "image/png":
		return "png"
	case "image/svg+xml":
		return "svg"
	case "image/x-icon", "image/vnd.microsoft.icon":
		return "ico"
	case "image/gif":
		return "gif"
	case "image/webp":
		return "webp"
	case "image/avif":
		return "avif"
	case "image/heic", "image/heif":
		return "heic"
	default:
		return ""
	}
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
