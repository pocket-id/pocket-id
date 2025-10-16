package utils

import (
	"strconv"
	"strings"
)

// ConvertStringToType attempts to convert a string to bool, int, or float.
func ConvertStringToType(value string) any {
	v := strings.TrimSpace(value)
	if v == "" {
		return v
	}

	// Try bool
	if v == "true" {
		return true
	}
	if v == "false" {
		return false
	}

	// Try int
	if i, err := strconv.Atoi(v); err == nil {
		return i
	}

	// Try float
	if f, err := strconv.ParseFloat(v, 64); err == nil {
		return f
	}

	// Default: string
	return v
}
