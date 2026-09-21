package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUserInitials(t *testing.T) {
	tests := []struct {
		name     string
		user     User
		expected string
	}{
		{
			name:     "first and last name",
			user:     User{Username: "username", FirstName: "John", LastName: "Doe"},
			expected: "JD",
		},
		{
			name:     "first name only",
			user:     User{Username: "username", FirstName: "John"},
			expected: "J",
		},
		{
			name:     "last name only",
			user:     User{Username: "username", LastName: "Doe"},
			expected: "D",
		},
		{
			name:     "ASCII username",
			user:     User{Username: "username"},
			expected: "U",
		},
		{
			name:     "single-character username",
			user:     User{Username: "a"},
			expected: "A",
		},
		{
			name:     "empty username",
			user:     User{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.user.Initials())
		})
	}
}
