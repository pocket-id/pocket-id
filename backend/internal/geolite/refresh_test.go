package geolite

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// newRefresherForTest returns a refresher wired to a service backed by a database file inside dir
func newRefresherForTest(t *testing.T, dir string, httpClient *http.Client) *refresher {
	t.Helper()

	svc := newServiceAtPathForTest(t, filepath.Join(dir, "GeoLite2-City.mmdb"))

	return &refresher{
		log:         testLogger(),
		service:     svc,
		httpClient:  httpClient,
		downloadURL: testDownloadURL,
		watching:    make(chan struct{}),
	}
}

// runRefresherForTest starts the refresher and waits until its watcher is established, so a change made afterwards is guaranteed to be noticed
// The refresher is stopped when the test ends
func runRefresherForTest(t *testing.T, r *refresher) context.Context {
	t.Helper()

	ctx, cancel := context.WithCancel(t.Context())

	done := make(chan error, 1)
	go func() {
		done <- r.Run(ctx)
	}()

	t.Cleanup(func() {
		cancel()
		require.NoError(t, <-done)
	})

	select {
	case <-r.watching:
	case <-time.After(30 * time.Second):
		t.Fatal("timed out waiting for the database watcher to start")
	}

	return ctx
}

func TestRefresherTimeUntilRefresh(t *testing.T) {
	t.Run("missing database", func(t *testing.T) {
		r := newRefresherForTest(t, t.TempDir(), nil)
		require.Zero(t, r.timeUntilRefresh())
	})

	t.Run("fresh database", func(t *testing.T) {
		dir := t.TempDir()
		r := newRefresherForTest(t, dir, nil)
		writeDatabaseFileForTest(t, r.service.dbPath, readTestDatabase(t))

		// The wait is the remaining lifetime of the file, give or take the jitter
		delay := r.timeUntilRefresh()
		require.InDelta(t, databaseMaxAge, delay, float64(refreshJitter+time.Minute))
		require.Positive(t, delay)
	})

	t.Run("stale database", func(t *testing.T) {
		dir := t.TempDir()
		r := newRefresherForTest(t, dir, nil)
		writeDatabaseFileForTest(t, r.service.dbPath, readTestDatabase(t))

		aged := time.Now().Add(-databaseMaxAge - time.Hour)
		err := os.Chtimes(r.service.dbPath, aged, aged)
		require.NoError(t, err)

		require.Zero(t, r.timeUntilRefresh())
	})
}

func TestRefresherRefresh(t *testing.T) {
	database := readTestDatabase(t)
	archive := buildTarGzForTest(t, map[string][]byte{"GeoLite2-City_20260101/" + databaseFileName: database})
	httpClient, transport := newDownloadClientForTest(archive)

	r := newRefresherForTest(t, t.TempDir(), httpClient)

	// Nothing to look up against before the first refresh
	country, _, err := r.service.GetLocationByIP(t.Context(), "81.2.69.142")
	require.NoError(t, err)
	require.Empty(t, country)

	err = r.refresh(t.Context())
	require.NoError(t, err)
	require.Equal(t, int32(1), transport.requests.Load())

	// The database is on disk, and the service is serving from it without waiting for the watcher
	require.FileExists(t, r.service.dbPath)
	country, city, err := r.service.GetLocationByIP(t.Context(), "81.2.69.142")
	require.NoError(t, err)
	require.Equal(t, "United Kingdom", country)
	require.Equal(t, "London", city)

	// The refresh it just performed pushes the next one out by the full lifetime of the database
	require.Positive(t, r.timeUntilRefresh())
}

func TestRefresherRefreshFailure(t *testing.T) {
	httpClient, transport := newDownloadClientForTest(nil)
	transport.statusCode = http.StatusInternalServerError

	r := newRefresherForTest(t, t.TempDir(), httpClient)

	err := r.refresh(t.Context())
	require.Error(t, err)
	require.ErrorContains(t, err, "received HTTP 500")

	// No file is left behind, so the next attempt is still due right away
	require.NoFileExists(t, r.service.dbPath)
	require.Zero(t, r.timeUntilRefresh())
}

