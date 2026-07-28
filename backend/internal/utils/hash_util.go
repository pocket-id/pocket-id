package utils

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
)

// CreateSha256Hash creates the SHA256 hash of a string, returning a hex-encoded string.
func CreateSha256Hash(input string) string {
	hash := sha256.Sum256([]byte(input))
	return hex.EncodeToString(hash[:])
}

// ConstantTimeStringEqual compares two strings in constant time to prevent timing attacks.
func ConstantTimeStringEqual(left, right string) bool {
	return subtle.ConstantTimeCompare([]byte(left), []byte(right)) == 1
}
