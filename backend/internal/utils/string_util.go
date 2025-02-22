package utils

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"net/url"
	"regexp"
	"strings"
	"unicode"
)

// GenerateRandomAlphanumericString generates a random alphanumeric string of the given length
func GenerateRandomAlphanumericString(length int) (string, error) {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	const charsetLength = int64(len(charset))

	if length <= 0 {
		return "", fmt.Errorf("length must be a positive integer")
	}

	result := make([]byte, length)

	for i := range result {
		num, err := rand.Int(rand.Reader, big.NewInt(charsetLength))
		if err != nil {
			return "", err
		}
		result[i] = charset[num.Int64()]
	}

	return string(result), nil
}

func GetHostnameFromURL(rawURL string) string {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	return parsedURL.Hostname()
}

// Returns an absolute path when provided with a URL (https://test.com) and a path "/.well-known/openid-configuration"
func GetAbsoluteURL(baseURL string, path string) string {
	url, err := url.JoinPath(baseURL, path)
	if err != nil {
		return ""
	}
	return url
}

// StringPointer creates a string pointer from a string value
func StringPointer(s string) *string {
	return &s
}

func CapitalizeFirstLetter(s string) string {
	if s == "" {
		return s
	}
	runes := []rune(s)
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}

func CamelCaseToSnakeCase(s string) string {
	var result []rune
	for i, r := range s {
		if unicode.IsUpper(r) && i > 0 {
			result = append(result, '_')
		}
		result = append(result, unicode.ToLower(r))
	}
	return string(result)
}

func CamelCaseToScreamingSnakeCase(s string) string {
	// Insert underscores before uppercase letters (except the first one)
	re := regexp.MustCompile(`([a-z0-9])([A-Z])`)
	snake := re.ReplaceAllString(s, `${1}_${2}`)

	// Convert to uppercase
	return strings.ToUpper(snake)
}
