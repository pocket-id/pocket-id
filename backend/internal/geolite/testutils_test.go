package geolite

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"

	testutils "github.com/pocket-id/pocket-id/backend/internal/utils/testing"
)

// testDownloadURL is the URL the mock HTTP client serves the database from
const testDownloadURL = "https://example.com/geolite/GeoLite2-City.tar.gz"

// testDatabasePath is the sample database published by MaxMind, see testdata/README.md
const testDatabasePath = "testdata/GeoLite2-City-Test.mmdb"

// readTestDatabase returns the raw sample GeoLite2 City database
func readTestDatabase(t *testing.T) []byte {
	t.Helper()

	data, err := os.ReadFile(testDatabasePath)
	require.NoError(t, err)

	return data
}

// testLogger returns a logger that discards everything, so tests don't spam the output
func testLogger() *slog.Logger {
	return slog.New(slog.DiscardHandler)
}

// newServiceForTest returns a Service backed by a database file inside a temporary directory, along with the path of that file
// When data is nil no database is written, so the service starts with nothing to look up against
func newServiceForTest(t *testing.T, data []byte) (*Service, string) {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "GeoLite2-City.mmdb")
	if data != nil {
		writeDatabaseFileForTest(t, dbPath, data)
	}

	svc := newService(testLogger(), dbPath)
	err := svc.load(t.Context())
	require.NoError(t, err)

	return svc, dbPath
}

// writeDatabaseFileForTest puts a database at path the same way the refresher does: written elsewhere, then moved into place
func writeDatabaseFileForTest(t *testing.T, path string, data []byte) {
	t.Helper()

	tmpPath := path + ".tmp"
	err := os.WriteFile(tmpPath, data, 0600)
	require.NoError(t, err)

	err = os.Rename(tmpPath, path)
	require.NoError(t, err)
}

// countingRoundTripper serves a fixed response for testDownloadURL and counts how many requests it has received
type countingRoundTripper struct {
	body       []byte
	statusCode int
	requests   atomic.Int32
}

func (rt *countingRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.URL.String() != testDownloadURL {
		return testutils.NewMockResponse(http.StatusNotFound, ""), nil
	}

	rt.requests.Add(1)

	statusCode := rt.statusCode
	if statusCode == 0 {
		statusCode = http.StatusOK
	}

	return &http.Response{
		StatusCode:    statusCode,
		Body:          io.NopCloser(bytes.NewReader(rt.body)),
		Header:        make(http.Header),
		ContentLength: int64(len(rt.body)),
	}, nil
}

// newDownloadClientForTest returns an HTTP client that serves body at testDownloadURL, along with the transport that counts the requests it receives
func newDownloadClientForTest(body []byte) (*http.Client, *countingRoundTripper) {
	rt := &countingRoundTripper{body: body}
	return &http.Client{Transport: rt}, rt
}

// buildTarGzForTest returns a gzipped tarball holding the given files, mirroring the archive MaxMind publishes
func buildTarGzForTest(t *testing.T, files map[string][]byte) []byte {
	t.Helper()

	buf := &bytes.Buffer{}
	gzw := gzip.NewWriter(buf)
	tw := tar.NewWriter(gzw)

	for name, content := range files {
		err := tw.WriteHeader(&tar.Header{
			Name:     name,
			Mode:     0o644,
			Size:     int64(len(content)),
			Typeflag: tar.TypeReg,
		})
		require.NoError(t, err)

		_, err = tw.Write(content)
		require.NoError(t, err)
	}

	require.NoError(t, tw.Close())
	require.NoError(t, gzw.Close())

	return buf.Bytes()
}
