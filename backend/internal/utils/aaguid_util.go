package utils

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"sync"

	"github.com/pocket-id/pocket-id/backend/resources"
)

type AAGUIDEntry struct {
	Name string `json:"name"`
}

var (
	aaguidMap     map[string]string
	aaguidMapOnce sync.Once
	aaguidMapMu   sync.RWMutex
)

// FormatAAGUID converts an AAGUID byte slice to UUID string format
func FormatAAGUID(aaguid []byte) string {

	if len(aaguid) == 0 {
		return ""
	}

	// If exactly 16 bytes, format as UUID
	if len(aaguid) == 16 {
		return fmt.Sprintf("%x-%x-%x-%x-%x",
			aaguid[0:4], aaguid[4:6], aaguid[6:8], aaguid[8:10], aaguid[10:16])
	}

	// Otherwise just return as hex
	return hex.EncodeToString(aaguid)

}

// GetAuthenticatorName returns the name of the authenticator for the given AAGUID
func GetAuthenticatorName(aaguid []byte) string {
	aaguidStr := FormatAAGUID(aaguid)
	if aaguidStr == "" {
		return ""
	}

	// Then check JSON-sourced map
	aaguidMapOnce.Do(loadAAGUIDsFromFile)

	aaguidMapMu.RLock()
	defer aaguidMapMu.RUnlock()

	if name, ok := aaguidMap[aaguidStr]; ok {
		return name + " Passkey"
	}

	return ""
}

// loadAAGUIDsFromFile loads AAGUID data from the embedded file system
func loadAAGUIDsFromFile() {
	aaguidMapMu.Lock()
	defer aaguidMapMu.Unlock()

	aaguidMap = make(map[string]string)

	// Read from embedded file system
	data, err := resources.FS.ReadFile("aaguids.json")
	if err != nil {
		log.Printf("Error reading embedded AAGUID file: %v", err)
		return
	}

	var entries map[string]AAGUIDEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		log.Printf("Error unmarshalling AAGUID data: %v", err)
		return
	}

	// Populate the AAGUID map
	for guid, entry := range entries {
		if guid != "" && entry.Name != "" {
			aaguidMap[guid] = entry.Name
		}
	}
}
