package dto

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateUsername(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"valid simple", "user123", true},
		{"valid with dot", "user.name", true},
		{"valid with underscore", "user_name", true},
		{"valid with hyphen", "user-name", true},
		{"valid with at", "user@name", true},
		{"starts with symbol", ".username", false},
		{"ends with non-alphanumeric", "username-", false},
		{"contains space", "user name", false},
		{"empty", "", false},
		{"only special chars", "-._@", false},
		{"valid long", "a1234567890_b.c-d@e", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, ValidateUsername(tt.input))
		})
	}
}

func TestValidateClientID(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"valid simple", "client123", true},
		{"valid with dot", "client.id", true},
		{"valid with underscore", "client_id", true},
		{"valid with hyphen", "client-id", true},
		{"valid with all", "client.id-123_abc", true},
		{"contains space", "client id", false},
		{"contains at", "client@id", false},
		{"empty", "", false},
		{"only special chars", "-._", true},
		{"invalid char", "client!id", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, ValidateClientID(tt.input))
		})
	}
}
