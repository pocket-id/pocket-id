package geolite

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/oschwald/maxminddb-golang/v2"
)

const (
	// databaseFileName is the name of the database inside the archive published by MaxMind
	databaseFileName = "GeoLite2-City.mmdb"
	// maxDatabaseSize is the largest (decompressed) database we accept
	maxDatabaseSize = 300 << 20 // 300 MB
)

// downloadDatabase downloads the GeoLite2 City database and puts it at targetPath
// The database is streamed to a temporary file next to the target and moved into place only once it has been verified, atomically
func downloadDatabase(ctx context.Context, httpClient *http.Client, downloadURL string, licenseKey string, targetPath string) error {
	// When downloadURL contains a "%s" placeholder, it is replaced with the license key
	if strings.Contains(downloadURL, "%s") {
		downloadURL = fmt.Sprintf(downloadURL, licenseKey)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	res, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download database: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download database, received HTTP %d", res.StatusCode)
	}

	err = writeDatabase(res.Body, targetPath)
	if err != nil {
		return err
	}

	return nil
}

// writeDatabase extracts the database from a downloaded body and puts it at targetPath
func writeDatabase(body io.Reader, targetPath string) error {
	baseDir := filepath.Dir(targetPath)
	err := os.MkdirAll(baseDir, 0700)
	if err != nil {
		return fmt.Errorf("failed to create the database directory: %w", err)
	}

	tmpFile, err := os.CreateTemp(baseDir, "geolite.*.mmdb.tmp")
	if err != nil {
		return fmt.Errorf("failed to create temporary database file: %w", err)
	}
	tmpName := tmpFile.Name()

	// Remove the temporary file unless it has been moved into place
	moved := false
	defer func() {
		tmpFile.Close()
		if !moved {
			os.Remove(tmpName)
		}
	}()

	err = extractDatabase(body, tmpFile)
	if err != nil {
		return fmt.Errorf("failed to extract database: %w", err)
	}

	err = tmpFile.Close()
	if err != nil {
		return fmt.Errorf("failed to write database file: %w", err)
	}

	// Make sure the database isn't corrupted before it replaces the one currently in place
	db, err := maxminddb.Open(tmpName)
	if err != nil {
		return fmt.Errorf("failed to open downloaded database: %w", err)
	}
	_ = db.Close()

	err = os.Rename(tmpName, targetPath)
	if err != nil {
		return fmt.Errorf("failed to replace database file: %w", err)
	}
	moved = true

	return nil
}

// extractDatabase copies the raw MaxMind DB file out of a downloaded body and into dst
// The body is either the gzipped tarball published by MaxMind, or the database file itself
func extractDatabase(body io.Reader, dst io.Writer) error {
	// A buffered reader lets the gzip magic number be checked without consuming it
	reader := bufio.NewReader(body)

	magic, err := reader.Peek(2)
	if err != nil {
		return fmt.Errorf("failed to read magic number: %w", err)
	}

	// If the body doesn't start with the gzip magic number, assume it's a plain database file
	// Gosec returns false positive for "G602: slice index out of range"
	//nolint:gosec
	if magic[0] != 0x1f || magic[1] != 0x8b {
		return copyDatabase(dst, reader)
	}

	gzr, err := gzip.NewReader(reader)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzr.Close()

	tarReader := tar.NewReader(gzr)
	for {
		header, err := tarReader.Next()
		if errors.Is(err, io.EOF) {
			break
		} else if err != nil {
			return fmt.Errorf("failed to read tar archive: %w", err)
		}

		// The archive contains the database in a versioned folder, alongside other files such as the license
		if header.Typeflag != tar.TypeReg || path.Base(header.Name) != databaseFileName {
			continue
		}

		err = copyDatabase(dst, tarReader)
		if err != nil {
			return err
		}
		return nil
	}

	return errors.New(databaseFileName + " not found in archive")
}

// copyDatabase streams the database into dst, refusing anything larger than maxDatabaseSize
func copyDatabase(dst io.Writer, src io.Reader) error {
	return copyDatabaseWithLimit(dst, src, maxDatabaseSize)
}

// copyDatabaseWithLimit is copyDatabase with an explicit limit, which lets tests exercise the limit without streaming hundreds of megabytes
func copyDatabaseWithLimit(dst io.Writer, src io.Reader, limit int64) error {
	// Copy one byte more than the limit, so content that is exactly at the limit can be told apart from content that exceeds it
	written, err := io.Copy(dst, io.LimitReader(src, limit+1))
	if err != nil {
		return fmt.Errorf("failed to read database: %w", err)
	}

	if written > limit {
		return errors.New("database size exceeds maximum allowed limit")
	}

	return nil
}
