package utils

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
)

func CreateSha256Hash(input string) string {
	hash := sha256.Sum256([]byte(input))
	return hex.EncodeToString(hash[:])
}

func CreateSha256HashBase64(input string) string {
	hash := sha256.Sum256([]byte(input))
	return base64.RawStdEncoding.EncodeToString(hash[:])
}
