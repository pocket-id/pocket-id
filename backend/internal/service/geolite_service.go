package service

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/netip"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/oschwald/maxminddb-golang/v2"

	"github.com/pocket-id/pocket-id/backend/internal/common"
)

type GeoLiteService struct {
	disableUpdater bool
	mutex          sync.Mutex
}

var localhostIPNets = []*net.IPNet{
	{IP: net.IPv4(127, 0, 0, 0), Mask: net.CIDRMask(8, 32)}, // 127.0.0.0/8
	{IP: net.IPv6loopback, Mask: net.CIDRMask(128, 128)},    // ::1/128
}

var privateLanIPNets = []*net.IPNet{
	{IP: net.IPv4(10, 0, 0, 0), Mask: net.CIDRMask(8, 32)},     // 10.0.0.0/8
	{IP: net.IPv4(172, 16, 0, 0), Mask: net.CIDRMask(12, 32)},  // 172.16.0.0/12
	{IP: net.IPv4(192, 168, 0, 0), Mask: net.CIDRMask(16, 32)}, // 192.168.0.0/16
}

var tailscaleIPNets = []*net.IPNet{
	{IP: net.IPv4(100, 64, 0, 0), Mask: net.CIDRMask(10, 32)}, // 100.64.0.0/10
}

// NewGeoLiteService initializes a new GeoLiteService instance and starts a goroutine to update the GeoLite2 City database.
func NewGeoLiteService() *GeoLiteService {
	service := &GeoLiteService{}

	if common.EnvConfig.MaxMindLicenseKey == "" && common.EnvConfig.GeoLiteDBUrl == common.MaxMindGeoLiteCityUrl {
		// Warn the user, and disable the periodic updater
		log.Println("MAXMIND_LICENSE_KEY environment variable is empty. The GeoLite2 City database won't be updated.")
		service.disableUpdater = true
	}

	return service
}

func (s *GeoLiteService) HasBackgroundService() bool {
	return !s.disableUpdater
}

func (s *GeoLiteService) Run(ctx context.Context) error {
	if s.disableUpdater {
		// If the updater is disabled, this runner is a no-op, which is blocked until the context is canceled
		<-ctx.Done()
		return ctx.Err()
	}

	// Update the GeoIP database immediately (after a 1s delay) and every 2 days
	ticker := time.NewTicker(48 * time.Hour)
	defer ticker.Stop()
	first := time.NewTimer(time.Second)

	// Run on a loop
	for {
		select {
		case <-ctx.Done():
			// Service is shutting down, return
			return ctx.Err()
		case <-first.C:
			// Run on the first run, and log errors only
			err := s.updateDatabase(ctx)
			if err != nil {
				log.Printf("Failed to update GeoLite2 City database: %v", err)
			}
		case <-ticker.C:
			// Run periodically, logging errors only
			err := s.updateDatabase(ctx)
			if err != nil {
				log.Printf("Failed to update GeoLite2 City database: %v", err)
			}
		}
	}
}

// GetLocationByIP returns the country and city of the given IP address.
func (s *GeoLiteService) GetLocationByIP(ipAddress string) (country, city string, err error) {
	// Check the IP address against known private IP ranges
	if ip := net.ParseIP(ipAddress); ip != nil {
		for _, ipNet := range tailscaleIPNets {
			if ipNet.Contains(ip) {
				return "Internal Network", "Tailscale", nil
			}
		}
		for _, ipNet := range privateLanIPNets {
			if ipNet.Contains(ip) {
				return "Internal Network", "LAN/Docker/k8s", nil
			}
		}
		for _, ipNet := range localhostIPNets {
			if ipNet.Contains(ip) {
				return "Internal Network", "localhost", nil
			}
		}
	}

	// Race condition between reading and writing the database.
	s.mutex.Lock()
	defer s.mutex.Unlock()

	db, err := maxminddb.Open(common.EnvConfig.GeoLiteDBPath)
	if err != nil {
		return "", "", err
	}
	defer db.Close()

	addr := netip.MustParseAddr(ipAddress)

	var record struct {
		City struct {
			Names map[string]string `maxminddb:"names"`
		} `maxminddb:"city"`
		Country struct {
			Names map[string]string `maxminddb:"names"`
		} `maxminddb:"country"`
	}

	err = db.Lookup(addr).Decode(&record)
	if err != nil {
		return "", "", err
	}

	return record.Country.Names["en"], record.City.Names["en"], nil
}

