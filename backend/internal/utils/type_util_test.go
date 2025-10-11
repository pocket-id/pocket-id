package utils

import (
	"testing"
)

func TestConvertStringToType(t *testing.T) {
	tests := []struct {
		input    string
		expected any
	}{
		{"true", true},
		{"false", false},
		{"  true  ", true},
		{"  false  ", false},
		{"42", 42},
		{"  42  ", 42},
		{"3.14", 3.14},
		{"  3.14  ", 3.14},
		{"hello", "hello"},
		{"  hello  ", "hello"},
		{"", ""},
		{"   ", ""},
	}

	for _, tt := range tests {
		result := ConvertStringToType(tt.input)
		if result != tt.expected {
			if f, ok := tt.expected.(float64); ok {
				if rf, ok := result.(float64); ok && rf == f {
					continue
				}
			}
			t.Errorf("ConvertStringToType(%q) = %#v (type %T), want %#v (type %T)", tt.input, result, result, tt.expected, tt.expected)
		}
	}
}
