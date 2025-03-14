package utils

import (
	"regexp"
	"strconv"
	"strings"
	"testing"
)

func TestGenerateRandomAlphanumericString(t *testing.T) {
	t.Run("valid length returns correct string", func(t *testing.T) {
		const length = 10
		str, err := GenerateRandomAlphanumericString(length)

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if len(str) != length {
			t.Errorf("Expected length %d, got %d", length, len(str))
		}

		matched, err := regexp.MatchString(`^[a-zA-Z0-9]+$`, str)
		if err != nil {
			t.Errorf("Regex match failed: %v", err)
		}
		if !matched {
			t.Errorf("String contains non-alphanumeric characters: %s", str)
		}
	})

	t.Run("zero length returns error", func(t *testing.T) {
		_, err := GenerateRandomAlphanumericString(0)
		if err == nil {
			t.Error("Expected error for zero length, got nil")
		}
	})

	t.Run("negative length returns error", func(t *testing.T) {
		_, err := GenerateRandomAlphanumericString(-1)
		if err == nil {
			t.Error("Expected error for negative length, got nil")
		}
	})

	t.Run("generates different strings", func(t *testing.T) {
		str1, _ := GenerateRandomAlphanumericString(10)
		str2, _ := GenerateRandomAlphanumericString(10)
		if str1 == str2 {
			t.Error("Generated strings should be different")
		}
	})
}

func BenchmarkGenerateRandomAlphanumericString(b *testing.B) {
	lengths := []int{8, 16, 32, 64, 128}

	for _, length := range lengths {
		b.Run(strconv.Itoa(length), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				GenerateRandomAlphanumericString(length)
			}
		})
	}
}

func TestCapitalizeFirstLetter(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"empty string", "", ""},
		{"lowercase first letter", "hello", "Hello"},
		{"already capitalized", "Hello", "Hello"},
		{"single lowercase letter", "h", "H"},
		{"single uppercase letter", "H", "H"},
		{"starts with number", "123abc", "123abc"},
		{"unicode character", "étoile", "Étoile"},
		{"special character", "_test", "_test"},
		{"multi-word", "hello world", "Hello world"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CapitalizeFirstLetter(tt.input)
			if result != tt.expected {
				t.Errorf("CapitalizeFirstLetter(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func BenchmarkCapitalizeFirstLetter(b *testing.B) {
	lengths := []int{8, 16, 32, 64, 128}

	for _, length := range lengths {
		b.Run(strconv.Itoa(length), func(b *testing.B) {
			str := strings.Repeat("a", length)
			for i := 0; i < b.N; i++ {
				CapitalizeFirstLetter(str)
			}
		})
	}
}

func TestCamelCaseToSnakeCase(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"empty string", "", ""},
		{"simple camelCase", "camelCase", "camel_case"},
		{"PascalCase", "PascalCase", "pascal_case"},
		{"multipleWordsInCamelCase", "multipleWordsInCamelCase", "multiple_words_in_camel_case"},
		{"consecutive uppercase", "HTTPRequest", "h_t_t_p_request"},
		{"single lowercase word", "word", "word"},
		{"single uppercase word", "WORD", "w_o_r_d"},
		{"with numbers", "camel123Case", "camel123_case"},
		{"with numbers in middle", "model2Name", "model2_name"},
		{"mixed case", "iPhone6sPlus", "i_phone6s_plus"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CamelCaseToSnakeCase(tt.input)
			if result != tt.expected {
				t.Errorf("CamelCaseToSnakeCase(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func BenchmarkCamelCaseToSnakeCase(b *testing.B) {
	testCases := []string{
		"simpleCase",
		"PascalCase",
		"ThisIsALongerStringWithMultipleWords",
		"HTTPRequestHandler",
		"OAuth2ClientID",
	}

	for _, tc := range testCases {
		b.Run(tc, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				CamelCaseToSnakeCase(tc)
			}
		})
	}
}
