package service

import (
	"github.com/google/uuid"
	"testing"
)

func TestGetDNProperty(t *testing.T) {
	tests := []struct {
		name           string
		property       string
		dn             string
		expectedResult string
	}{
		{
			name:           "simple case",
			property:       "cn",
			dn:             "cn=username,ou=people,dc=example,dc=com",
			expectedResult: "username",
		},
		{
			name:           "property not found",
			property:       "uid",
			dn:             "cn=username,ou=people,dc=example,dc=com",
			expectedResult: "",
		},
		{
			name:           "mixed case property",
			property:       "CN",
			dn:             "cn=username,ou=people,dc=example,dc=com",
			expectedResult: "username",
		},
		{
			name:           "mixed case DN",
			property:       "cn",
			dn:             "CN=username,OU=people,DC=example,DC=com",
			expectedResult: "username",
		},
		{
			name:           "spaces in DN",
			property:       "cn",
			dn:             "cn=username, ou=people, dc=example, dc=com",
			expectedResult: "username",
		},
		{
			name:           "value with special characters",
			property:       "cn",
			dn:             "cn=user.name+123,ou=people,dc=example,dc=com",
			expectedResult: "user.name+123",
		},
		{
			name:           "empty DN",
			property:       "cn",
			dn:             "",
			expectedResult: "",
		},
		{
			name:           "empty property",
			property:       "",
			dn:             "cn=username,ou=people,dc=example,dc=com",
			expectedResult: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getDNProperty(tt.property, tt.dn)
			if result != tt.expectedResult {
				t.Errorf("getDNProperty(%q, %q) = %q, want %q",
					tt.property, tt.dn, result, tt.expectedResult)
			}
		})
	}
}

func TestConvertLdapIdToString(t *testing.T) {
	u := uuid.New()
	hexStr := u.String()
	hexStr = hexStr[:8] + hexStr[9:13] + hexStr[14:18] + hexStr[19:23] + hexStr[24:]

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "valid UTF-8 string",
			input:    "simple-utf8-id",
			expected: "simple-utf8-id",
		},
		{
			name:     "binary UUID (16 bytes)",
			input:    string(u[:]),
			expected: u.String(),
		},
		{
			name:     "non-UTF8, non-UUID returns base64",
			input:    string([]byte{0xff, 0xfe, 0xfd, 0xfc}),
			expected: "//79/A==",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := convertLdapIdToString(tt.input)
			if got != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, got)
			}
		})
	}
}
