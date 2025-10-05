package utils

import (
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"net/url"
	"regexp"
	"strings"
	"unicode"
)

// GenerateRandomAlphanumericString generates a random alphanumeric string of the given length
func GenerateRandomAlphanumericString(length int) (string, error) {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

	if length <= 0 {
		return "", errors.New("length must be a positive integer")
	}

	// The algorithm below is adapted from https://stackoverflow.com/a/35615565
	const (
		letterIdxBits = 6                    // 6 bits to represent a letter index
		letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
	)

	result := strings.Builder{}
	result.Grow(length)
	// Because we discard a bunch of bytes, we read more in the buffer to minimize the changes of performing additional IO
	bufferSize := int(float64(length) * 1.3)
	randomBytes := make([]byte, bufferSize)
	for i, j := 0, 0; i < length; j++ {
		// Fill the buffer if needed
		if j%bufferSize == 0 {
			_, err := io.ReadFull(rand.Reader, randomBytes)
			if err != nil {
				return "", fmt.Errorf("failed to generate random bytes: %w", err)
			}
		}

		// Discard bytes that are outside of the range
		// This allows making sure that we maintain uniform distribution
		idx := int(randomBytes[j%length] & letterIdxMask)
		if idx < len(charset) {
			result.WriteByte(charset[idx])
			i++
		}
	}

	return result.String(), nil
}

func GetHostnameFromURL(rawURL string) string {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	return parsedURL.Hostname()
}

// StringPointer creates a string pointer from a string value
func StringPointer(s string) *string {
	return &s
}

func CapitalizeFirstLetter(str string) string {
	if str == "" {
		return ""
	}

	result := strings.Builder{}
	result.Grow(len(str))
	for i, r := range str {
		if i == 0 {
			result.WriteRune(unicode.ToUpper(r))
		} else {
			result.WriteRune(r)
		}
	}
	return result.String()
}

func CamelCaseToSnakeCase(str string) string {
	result := strings.Builder{}
	result.Grow(int(float32(len(str)) * 1.1))
	for i, r := range str {
		if unicode.IsUpper(r) && i > 0 {
			result.WriteByte('_')
		}
		result.WriteRune(unicode.ToLower(r))
	}
	return result.String()
}

var camelCaseToScreamingSnakeCaseRe = regexp.MustCompile(`([a-z0-9])([A-Z])`)

func CamelCaseToScreamingSnakeCase(s string) string {
	// Insert underscores before uppercase letters (except the first one)
	snake := camelCaseToScreamingSnakeCaseRe.ReplaceAllString(s, `${1}_${2}`)

	// Convert to uppercase
	return strings.ToUpper(snake)
}

// GetFirstCharacter returns the first non-whitespace character of the string, correctly handling Unicode
func GetFirstCharacter(str string) string {
	for _, c := range str {
		if unicode.IsSpace(c) {
			continue
		}
		return string(c)
	}

	// Empty string case
	return ""
}

// NormalizeGroupName converts a friendly name to a normalized group name
// by converting to lowercase and replacing non-alphanumeric characters with underscores
func NormalizeGroupName(friendlyName string) string {
	var result strings.Builder
	result.Grow(len(friendlyName))

	for _, r := range friendlyName {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' {
			result.WriteRune(r)
		} else if r >= 'A' && r <= 'Z' {
			result.WriteRune(unicode.ToLower(r))
		} else {
			result.WriteRune('_')
		}
	}

	return result.String()
}
