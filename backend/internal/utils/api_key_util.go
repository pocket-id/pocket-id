package utils

import (
	"crypto/sha256"
	"encoding/hex"
)

// HashApiKey creates a SHA-256 hash of the API key
func HashApiKey(apiKey string) string {
	hash := sha256.Sum256([]byte(apiKey))
	return hex.EncodeToString(hash[:])
}
