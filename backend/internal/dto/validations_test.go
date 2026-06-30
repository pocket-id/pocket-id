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
		{"valid single char", "a", true},
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

func TestValidateResourceURI(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"valid https URI", "https://api.example.com", true},
		{"valid custom scheme URI", "api://PocketID", true},
		{"valid URN", "urn:my-app", true},
		{"valid query component", "https://api.example.com?tenant=1", true},
		{"invalid relative path", "/foo", false},
		{"invalid plain string", "foo", false},
		{"invalid fragment", "https://api.example.com#tokens", false},
		{"invalid empty fragment", "https://api.example.com#", false},
		{"invalid unescaped path space", "https://api.example.com/a b", false},
		{"invalid unescaped opaque space", "urn:my app", false},
		{"invalid malformed URI", "http://[::1", false},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, ValidateResourceURI(tt.input))
		})
	}
}

func TestValidateCallbackURL(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"valid https URL", "https://example.com/callback", true},
		{"valid loopback URL", "http://127.0.0.1:49813/callback", true},
		{"empty scheme", "//127.0.0.1:49813/callback", true},
		{"valid custom scheme", "pocketid://callback", true},
		{"invalid malformed URL", "http://[::1", false},
		{"invalid missing scheme separator", "://example.com/callback", false},
		{"rejects javascript scheme", "javascript:alert(1)", false},
		{"rejects mixed case javascript scheme", "JavaScript:alert(1)", false},
		{"rejects data scheme", "data:text/html;base64,PGgxPkhlbGxvPC9oMT4=", false},
		{"rejects mixed case data scheme", "DaTa:text/html;base64,PGgxPkhlbGxvPC9oMT4=", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, ValidateCallbackURL(tt.input))
		})
	}
}

func TestValidateCallbackURLPattern(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"valid exact URL", "https://example.com/callback", true},
		{"valid wildcard URL", "https://*.example.com/callback", true},
		{"valid custom scheme", "pocketid://callback", true},
		{"valid global wildcard", "*", true},
		{"invalid relative URL", "/callback", false},
		{"invalid malformed URL", "http://[::1", false},
		{"rejects javascript scheme", "javascript:alert(1)", false},
		{"rejects data scheme", "data:text/html;base64,PGgxPkhlbGxvPC9oMT4=", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, ValidateCallbackURLPattern(tt.input))
		})
	}
}
