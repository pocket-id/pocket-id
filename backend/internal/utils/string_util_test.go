package utils

import (
	"regexp"
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

func TestGetCallbackURLFromList(t *testing.T) {
	tests := []struct {
		name         string
		callbackURLs []string
		input        string
		expected     string
	}{
		{
			"classic callback",
			[]string{"https://example.com/callback"},
			"https://example.com/callback",
			"https://example.com/callback",
		},
		{
			"not match",
			[]string{"https://example.org/callback"},
			"https://example.com/callback",
			"",
		},
		{
			"valid localhost",
			[]string{"http://localhost:8080/callback"},
			"http://localhost:8080/callback",
			"http://localhost:8080/callback",
		},
		{
			"invalid localhost",
			[]string{"http://localhost:8080/callback"},
			"http://localhost:8081/callback",
			"",
		},
		{
			"valid ipv4 loopback redirect",
			[]string{"http://127.0.0.1/callback"},
			"http://127.0.0.1:12345/callback",
			"http://127.0.0.1:12345/callback",
		},
		{
			"valid ipv6 loopback redirect",
			[]string{"http://[::1]/callback"},
			"http://[::1]:12345/callback",
			"http://[::1]:12345/callback",
		},
		{
			"invalid https ipv4 loopback redirect",
			[]string{"https://127.0.0.1/callback"},
			"https://127.0.0.1:12345/callback",
			"",
		},
		{
			"invalid localhost loopback redirect",
			[]string{"http://localhost/callback"},
			"http://localhost:12345/callback",
			"",
		},
	}

	for _, tt := range tests {
		result, err := GetCallbackURLFromList(tt.callbackURLs, tt.input)
		if err != nil {
			t.Errorf("GetCallbackURLFromList(%q, %q) failed: %v", tt.callbackURLs, tt.input, err)
		}
		if result != tt.expected {
			t.Errorf("GetCallbackURLFromList(%q, %q) = %q, want %q", tt.callbackURLs, tt.input, result, tt.expected)
		}
	}
}
