package utils

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"sync"

	"github.com/pocket-id/pocket-id/backend/resources"
)

var (
	aaguidMap     map[string]string
	aaguidMapOnce *sync.Once
)

func init() {
	aaguidMapOnce = &sync.Once{}
}

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

	if name, ok := aaguidMap[aaguidStr]; ok {
		return name + " Passkey"
	}

	return ""
}

// loadAAGUIDsFromFile loads AAGUID data from the embedded file system
func loadAAGUIDsFromFile() {
	// Read from embedded file system
	data, err := resources.FS.ReadFile("aaguids.json")
	if err != nil {
		log.Printf("Error reading embedded AAGUID file: %v", err)
		return
	}

	if err := json.Unmarshal(data, &aaguidMap); err != nil {
		log.Printf("Error unmarshalling AAGUID data: %v", err)
		return
	}
}
