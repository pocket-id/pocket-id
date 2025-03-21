package utils

import (
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/cespare/xxhash/v2"

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
	// First, get a temp file name that doesn't exist already
	var tmpFileName string
	for {
		suffix := xxhash.Sum64String(dstFileName + strconv.FormatInt(time.Now().UnixMicro(), 10))
		tmpFileName = dstFileName + "." + strconv.FormatUint(suffix, 10)
		exists, err := FileExists(tmpFileName)
		if err != nil {
			return fmt.Errorf("failed to check if file '%s' exists: %w", tmpFileName, err)
		}
		if !exists {
			break
		}
	}

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
