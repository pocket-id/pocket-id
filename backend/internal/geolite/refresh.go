package geolite

import (
	"context"
	"errors"
	"log/slog"
	"math/rand/v2"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
)

const (
	// databaseMaxAge is how old the database on disk is allowed to get before it's downloaded again
	databaseMaxAge = 14 * 24 * time.Hour
	// refreshJitter is subtracted or added at random when scheduling a refresh
	// On a shared mount it keeps several replicas from waking up together and racing to download the same file
	refreshJitter = 30 * time.Minute
	// refreshRetryInterval is how long the refresher waits before trying again after a failed download
	refreshRetryInterval = time.Hour
	// downloadTimeout bounds a single download of the database
	downloadTimeout = 10 * time.Minute
	// watcherDebounce is how long the watcher waits for the file to settle before reloading it
	watcherDebounce = time.Second
)

// refresher keeps the database at dbPath up-to-date and reloads the service when the file changes
type refresher struct {
	log        *slog.Logger
	service    *Service
	httpClient *http.Client

	downloadURL string
	licenseKey  string
	// disabled stops the periodic download, for deployments that have no way to reach the download URL
	// The file is still watched, since it may be supplied by hand
	disabled bool

	// watching is closed once the watcher is established, if set
	// It's a hook for tests, which need to know when a change to the file is guaranteed to be noticed
	watching chan struct{}
}

// Run watches the database file and periodically refreshes it, until the context is canceled
// It satisfies servicerunner.Service
func (r *refresher) Run(ctx context.Context) error {
	watcherDone := make(chan struct{})
	go func() {
		defer close(watcherDone)
		r.watch(ctx)
	}()

	if !r.disabled {
		r.refreshPeriodically(ctx)
	} else {
		<-ctx.Done()
	}

	<-watcherDone

	return nil
}

// refreshPeriodically downloads the database whenever the one on disk has aged past databaseMaxAge
func (r *refresher) refreshPeriodically(ctx context.Context) {
	for {
		delay := r.timeUntilRefresh()
		r.log.DebugContext(ctx, "Scheduled the next GeoLite2 City database refresh", slog.Duration("in", delay))

		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}

		err := r.refresh(ctx)
		if err != nil {
			// The next attempt is scheduled by timeUntilRefresh, which still sees a missing or stale file and comes back after refreshRetryInterval
			r.log.ErrorContext(ctx, "Failed to refresh the GeoLite2 City database, will try again later", slog.Any("error", err))

			timer = time.NewTimer(refreshRetryInterval)
			select {
			case <-ctx.Done():
				timer.Stop()
				return
			case <-timer.C:
			}
		}
	}
}

// timeUntilRefresh reports how long to wait before the database on disk needs downloading again
//
// The age of the file is the only input, which is what makes this work across replicas without any coordination: on a shared mount, whichever replica gets there first refreshes the file and the others find it fresh and go back to sleep, and on separate disks each replica refreshes its own copy.
func (r *refresher) timeUntilRefresh() time.Duration {
	info, err := os.Stat(r.service.dbPath)
	if err != nil {
		// Treat a database that is missing, or that can't be read, as due for a download
		return 0
	}

	remaining := databaseMaxAge - time.Since(info.ModTime())
	if remaining <= 0 {
		return 0
	}

	return remaining + jitter()
}

// jitter returns a random offset within refreshJitter, so replicas sharing a mount don't wake up in lockstep
func jitter() time.Duration {
	// #nosec G404 -- not used for anything security related
	return time.Duration(rand.Int64N(int64(2*refreshJitter))) - refreshJitter
}

// refresh downloads the database and loads it
func (r *refresher) refresh(parentCtx context.Context) error {
	r.log.InfoContext(parentCtx, "Refreshing the GeoLite2 City database")

	ctx, cancel := context.WithTimeout(parentCtx, downloadTimeout)
	defer cancel()

	err := downloadDatabase(ctx, r.httpClient, r.downloadURL, r.licenseKey, r.service.dbPath)
	if err != nil {
		return err
	}

	r.log.InfoContext(parentCtx, "GeoLite2 City database successfully refreshed")

	// The watcher would pick the new file up too, but loading it here makes the refresh complete on its own
	return r.service.load(parentCtx)
}

// watch reloads the database whenever the file changes, so a database replaced by hand takes effect without a restart
func (r *refresher) watch(ctx context.Context) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		r.log.ErrorContext(ctx, "Failed to create the GeoLite2 City database watcher, changes to the file will need a restart", slog.Any("error", err))
		return
	}
	defer watcher.Close()

	// The directory is watched rather than the file itself: a database is put in place by renaming over it, which leaves a watch on the old file pointing at an inode nothing writes to again
	baseDir := filepath.Dir(r.service.dbPath)
	err = os.MkdirAll(baseDir, 0700)
	if err != nil {
		r.log.ErrorContext(ctx, "Failed to create the GeoLite2 City database directory, changes to the file will need a restart", slog.Any("error", err))
		return
	}
	err = watcher.Add(baseDir)
	if err != nil {
		r.log.ErrorContext(ctx, "Failed to watch the GeoLite2 City database directory, changes to the file will need a restart", slog.Any("error", err))
		return
	}

	// Load once the watch is in place, to pick up a database that showed up while the watcher was being set up
	// Without this, a file put there in that window would go unnoticed until the next refresh, and there is no next refresh when refreshes are disabled
	err = r.service.load(ctx)
	if err != nil {
		r.log.ErrorContext(ctx, "Failed to load the GeoLite2 City database", slog.Any("error", err))
	}

	if r.watching != nil {
		close(r.watching)
	}

	// The timer debounces the burst of events a write produces, so the database is loaded once the file has settled
	reload := time.NewTimer(watcherDebounce)
	if !reload.Stop() {
		<-reload.C
	}
	defer reload.Stop()

	for {
		select {
		case <-ctx.Done():
			return

		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			if filepath.Base(event.Name) != filepath.Base(r.service.dbPath) {
				continue
			}
			if !event.Has(fsnotify.Create | fsnotify.Write | fsnotify.Rename | fsnotify.Remove) {
				continue
			}

			r.log.DebugContext(ctx, "GeoLite2 City database change detected", slog.String("path", event.Name))
			reload.Stop()
			select {
			case <-reload.C:
			default:
			}
			reload.Reset(watcherDebounce)

		case <-reload.C:
			err := r.service.load(ctx)
			if err != nil {
				r.log.ErrorContext(ctx, "Failed to load the GeoLite2 City database after it changed", slog.Any("error", err))
			}

		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			if !errors.Is(err, context.Canceled) {
				r.log.ErrorContext(ctx, "GeoLite2 City database watcher error", slog.Any("error", err))
			}
		}
	}
}
