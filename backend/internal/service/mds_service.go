package service

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
)

const (
	mdsURL = "https://mds3.fidoalliance.org/"
)

var mdsCacheDir string

// MdsService handles interactions with the FIDO Metadata Service
type MdsService struct {
	aaguidMap  map[string]string
	lastUpdate time.Time
	mutex      sync.RWMutex
}

// MdsEntry represents a single entry in the FIDO Metadata Service
type MdsEntry struct {
	AaGUID            string `json:"aaguid"`
	MetadataStatement struct {
		Description string `json:"description"`
		Icon        string `json:"icon"`
	} `json:"metadataStatement"`
}

// MdsJWTPayload represents the payload of the JWT from the FIDO Metadata Service
type MdsJWTPayload struct {
	LegalHeader string     `json:"legalHeader"`
	Entries     []MdsEntry `json:"entries"`
}

// NewMdsService creates a new MDS service instance
func NewMdsService() *MdsService {
	mdsCacheDir = common.EnvConfig.MdsCachePath

	service := &MdsService{
		aaguidMap: make(map[string]string),
	}

	// Ensure cache directory exists
	// Ensure cache directory exists
	if err := ensureCacheDirExists(); err != nil {
		log.Printf("Error creating MDS cache directory: %v", err)
	}

	// Try to load cached data first
	if err := service.loadCachedData(); err != nil {
		log.Printf("Could not load cached MDS data: %v, will fetch fresh data", err)
	}

	return service
}

// GetAuthenticatorName returns the name of the authenticator for the given AAGUID
func (s *MdsService) GetAuthenticatorName(aaguid []byte) string {
	aaguidStr := utils.FormatAAGUID(aaguid)
	if aaguidStr == "" {
		return ""
	}

	// First check built-in map (higher priority)
	if name, ok := utils.AAGUIDMap[aaguidStr]; ok {
		return name + " Passkey"
	}

	// Then check MDS-sourced map
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	if name, ok := s.aaguidMap[aaguidStr]; ok {
		return name + " Passkey"
	}

	return ""
}

// loadCachedData attempts to load AAGUID data from the cache file
func (s *MdsService) loadCachedData() error {
	// Check if cache file exists
	_, err := os.Stat(filepath.Join(mdsCacheDir, "aaguid_cache.json"))
	if os.IsNotExist(err) {
		return fmt.Errorf("cache file does not exist")
	}

	// Read the cache file
	data, err := os.ReadFile(filepath.Join(mdsCacheDir, "aaguid_cache.json"))
	if err != nil {
		return fmt.Errorf("error reading cache file: %w", err)
	}

	// Parse the cache file
	var cacheData struct {
		LastUpdate time.Time         `json:"lastUpdate"`
		AAGUIDMap  map[string]string `json:"aaguidMap"`
	}

	if err := json.Unmarshal(data, &cacheData); err != nil {
		return fmt.Errorf("error unmarshalling cache data: %w", err)
	}

	// Update the service with cached data
	s.mutex.Lock()
	s.aaguidMap = cacheData.AAGUIDMap
	s.lastUpdate = cacheData.LastUpdate
	s.mutex.Unlock()

	log.Printf("Loaded %d AAGUIDs from cache (last updated: %v)", len(cacheData.AAGUIDMap), cacheData.LastUpdate)
	return nil
}

// saveCachedData saves the current AAGUID map to the cache file
func (s *MdsService) saveCachedData() error {
	s.mutex.RLock()
	cacheData := struct {
		LastUpdate time.Time         `json:"lastUpdate"`
		AAGUIDMap  map[string]string `json:"aaguidMap"`
	}{
		LastUpdate: s.lastUpdate,
		AAGUIDMap:  s.aaguidMap,
	}
	s.mutex.RUnlock()

	data, err := json.Marshal(cacheData)
	if err != nil {
		return fmt.Errorf("error marshalling cache data: %w", err)
	}

	if err := os.WriteFile(filepath.Join(mdsCacheDir, "aaguid_cache.json"), data, 0644); err != nil {
		return fmt.Errorf("error writing cache file: %w", err)
	}

	// Also save the raw JWT for reference
	if rawJWT, err := os.ReadFile(filepath.Join(mdsCacheDir, "cache.jwt")); err == nil && len(rawJWT) > 0 {
		log.Printf("Saved raw JWT to cache file (%d bytes)", len(rawJWT))
	}

	return nil
}

// UpdateAaguidMap fetches and processes the latest FIDO MDS data
func (s *MdsService) UpdateAaguidMap() error {

	// Fetch the JWT from the MDS
	resp, err := http.Get(mdsURL)
	if err != nil {
		log.Printf("Error fetching FIDO MDS: %v", err)
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Error reading FIDO MDS response: %v", err)
		return err
	}

	// Save the raw JWT
	if err := os.WriteFile(filepath.Join(mdsCacheDir, "cache.jwt"), body, 0644); err != nil {
		log.Printf("Error saving raw JWT: %v", err)
	}

	// JWT has 3 parts separated by dots
	parts := splitJWT(string(body))
	if len(parts) != 3 {
		return fmt.Errorf("invalid JWT format")
	}

	// Decode the payload (part 1)
	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return fmt.Errorf("error decoding JWT payload: %w", err)
	}

	var payload MdsJWTPayload
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		return fmt.Errorf("error unmarshalling JWT payload: %w", err)
	}

	// Process the entries
	newMap := make(map[string]string)
	for _, entry := range payload.Entries {
		if entry.AaGUID != "" && entry.MetadataStatement.Description != "" {
			newMap[entry.AaGUID] = entry.MetadataStatement.Description
		}
	}

	// Update the map atomically
	s.mutex.Lock()
	s.aaguidMap = newMap
	s.lastUpdate = time.Now()
	s.mutex.Unlock()

	// Save to cache file
	if err := s.saveCachedData(); err != nil {
		log.Printf("Error saving cache data: %v", err)
	}
	return nil
}

// Helper function to ensure the cache directory exists
func ensureCacheDirExists() error {
	return os.MkdirAll(mdsCacheDir, 0755)
}

// Helper function to split a JWT string into its parts
func splitJWT(token string) []string {
	var parts []string
	for len(token) > 0 {
		idx := -1
		for i, c := range token {
			if c == '.' {
				idx = i
				break
			}
		}
		if idx < 0 {
			parts = append(parts, token)
			break
		}
		parts = append(parts, token[:idx])
		token = token[idx+1:]
	}
	return parts
}