// UpdateDatabase checks the age of the database and updates it if it's older than 14 days.
func (s *GeoLiteService) updateDatabase(parentCtx context.Context) error {
	if s.isDatabaseUpToDate() {
		log.Println("GeoLite2 City database is up-to-date.")
		return nil
	}

	log.Println("Updating GeoLite2 City database...")
	downloadUrl := fmt.Sprintf(common.EnvConfig.GeoLiteDBUrl, common.EnvConfig.MaxMindLicenseKey)

	ctx, cancel := context.WithTimeout(parentCtx, 10*time.Minute)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadUrl, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download database: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download database, received HTTP %d", resp.StatusCode)
	}

	// Extract the database file directly to the target path
	if err := s.extractDatabase(resp.Body); err != nil {
		return fmt.Errorf("failed to extract database: %w", err)
	}

	log.Println("GeoLite2 City database successfully updated.")
	return nil
}

// isDatabaseUpToDate checks if the database file is older than 14 days.
func (s *GeoLiteService) isDatabaseUpToDate() bool {
	info, err := os.Stat(common.EnvConfig.GeoLiteDBPath)
	if err != nil {
		// If the file doesn't exist, treat it as not up-to-date
		return false
	}
	return time.Since(info.ModTime()) < 14*24*time.Hour
}

// extractDatabase extracts the database file from the tar.gz archive directly to the target location.
func (s *GeoLiteService) extractDatabase(reader io.Reader) error {
	gzr, err := gzip.NewReader(reader)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzr.Close()

	tarReader := tar.NewReader(gzr)

	var totalSize int64
	const maxTotalSize = 300 * 1024 * 1024 // 300 MB limit for total decompressed size

	// Iterate over the files in the tar archive
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read tar archive: %w", err)
		}

		// Check if the file is the GeoLite2-City.mmdb file
		if header.Typeflag == tar.TypeReg && filepath.Base(header.Name) == "GeoLite2-City.mmdb" {
			totalSize += header.Size
			if totalSize > maxTotalSize {
				return errors.New("total decompressed size exceeds maximum allowed limit")
			}

			// extract to a temporary file to avoid having a corrupted db in case of write failure.
			baseDir := filepath.Dir(common.EnvConfig.GeoLiteDBPath)
			tmpFile, err := os.CreateTemp(baseDir, "geolite.*.mmdb.tmp")
			if err != nil {
				return fmt.Errorf("failed to create temporary database file: %w", err)
			}
			tempName := tmpFile.Name()

			// Write the file contents directly to the target location
			if _, err := io.Copy(tmpFile, tarReader); err != nil { //nolint:gosec
				// if fails to write, then cleanup and throw an error
				tmpFile.Close()
				os.Remove(tempName)
				return fmt.Errorf("failed to write database file: %w", err)
			}
			tmpFile.Close()

			// ensure the database is not corrupted
			db, err := maxminddb.Open(tempName)
			if err != nil {
				// if fails to write, then cleanup and throw an error
				os.Remove(tempName)
				return fmt.Errorf("failed to open downloaded database file: %w", err)
			}
			db.Close()

			// ensure we lock the structure before we overwrite the database
			// to prevent race conditions between reading and writing the mmdb.
			s.mutex.Lock()
			// replace the old file with the new file
			err = os.Rename(tempName, common.EnvConfig.GeoLiteDBPath)
			s.mutex.Unlock()

			if err != nil {
				// if cannot overwrite via rename, then cleanup and throw an error
				os.Remove(tempName)
				return fmt.Errorf("failed to replace database file: %w", err)
			}
			return nil
		}
	}

	return errors.New("GeoLite2-City.mmdb not found in archive")
}
