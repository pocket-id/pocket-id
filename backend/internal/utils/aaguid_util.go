package utils

import (
	"encoding/hex"

	"github.com/google/uuid"
)

//go:generate go run ../../tools/gen-aaguid/main.go ../../resources/aaguids.json aaguid_map_gen.go

// FormatAAGUID converts an AAGUID byte slice to UUID string format
func FormatAAGUID(aaguid []byte) string {
	if len(aaguid) == 0 {
		return ""
	}

	// If exactly 16 bytes, format as UUID
	if len(aaguid) == 16 {
		u, err := uuid.FromBytes(aaguid)
		if err == nil {
			return u.String()
		}
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

	// Check the generated static map
	if name, ok := AAGUIDMap[aaguidStr]; ok {
		return name + " Passkey"
	}

	return ""
}
