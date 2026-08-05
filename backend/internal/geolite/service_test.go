package geolite

import (
	"os"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestServiceGetLocationByIPPrivateRanges(t *testing.T) {
	// Private addresses are short-circuited, so the service resolves them even with no database around
	svc, _ := newServiceForTest(t, nil)

	tests := []struct {
		name      string
		ipAddress string
		country   string
		city      string
	}{
		{name: "empty address", ipAddress: ""},
		{name: "private LAN IPv4", ipAddress: "192.168.1.20", country: internalNetworkCountry, city: "LAN"},
		{name: "private LAN IPv4 in the 10/8 range", ipAddress: "10.4.5.6", country: internalNetworkCountry, city: "LAN"},
		{name: "Tailscale IPv4", ipAddress: "100.101.102.103", country: internalNetworkCountry, city: "Tailscale"},
		{name: "IPv6 unique local address", ipAddress: "fd00::1", country: internalNetworkCountry, city: "LAN"},
		{name: "IPv4 loopback", ipAddress: "127.0.0.1", country: internalNetworkCountry, city: "LAN"},
		{name: "IPv6 loopback", ipAddress: "::1", country: internalNetworkCountry, city: "LAN"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			country, city, err := svc.GetLocationByIP(t.Context(), tt.ipAddress)
			require.NoError(t, err)
			require.Equal(t, tt.country, country)
			require.Equal(t, tt.city, city)
		})
	}
}

func TestServiceGetLocationByIPInvalidAddress(t *testing.T) {
	svc, _ := newServiceForTest(t, nil)

	_, _, err := svc.GetLocationByIP(t.Context(), "not-an-ip")
	require.Error(t, err)
	require.ErrorContains(t, err, "failed to parse IP address")
}

func TestServiceGetLocationByIP(t *testing.T) {
	svc, _ := newServiceForTest(t, readTestDatabase(t))

	tests := []struct {
		name      string
		ipAddress string
		country   string
		city      string
	}{
		{name: "public IPv4 with country and city", ipAddress: "81.2.69.142", country: "United Kingdom", city: "London"},
		{name: "public IPv4 in another country", ipAddress: "216.160.83.56", country: "United States", city: "Milton"},
		{name: "public IPv4 with country only", ipAddress: "67.43.156.1", country: "Bhutan"},
		{name: "public IPv6", ipAddress: "2001:218::1", country: "Japan"},
		{name: "public address not in the database", ipAddress: "8.8.8.8"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			country, city, err := svc.GetLocationByIP(t.Context(), tt.ipAddress)
			require.NoError(t, err)
			require.Equal(t, tt.country, country)
			require.Equal(t, tt.city, city)
		})
	}
}

func TestServiceGetLocationByIPWithoutDatabase(t *testing.T) {
	// Air-gapped deployments that haven't supplied a database yet get no location, rather than an error on every audit log entry
	svc, _ := newServiceForTest(t, nil)

	country, city, err := svc.GetLocationByIP(t.Context(), "81.2.69.142")
	require.NoError(t, err)
	require.Empty(t, country)
	require.Empty(t, city)
}

func TestServiceLoadMissingDatabase(t *testing.T) {
	svc, dbPath := newServiceForTest(t, readTestDatabase(t))

	country, _, err := svc.GetLocationByIP(t.Context(), "81.2.69.142")
	require.NoError(t, err)
	require.Equal(t, "United Kingdom", country)

	// A database that goes away stops being used, rather than being served from a mapping of a file that no longer exists
	require.NoError(t, os.Remove(dbPath))
	require.NoError(t, svc.load(t.Context()))

	country, _, err = svc.GetLocationByIP(t.Context(), "81.2.69.142")
	require.NoError(t, err)
	require.Empty(t, country)
}

func TestServiceLoadInvalidDatabase(t *testing.T) {
	svc, dbPath := newServiceForTest(t, readTestDatabase(t))

	// A corrupted file fails to load, and the database already mapped keeps serving lookups
	writeDatabaseFileForTest(t, dbPath, []byte("not a database"))

	err := svc.load(t.Context())
	require.Error(t, err)
	require.ErrorContains(t, err, "failed to open the GeoLite2 City database")

	country, _, err := svc.GetLocationByIP(t.Context(), "81.2.69.142")
	require.NoError(t, err)
	require.Equal(t, "United Kingdom", country)
}

func TestServiceLoadUnchangedDatabase(t *testing.T) {
	// Reloading a file that hasn't changed keeps the current mapping, so an unrelated event in the watched directory costs nothing
	svc, _ := newServiceForTest(t, readTestDatabase(t))

	svc.mu.RLock()
	before := svc.db
	svc.mu.RUnlock()

	require.NoError(t, svc.load(t.Context()))

	svc.mu.RLock()
	after := svc.db
	svc.mu.RUnlock()

	require.Same(t, before, after)
}

func TestServiceConcurrentLookupsDuringReload(t *testing.T) {
	// Lookups read straight out of the mapped file, so a reload must not unmap a database that a lookup is still reading
	database := readTestDatabase(t)
	svc, dbPath := newServiceForTest(t, database)

	const (
		lookers = 16
		reloads = 25
	)

	stop := make(chan struct{})
	errs := make([]error, lookers)

	var wg sync.WaitGroup
	wg.Add(lookers)
	for i := range lookers {
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
				}

				_, _, err := svc.GetLocationByIP(t.Context(), "81.2.69.142")
				if err != nil {
					errs[i] = err
					return
				}
			}
		}()
	}

	for i := range reloads {
		writeDatabaseFileForTest(t, dbPath, database)

		// Force a distinct modification time, so every load really does remap the file instead of finding it unchanged
		modTime := time.Now().Add(-time.Duration(i) * time.Second)
		require.NoError(t, os.Chtimes(dbPath, modTime, modTime))

		require.NoError(t, svc.load(t.Context()))
	}

	close(stop)
	wg.Wait()

	for _, err := range errs {
		require.NoError(t, err)
	}
}
