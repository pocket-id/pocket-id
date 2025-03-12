package utils

import (
	"crypto/sha256"
	"encoding/hex"
)

func CreateSha256Hash(input string) string {
	hash := sha256.Sum256([]byte(input))
	return hex.EncodeToString(hash[:])
}
