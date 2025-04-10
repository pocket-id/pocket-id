package service

import (
	"testing"
)

func TestGetCN(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "valid CN in the beginning",
			input:    "CN=username,OU=people,DC=example,DC=com",
			expected: "username",
		},
		{
			name:     "lowercase cn",
			input:    "cn=username,ou=people,dc=example,dc=com",
			expected: "username",
		},
		{
			name:     "no CN component",
			input:    "UID=jsmith,OU=people,DC=example,DC=com",
			expected: "",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "CN with special characters",
			input:    "CN=John Doe (Admin),OU=IT Staff,DC=example,DC=com",
			expected: "John Doe (Admin)",
		},
		{
			name:     "multiple CN components",
			input:    "CN=username,CN=backup,OU=people,DC=example,DC=com",
			expected: "username",
		},
		{
			name:     "CN with trailing spaces",
			input:    "CN=username,OU=people,DC=example,DC=com",
			expected: "username",
		},
		{
			name:     "malformed DN without equals",
			input:    "CNusername,OU=people,DC=example,DC=com",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getCN(tt.input)
			if got != tt.expected {
				t.Errorf("getCN(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}
