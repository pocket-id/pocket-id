package utils

import (
	"regexp"
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

func TestGenerateRandomUnambiguousString(t *testing.T) {
	t.Run("valid length returns correct string", func(t *testing.T) {
		const length = 10
		str, err := GenerateRandomUnambiguousString(length)

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if len(str) != length {
			t.Errorf("Expected length %d, got %d", length, len(str))
		}

		matched, err := regexp.MatchString(`^[abcdefghjkmnpqrstuvwxyzABCDEFGHJKMNPQRSTUVWXYZ23456789]+$`, str)
		if err != nil {
			t.Errorf("Regex match failed: %v", err)
		}
		if !matched {
			t.Errorf("String contains ambiguous characters: %s", str)
		}
	})

	t.Run("zero length returns error", func(t *testing.T) {
		_, err := GenerateRandomUnambiguousString(0)
		if err == nil {
			t.Error("Expected error for zero length, got nil")
		}
	})

	t.Run("negative length returns error", func(t *testing.T) {
		_, err := GenerateRandomUnambiguousString(-1)
		if err == nil {
			t.Error("Expected error for negative length, got nil")
		}
	})
}

func TestGenerateRandomString(t *testing.T) {
	t.Run("valid length returns characters from charset", func(t *testing.T) {
		const length = 20
		const charset = "abc"
		str, err := GenerateRandomString(length, charset)

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if len(str) != length {
			t.Errorf("Expected length %d, got %d", length, len(str))
		}

		for _, r := range str {
			if !strings.ContainsRune(charset, r) {
				t.Fatalf("String contains character outside charset: %q", r)
			}
		}
	})

	t.Run("zero length returns error", func(t *testing.T) {
		_, err := GenerateRandomString(0, "abc")
		if err == nil {
			t.Error("Expected error for zero length, got nil")
		}
	})

	t.Run("negative length returns error", func(t *testing.T) {
		_, err := GenerateRandomString(-1, "abc")
		if err == nil {
			t.Error("Expected error for negative length, got nil")
		}
	})
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
		{"consecutive uppercase", "HTTPRequest", "http_request"},
		{"single lowercase word", "word", "word"},
		{"single uppercase word", "WORD", "word"},
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

func TestCamelCaseToScreamingSnakeCase(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"empty string", "", ""},
		{"simple camelCase", "camelCase", "CAMEL_CASE"},
		{"PascalCase", "PascalCase", "PASCAL_CASE"},
		{"multipleWordsInCamelCase", "multipleWordsInCamelCase", "MULTIPLE_WORDS_IN_CAMEL_CASE"},
		{"consecutive uppercase", "HTTPRequest", "HTTP_REQUEST"},
		{"single lowercase word", "word", "WORD"},
		{"single uppercase word", "WORD", "WORD"},
		{"with numbers", "camel123Case", "CAMEL123_CASE"},
		{"with numbers in middle", "model2Name", "MODEL2_NAME"},
		{"mixed case", "iPhone6sPlus", "I_PHONE6S_PLUS"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CamelCaseToScreamingSnakeCase(tt.input)
			if result != tt.expected {
				t.Errorf("CamelCaseToScreamingSnakeCase(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestGetFirstCharacter(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"empty string", "", ""},
		{"single character", "a", "a"},
		{"multiple characters", "hello", "h"},
		{"unicode character", "étoile", "é"},
		{"special character", "!test", "!"},
		{"number as first character", "123abc", "1"},
		{"whitespace as first character", " hello", "h"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetFirstCharacter(tt.input)
			if result != tt.expected {
				t.Errorf("GetFirstCharacter(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
