package profilepicture

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInitials(t *testing.T) {
	tests := []struct {
		name      string
		firstName string
		lastName  string
		username  string
		expected  string
	}{
		{name: "first and last name", firstName: "John", lastName: "Doe", username: "username", expected: "JD"},
		{name: "first name only", firstName: "John", username: "username", expected: "J"},
		{name: "last name only", lastName: "Doe", username: "username", expected: "D"},
		{name: "accented Latin name", firstName: "émile", lastName: "Öz", username: "username", expected: "ÉÖ"},
		{name: "name the font can't render falls back to username", firstName: "민준", lastName: "김", username: "kim", expected: "K"},
		{name: "partially unrenderable name falls back to username", firstName: "John", lastName: "Иванов", username: "ivan", expected: "I"},
		{name: "ASCII username", username: "username", expected: "U"},
		{name: "single-character username", username: "a", expected: "A"},
		{name: "empty username", expected: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, Initials(tt.firstName, tt.lastName, tt.username))
		})
	}
}

func TestCanRenderInitials(t *testing.T) {
	assert.True(t, canRenderInitials("ES"))
	assert.True(t, canRenderInitials("ÉÖ"))
	assert.True(t, canRenderInitials("0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ"))
	assert.False(t, canRenderInitials("한글"))
	assert.False(t, canRenderInitials("EЖ"))
}
