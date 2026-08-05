package geolite

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/netip"
	"os"
	"sync"

	"github.com/oschwald/maxminddb-golang/v2"

	"github.com/pocket-id/pocket-id/backend/internal/utils"
)

// The GeoLite2 City database is kept on disk and memory-mapped (the format is optimized for random access)
//
// The database file is considered cache, not state: it is a copy of a public artifact that any replica can rebuild on its own, so nothing is lost when a node goes away, and every replica keeps its own without needing to replicate anything
// It is also the supported way to supply a database by hand, which is what air-gapped deployments do: the file is watched, so replacing it takes effect without a restart

// internalNetworkCountry is reported for addresses that aren't routable on the public Internet
const internalNetworkCountry = "Internal Network"

// Service resolves IP addresses to locations, against a memory-mapped GeoLite2 City database
type Service struct {
	log    *slog.Logger
	dbPath string

	// mu guards the fields below
	// A lookup holds it for reading throughout, so a reload can't unmap the database from under it
	mu sync.RWMutex
	// db is the database currently mapped, or nil when there is no readable database at dbPath
	db *maxminddb.Reader
	// dbModTime and dbSize identify the file that was mapped, so a reload of an unchanged file is skipped
	dbModTime int64
	dbSize    int64
}

func newService(log *slog.Logger, dbPath string) *Service {
	return &Service{
		log:    log,
		dbPath: dbPath,
	}
}

// GetLocationByIP returns the country and city of the given IP address
// Both are empty when the address isn't in the database, or when no database is available
func (s *Service) GetLocationByIP(_ context.Context, ipAddress string) (country string, city string, err error) {
	if ipAddress == "" {
		return "", "", nil
	}

	// Check the IP address against known private IP ranges, which can be short-circuited
	ip := net.ParseIP(ipAddress)
	if ip != nil {
		switch {
		case utils.IsLocalIPv6(ip):
			return internalNetworkCountry, "LAN", nil
		case utils.IsTailscaleIP(ip):
			return internalNetworkCountry, "Tailscale", nil
		case utils.IsPrivateIP(ip):
			return internalNetworkCountry, "LAN", nil
		case utils.IsLocalhostIP(ip):
			return internalNetworkCountry, "localhost", nil
		}
	}

	addr, err := netip.ParseAddr(ipAddress)
	if err != nil {
		return "", "", fmt.Errorf("failed to parse IP address: %w", err)
	}

	// The read lock is held for the whole lookup, including decoding, because the record is decoded straight out of the mapped file
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.db == nil {
		// No database is available
		return "", "", nil
	}

	result := s.db.Lookup(addr)
	if !result.Found() {
		return "", "", nil
	}

	var record geoLiteRecord
	err = result.Decode(&record)
	if err != nil {
		return "", "", fmt.Errorf("failed to decode database record: %w", err)
	}

	return record.Country.Names["en"], record.City.Names["en"], nil
}

// geoLiteRecord is the subset of a GeoLite2 City record that Pocket ID uses
type geoLiteRecord struct {
	City struct {
		Names map[string]string `maxminddb:"names"`
	} `maxminddb:"city"`
	Country struct {
		Names map[string]string `maxminddb:"names"`
	} `maxminddb:"country"`
}

// load maps the database at dbPath, replacing the one currently mapped
// A file that is already mapped is left alone, so reloading after an unrelated change is free
func (s *Service) load(ctx context.Context) error {
	info, err := os.Stat(s.dbPath)
	if err != nil {
		if os.IsNotExist(err) {
			// There's no database to map yet: lookups return no location until one shows up
			s.unload()
			return nil
		}

		return fmt.Errorf("failed to stat the GeoLite2 City database: %w", err)
	}

	modTime, size := info.ModTime().UnixNano(), info.Size()

	s.mu.RLock()
	unchanged := s.db != nil && s.dbModTime == modTime && s.dbSize == size
	s.mu.RUnlock()

	if unchanged {
		return nil
	}

	db, err := maxminddb.Open(s.dbPath)
	if err != nil {
		return fmt.Errorf("failed to open the GeoLite2 City database: %w", err)
	}

	s.mu.Lock()
	old := s.db
	s.db = db
	s.dbModTime = modTime
	s.dbSize = size
	s.mu.Unlock()

	// The previous database is unmapped only once no lookup can still be reading it, which the write lock above guarantees
	closeDatabase(old)

	s.log.InfoContext(ctx, "Loaded the GeoLite2 City database",
		slog.String("path", s.dbPath),
		slog.Time("modTime", info.ModTime()),
		slog.Int64("size", size),
	)

	return nil
}

// unload drops the database currently mapped, so lookups stop returning locations from a file that is no longer there
func (s *Service) unload() {
	s.mu.Lock()
	old := s.db
	s.db = nil
	s.dbModTime = 0
	s.dbSize = 0
	s.mu.Unlock()

	closeDatabase(old)
}

// closeDatabase unmaps a database
// The caller must have already made it unreachable to lookups, since unmapping a database that is still being read would crash the process
func closeDatabase(db *maxminddb.Reader) {
	if db == nil {
		return
	}

	// Unmapping only fails if the mapping is already gone, which is not something the caller can act on
	_ = db.Close()
}
