package geolite

import (
	"bytes"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExtractDatabase(t *testing.T) {
	database := readTestDatabase(t)

	t.Run("gzipped tarball", func(t *testing.T) {
		archive := buildTarGzForTest(t, map[string][]byte{
			"GeoLite2-City_20260101/COPYRIGHT.txt":       []byte("copyright"),
			"GeoLite2-City_20260101/LICENSE.txt":         []byte("license"),
			"GeoLite2-City_20260101/" + databaseFileName: database,
		})

		dst := &bytes.Buffer{}
		err := extractDatabase(bytes.NewReader(archive), dst)
		require.NoError(t, err)
		require.Equal(t, database, dst.Bytes())
	})

	t.Run("plain database file", func(t *testing.T) {
		// A custom GEOLITE_DB_URL may serve the database uncompressed
		dst := &bytes.Buffer{}
		err := extractDatabase(bytes.NewReader(database), dst)
		require.NoError(t, err)
		require.Equal(t, database, dst.Bytes())
	})

	t.Run("tarball without the database", func(t *testing.T) {
		archive := buildTarGzForTest(t, map[string][]byte{
			"GeoLite2-City_20260101/COPYRIGHT.txt": []byte("copyright"),
		})

		err := extractDatabase(bytes.NewReader(archive), io.Discard)
		require.Error(t, err)
		require.ErrorContains(t, err, "not found in archive")
	})

	t.Run("truncated gzip stream", func(t *testing.T) {
		archive := buildTarGzForTest(t, map[string][]byte{"GeoLite2-City_20260101/" + databaseFileName: database})

		err := extractDatabase(bytes.NewReader(archive[:len(archive)/2]), io.Discard)
		require.Error(t, err)
	})

	t.Run("empty body", func(t *testing.T) {
		err := extractDatabase(strings.NewReader(""), io.Discard)
		require.Error(t, err)
		require.ErrorContains(t, err, "failed to read magic number")
	})
}

func TestCopyDatabase(t *testing.T) {
	t.Run("exactly at the limit", func(t *testing.T) {
		dst := &bytes.Buffer{}
		err := copyDatabaseWithLimit(dst, bytes.NewReader(bytes.Repeat([]byte{0x01}, 1024)), 1024)
		require.NoError(t, err)
		require.Equal(t, 1024, dst.Len())
	})

	t.Run("over the limit", func(t *testing.T) {
		err := copyDatabaseWithLimit(io.Discard, endlessReader{}, 1024)
		require.Error(t, err)
		require.ErrorContains(t, err, "exceeds maximum allowed limit")
	})
}

func TestDownloadDatabase(t *testing.T) {
	database := readTestDatabase(t)
	archive := buildTarGzForTest(t, map[string][]byte{"GeoLite2-City_20260101/" + databaseFileName: database})

	t.Run("success", func(t *testing.T) {
		httpClient, transport := newDownloadClientForTest(archive)
		targetPath := filepath.Join(t.TempDir(), "GeoLite2-City.mmdb")

		err := downloadDatabase(t.Context(), httpClient, testDownloadURL, "", targetPath)
		require.NoError(t, err)
		require.Equal(t, int32(1), transport.requests.Load())

		written, err := os.ReadFile(targetPath)
		require.NoError(t, err)
		require.Equal(t, database, written)
	})

	t.Run("creates the target directory", func(t *testing.T) {
		httpClient, _ := newDownloadClientForTest(archive)
		targetPath := filepath.Join(t.TempDir(), "nested", "data", "GeoLite2-City.mmdb")

		err := downloadDatabase(t.Context(), httpClient, testDownloadURL, "", targetPath)
		require.NoError(t, err)
		require.FileExists(t, targetPath)
	})

	t.Run("license key placeholder", func(t *testing.T) {
		// The default MaxMind URL carries the license key, which is filled in at download time
		var requestedURL string
		httpClient := &http.Client{
			Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
				requestedURL = req.URL.String()
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewReader(archive)),
					Header:     make(http.Header),
				}, nil
			}),
		}

		err := downloadDatabase(t.Context(), httpClient, "https://example.com/download?license_key=%s", "secret-key", filepath.Join(t.TempDir(), "db.mmdb"))
		require.NoError(t, err)
		require.Equal(t, "https://example.com/download?license_key=secret-key", requestedURL)
	})

	t.Run("non-200 response", func(t *testing.T) {
		httpClient, transport := newDownloadClientForTest(nil)
		transport.statusCode = http.StatusUnauthorized
		targetPath := filepath.Join(t.TempDir(), "GeoLite2-City.mmdb")

		err := downloadDatabase(t.Context(), httpClient, testDownloadURL, "", targetPath)
		require.Error(t, err)
		require.ErrorContains(t, err, "received HTTP 401")
		require.NoFileExists(t, targetPath)
	})

	t.Run("corrupted database leaves the existing file in place", func(t *testing.T) {
		corrupted := buildTarGzForTest(t, map[string][]byte{
			"GeoLite2-City_20260101/" + databaseFileName: []byte("not a database"),
		})
		httpClient, _ := newDownloadClientForTest(corrupted)

		dir := t.TempDir()
		targetPath := filepath.Join(dir, "GeoLite2-City.mmdb")
		require.NoError(t, os.WriteFile(targetPath, database, 0600))

		err := downloadDatabase(t.Context(), httpClient, testDownloadURL, "", targetPath)
		require.Error(t, err)
		require.ErrorContains(t, err, "failed to open downloaded database")

		// The database that was already there is untouched, and no temporary file is left behind
		existing, err := os.ReadFile(targetPath)
		require.NoError(t, err)
		require.Equal(t, database, existing)

		entries, err := os.ReadDir(dir)
		require.NoError(t, err)
		require.Len(t, entries, 1)
	})
}

// endlessReader returns an unbounded stream of zeroes, to exercise the download size limit
type endlessReader struct{}

func (endlessReader) Read(p []byte) (int, error) {
	clear(p)
	return len(p), nil
}

type roundTripperFunc func(req *http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}