func TestRefresherSharedDirectorySkipsDownload(t *testing.T) {
	// Replicas pointed at the same mount coordinate through the file itself: once one of them has refreshed it, the others find it fresh and don't download it again
	database := readTestDatabase(t)
	archive := buildTarGzForTest(t, map[string][]byte{"GeoLite2-City_20260101/" + databaseFileName: database})
	httpClient, transport := newDownloadClientForTest(archive)

	dir := t.TempDir()
	first := newRefresherForTest(t, dir, httpClient)
	second := newRefresherForTest(t, dir, httpClient)

	require.Zero(t, first.timeUntilRefresh())
	require.Zero(t, second.timeUntilRefresh())

	err := first.refresh(t.Context())
	require.NoError(t, err)
	require.Equal(t, int32(1), transport.requests.Load())

	// The second replica sees the file the first one wrote and goes back to sleep instead of downloading it again
	require.Positive(t, second.timeUntilRefresh())
}

func TestRefresherWatchesForReplacedDatabase(t *testing.T) {
	// Supplying a database by hand is how air-gapped deployments work, and it takes effect without a restart
	r := newRefresherForTest(t, t.TempDir(), nil)
	r.disabled = true

	ctx := runRefresherForTest(t, r)

	country, _, err := r.service.GetLocationByIP(ctx, "81.2.69.142")
	require.NoError(t, err)
	require.Empty(t, country)

	writeDatabaseFileForTest(t, r.service.dbPath, readTestDatabase(t))

	require.Eventually(t, func() bool {
		country, _, err := r.service.GetLocationByIP(ctx, "81.2.69.142")
		return err == nil && country == "United Kingdom"
	}, 30*time.Second, 100*time.Millisecond, "the database put in place by hand was never picked up")

	// Removing it stops lookups from resolving, rather than serving from a file that is gone
	err = os.Remove(r.service.dbPath)
	require.NoError(t, err)

	require.Eventually(t, func() bool {
		country, _, err := r.service.GetLocationByIP(ctx, "81.2.69.142")
		return err == nil && country == ""
	}, 30*time.Second, 100*time.Millisecond, "the removed database was still being served")
}

func TestRefresherRunRefreshesOnStart(t *testing.T) {
	database := readTestDatabase(t)
	archive := buildTarGzForTest(t, map[string][]byte{"GeoLite2-City_20260101/" + databaseFileName: database})
	httpClient, transport := newDownloadClientForTest(archive)

	r := newRefresherForTest(t, t.TempDir(), httpClient)
	ctx := runRefresherForTest(t, r)

	// A missing database is due right away, so the refresher downloads one as soon as it starts
	require.Eventually(t, func() bool {
		country, _, err := r.service.GetLocationByIP(ctx, "81.2.69.142")
		return err == nil && country == "United Kingdom"
	}, 30*time.Second, 100*time.Millisecond, "the database was never downloaded")

	// It doesn't download again once the database on disk is fresh
	require.Never(t, func() bool {
		return transport.requests.Load() > 1
	}, 3*time.Second, 250*time.Millisecond, "the database was downloaded again while it was still fresh")
}

func TestRefresherDisabledDoesNotDownload(t *testing.T) {
	// Without a way to reach the download URL there's nothing to refresh, but the file is still watched
	httpClient, transport := newDownloadClientForTest(nil)

	r := newRefresherForTest(t, t.TempDir(), httpClient)
	r.disabled = true

	runRefresherForTest(t, r)

	require.Never(t, func() bool {
		return transport.requests.Load() > 0
	}, 3*time.Second, 250*time.Millisecond, "the database was downloaded even though refreshes are disabled")
}
