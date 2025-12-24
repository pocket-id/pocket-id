package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMatchCallbackURL(t *testing.T) {
	tests := []struct {
		name        string
		pattern     string
		input       string
		shouldMatch bool
	}{
		// Basic matching
		{
			"exact match",
			"https://example.com/callback",
			"https://example.com/callback",
			true,
		},
		{
			"no match",
			"https://example.org/callback",
			"https://example.com/callback",
			false,
		},

		// Scheme
		{
			"scheme mismatch",
			"https://example.com/callback",
			"http://example.com/callback",
			false,
		},
		{
			"wildcard scheme",
			"*://example.com/callback",
			"https://example.com/callback",
			true,
		},

		// Hostname
		{
			"hostname mismatch",
			"https://example.com/callback",
			"https://malicious.com/callback",
			false,
		},
		{
			"wildcard subdomain",
			"https://*.example.com/callback",
			"https://subdomain.example.com/callback",
			true,
		},
		{
			"partial wildcard in hostname prefix",
			"https://app*.example.com/callback",
			"https://app1.example.com/callback",
			true,
		},
		{
			"partial wildcard in hostname suffix",
			"https://*-prod.example.com/callback",
			"https://api-prod.example.com/callback",
			true,
		},
		{
			"partial wildcard in hostname middle",
			"https://app-*-server.example.com/callback",
			"https://app-staging-server.example.com/callback",
			true,
		},
		{
			"subdomain wildcard doesn't match domain hijack attempt",
			"https://*.example.com/callback",
			"https://malicious.site?url=abc.example.com/callback",
			false,
		},
		{
			"hostname mismatch with confusable characters",
			"https://example.com/callback",
			"https://examp1e.com/callback",
			false,
		},
		{
			"hostname mismatch with homograph attack",
			"https://example.com/callback",
			"https://еxample.com/callback",
			false,
		},

		// Port
		{
			"port mismatch",
			"https://example.com:8080/callback",
			"https://example.com:9090/callback",
			false,
		},
		{
			"wildcard port",
			"https://example.com:*/callback",
			"https://example.com:8080/callback",
			true,
		},
		{
			"partial wildcard in port prefix",
			"https://example.com:80*/callback",
			"https://example.com:8080/callback",
			true,
		},

		// Path
		{
			"path mismatch",
			"https://example.com/callback",
			"https://example.com/other",
			false,
		},
		{
			"wildcard path segment",
			"https://example.com/api/*/callback",
			"https://example.com/api/v1/callback",
			true,
		},
		{
			"wildcard entire path",
			"https://example.com/*",
			"https://example.com/callback",
			true,
		},
		{
			"partial wildcard in path prefix",
			"https://example.com/test*",
			"https://example.com/testcase",
			true,
		},
		{
			"partial wildcard in path suffix",
			"https://example.com/*-callback",
			"https://example.com/oauth-callback",
			true,
		},
		{
			"partial wildcard in path middle",
			"https://example.com/api-*-v1/callback",
			"https://example.com/api-internal-v1/callback",
			true,
		},
		{
			"multiple partial wildcards in path",
			"https://example.com/*/test*/callback",
			"https://example.com/v1/testing/callback",
			true,
		},
		{
			"multiple wildcard segments in path",
			"https://example.com/**/callback",
			"https://example.com/api/v1/foo/bar/callback",
			true,
		},
		{
			"multiple wildcard segments in path",
			"https://example.com/**/v1/**/callback",
			"https://example.com/api/v1/foo/bar/callback",
			true,
		},
		{
			"partial wildcard matching full path segment",
			"https://example.com/foo-*",
			"https://example.com/foo-bar",
			true,
		},

		// Credentials
		{
			"username mismatch",
			"https://user:pass@example.com/callback",
			"https://admin:pass@example.com/callback",
			false,
		},
		{
			"missing credentials",
			"https://user:pass@example.com/callback",
			"https://example.com/callback",
			false,
		},
		{
			"unexpected credentials",
			"https://example.com/callback",
			"https://user:pass@example.com/callback",
			false,
		},
		{
			"wildcard password",
			"https://user:*@example.com/callback",
			"https://user:secret123@example.com/callback",
			true,
		},
		{
			"partial wildcard in username",
			"https://admin*:pass@example.com/callback",
			"https://admin123:pass@example.com/callback",
			true,
		},
		{
			"partial wildcard in password",
			"https://user:pass*@example.com/callback",
			"https://user:password123@example.com/callback",
			true,
		},
		{
			"wildcard password doesn't allow domain hijack",
			"https://user:*@example.com/callback",
			"https://user:password@malicious.site#example.com/callback",
			false,
		},
		{
			"credentials with @ in password trying to hijack hostname",
			"https://user:pass@example.com/callback",
			"https://user:pass@evil.com@example.com/callback",
			false,
		},

		// Query parameters
		{
			"extra query parameter",
			"https://example.com/callback?code=*",
			"https://example.com/callback?code=abc123&extra=value",
			false,
		},
		{
			"missing query parameter",
			"https://example.com/callback?code=*&state=*",
			"https://example.com/callback?code=abc123",
			false,
		},
		{
			"query parameter after fragment",
			"https://example.com/callback?code=123",
			"https://example.com/callback#section?code=123",
			false,
		},
		{
			"query parameter name mismatch",
			"https://example.com/callback?code=*",
			"https://example.com/callback?token=abc123",
			false,
		},
		{
			"wildcard query parameter",
			"https://example.com/callback?code=*",
			"https://example.com/callback?code=abc123",
			true,
		},
		{
			"multiple query parameters",
			"https://example.com/callback?code=*&state=*",
			"https://example.com/callback?code=abc123&state=xyz789",
			true,
		},
		{
			"query parameters in different order",
			"https://example.com/callback?state=*&code=*",
			"https://example.com/callback?code=abc123&state=xyz789",
			true,
		},
		{
			"exact query parameter value",
			"https://example.com/callback?mode=production",
			"https://example.com/callback?mode=production",
			true,
		},
		{
			"query parameter value mismatch",
			"https://example.com/callback?mode=production",
			"https://example.com/callback?mode=development",
			false,
		},
		{
			"mixed exact and wildcard query parameters",
			"https://example.com/callback?mode=production&code=*",
			"https://example.com/callback?mode=production&code=abc123",
			true,
		},
		{
			"mixed exact and wildcard with wrong exact value",
			"https://example.com/callback?mode=production&code=*",
			"https://example.com/callback?mode=development&code=abc123",
			false,
		},
		{
			"multiple values for same parameter",
			"https://example.com/callback?param=*&param=*",
			"https://example.com/callback?param=value1&param=value2",
			true,
		},
		{
			"unexpected query parameters",
			"https://example.com/callback",
			"https://example.com/callback?extra=value",
			false,
		},
		{
			"query parameter with redirect to external site",
			"https://example.com/callback?code=*",
			"https://example.com/callback?code=123&redirect=https://evil.com",
			false,
		},
		{
			"open redirect via encoded URL in query param",
			"https://example.com/callback?state=*",
			"https://example.com/callback?state=abc&next=//evil.com",
			false,
		},

		// Fragment
		{
			"fragment ignored when both pattern and input have fragment",
			"https://example.com/callback#fragment",
			"https://example.com/callback#fragment",
			true,
		},
		{
			"fragment ignored when pattern has fragment but input doesn't",
			"https://example.com/callback#fragment",
			"https://example.com/callback",
			true,
		},
		{
			"fragment ignored when input has fragment but pattern doesn't",
			"https://example.com/callback",
			"https://example.com/callback#section",
			true,
		},

		// Path traversal and injection attempts
		{
			"path traversal attempt",
			"https://example.com/callback",
			"https://example.com/../admin/callback",
			false,
		},
		{
			"backslash instead of forward slash",
			"https://example.com/callback",
			"https://example.com\\callback",
			false,
		},
		{
			"double slash in hostname (protocol smuggling)",
			"https://example.com/callback",
			"https://example.com//evil.com/callback",
			false,
		},
		{
			"CRLF injection attempt in path",
			"https://example.com/callback",
			"https://example.com/callback%0d%0aLocation:%20https://evil.com",
			false,
		},
		{
			"null byte injection",
			"https://example.com/callback",
			"https://example.com/callback%00.evil.com",
			false,
		},
	}

	for _, tt := range tests {
		matches, err := matchCallbackURL(tt.pattern, tt.input)
		require.NoError(t, err, tt.name)
		assert.Equal(t, tt.shouldMatch, matches, tt.name)

	}
}

func TestGetCallbackURLFromList_LoopbackSpecialHandling(t *testing.T) {
	tests := []struct {
		name             string
		urls             []string
		inputCallbackURL string
		expectedURL      string
		expectMatch      bool
	}{
		{
			name:             "127.0.0.1 with dynamic port - exact match",
			urls:             []string{"http://127.0.0.1/callback"},
			inputCallbackURL: "http://127.0.0.1:8080/callback",
			expectedURL:      "http://127.0.0.1:8080/callback",
			expectMatch:      true,
		},
		{
			name:             "127.0.0.1 with different port",
			urls:             []string{"http://127.0.0.1/callback"},
			inputCallbackURL: "http://127.0.0.1:9999/callback",
			expectedURL:      "http://127.0.0.1:9999/callback",
			expectMatch:      true,
		},
		{
			name:             "IPv6 loopback with dynamic port",
			urls:             []string{"http://[::1]/callback"},
			inputCallbackURL: "http://[::1]:8080/callback",
			expectedURL:      "http://[::1]:8080/callback",
			expectMatch:      true,
		},
		{
			name:             "IPv6 loopback without brackets in input",
			urls:             []string{"http://[::1]/callback"},
			inputCallbackURL: "http://::1:8080/callback",
			expectedURL:      "http://::1:8080/callback",
			expectMatch:      true,
		},
		{
			name:             "localhost with dynamic port",
			urls:             []string{"http://localhost/callback"},
			inputCallbackURL: "http://localhost:8080/callback",
			expectedURL:      "http://localhost:8080/callback",
			expectMatch:      true,
		},
		{
			name:             "https loopback doesn't trigger special handling",
			urls:             []string{"https://127.0.0.1/callback"},
			inputCallbackURL: "https://127.0.0.1:8080/callback",
			expectedURL:      "",
			expectMatch:      false,
		},
		{
			name:             "loopback with path match",
			urls:             []string{"http://127.0.0.1/auth/*"},
			inputCallbackURL: "http://127.0.0.1:3000/auth/callback",
			expectedURL:      "http://127.0.0.1:3000/auth/callback",
			expectMatch:      true,
		},
		{
			name:             "loopback with path mismatch",
			urls:             []string{"http://127.0.0.1/callback"},
			inputCallbackURL: "http://127.0.0.1:8080/different",
			expectedURL:      "",
			expectMatch:      false,
		},
		{
			name:             "non-loopback IP",
			urls:             []string{"http://192.168.1.1/callback"},
			inputCallbackURL: "http://192.168.1.1:8080/callback",
			expectedURL:      "",
			expectMatch:      false,
		},
		{
			name:             "wildcard matches loopback",
			urls:             []string{"*"},
			inputCallbackURL: "http://127.0.0.1:8080/callback",
			expectedURL:      "http://127.0.0.1:8080/callback",
			expectMatch:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := GetCallbackURLFromList(tt.urls, tt.inputCallbackURL)
			require.NoError(t, err)
			if tt.expectMatch {
				assert.Equal(t, tt.expectedURL, result)
			} else {
				assert.Empty(t, result)
			}
		})
	}
}

func TestGetCallbackURLFromList_MultiplePatterns(t *testing.T) {
	tests := []struct {
		name             string
		urls             []string
		inputCallbackURL string
		expectedURL      string
		expectMatch      bool
	}{
		{
			name: "matches first pattern",
			urls: []string{
				"https://example.com/callback",
				"https://example.org/callback",
			},
			inputCallbackURL: "https://example.com/callback",
			expectedURL:      "https://example.com/callback",
			expectMatch:      true,
		},
		{
			name: "matches second pattern",
			urls: []string{
				"https://example.com/callback",
				"https://example.org/callback",
			},
			inputCallbackURL: "https://example.org/callback",
			expectedURL:      "https://example.org/callback",
			expectMatch:      true,
		},
		{
			name: "matches none",
			urls: []string{
				"https://example.com/callback",
				"https://example.org/callback",
			},
			inputCallbackURL: "https://malicious.com/callback",
			expectedURL:      "",
			expectMatch:      false,
		},
		{
			name: "matches wildcard pattern",
			urls: []string{
				"https://example.com/callback",
				"https://*.example.org/callback",
			},
			inputCallbackURL: "https://subdomain.example.org/callback",
			expectedURL:      "https://subdomain.example.org/callback",
			expectMatch:      true,
		},
		{
			name:             "empty pattern list",
			urls:             []string{},
			inputCallbackURL: "https://example.com/callback",
			expectedURL:      "",
			expectMatch:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := GetCallbackURLFromList(tt.urls, tt.inputCallbackURL)
			require.NoError(t, err)
			if tt.expectMatch {
				assert.Equal(t, tt.expectedURL, result)
			} else {
				assert.Empty(t, result)
			}
		})
	}
}

func TestMatchPath(t *testing.T) {
	tests := []struct {
		name        string
		pattern     string
		input       string
		shouldMatch bool
	}{
		// Exact matches
		{
			name:        "exact match",
			pattern:     "/callback",
			input:       "/callback",
			shouldMatch: true,
		},
		{
			name:        "exact mismatch",
			pattern:     "/callback",
			input:       "/other",
			shouldMatch: false,
		},
		{
			name:        "empty paths",
			pattern:     "",
			input:       "",
			shouldMatch: true,
		},

		// Single wildcard (*)
		{
			name:        "single wildcard matches segment",
			pattern:     "/api/*/callback",
			input:       "/api/v1/callback",
			shouldMatch: true,
		},
		{
			name:        "single wildcard doesn't match multiple segments",
			pattern:     "/api/*/callback",
			input:       "/api/v1/v2/callback",
			shouldMatch: false,
		},
		{
			name:        "single wildcard at end",
			pattern:     "/callback/*",
			input:       "/callback/test",
			shouldMatch: true,
		},
		{
			name:        "single wildcard at start",
			pattern:     "/*/callback",
			input:       "/api/callback",
			shouldMatch: true,
		},
		{
			name:        "multiple single wildcards",
			pattern:     "/*/test/*",
			input:       "/api/test/callback",
			shouldMatch: true,
		},
		{
			name:        "partial wildcard prefix",
			pattern:     "/test*",
			input:       "/testing",
			shouldMatch: true,
		},
		{
			name:        "partial wildcard suffix",
			pattern:     "/*-callback",
			input:       "/oauth-callback",
			shouldMatch: true,
		},
		{
			name:        "partial wildcard middle",
			pattern:     "/api-*-v1",
			input:       "/api-internal-v1",
			shouldMatch: true,
		},

		// Double wildcard (**)
		{
			name:        "double wildcard matches multiple segments",
			pattern:     "/api/**/callback",
			input:       "/api/v1/v2/v3/callback",
			shouldMatch: true,
		},
		{
			name:        "double wildcard matches single segment",
			pattern:     "/api/**/callback",
			input:       "/api/v1/callback",
			shouldMatch: true,
		},
		{
			name:        "double wildcard doesn't match when pattern has extra slashes",
			pattern:     "/api/**/callback",
			input:       "/api/callback",
			shouldMatch: false,
		},
		{
			name:        "double wildcard at end",
			pattern:     "/api/**",
			input:       "/api/v1/v2/callback",
			shouldMatch: true,
		},
		{
			name:        "double wildcard in middle",
			pattern:     "/api/**/v2/**/callback",
			input:       "/api/v1/v2/v3/v4/callback",
			shouldMatch: true,
		},

		// Complex patterns
		{
			name:        "mix of single and double wildcards",
			pattern:     "/*/api/**/callback",
			input:       "/app/api/v1/v2/callback",
			shouldMatch: true,
		},
		{
			name:        "wildcard with special characters",
			pattern:     "/callback-*",
			input:       "/callback-123",
			shouldMatch: true,
		},
		{
			name:        "path with query-like string (no special handling)",
			pattern:     "/callback?code=*",
			input:       "/callback?code=abc",
			shouldMatch: true,
		},

		// Edge cases
		{
			name:        "single wildcard matches empty segment",
			pattern:     "/api/*/callback",
			input:       "/api//callback",
			shouldMatch: true,
		},
		{
			name:        "pattern longer than input",
			pattern:     "/api/v1/callback",
			input:       "/api",
			shouldMatch: false,
		},
		{
			name:        "input longer than pattern",
			pattern:     "/api",
			input:       "/api/v1/callback",
			shouldMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches, err := matchPath(tt.pattern, tt.input)
			require.NoError(t, err)
			assert.Equal(t, tt.shouldMatch, matches)
		})
	}
}

func TestSplitParts(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectedParts []string
		expectedPath  string
	}{
		{
			name:          "simple https URL",
			input:         "https://example.com/callback",
			expectedParts: []string{"https", "example", "com"},
			expectedPath:  "/callback",
		},
		{
			name:          "URL with port",
			input:         "https://example.com:8080/callback",
			expectedParts: []string{"https", "example", "com", "8080"},
			expectedPath:  "/callback",
		},
		{
			name:          "URL with subdomain",
			input:         "https://api.example.com/callback",
			expectedParts: []string{"https", "api", "example", "com"},
			expectedPath:  "/callback",
		},
		{
			name:          "URL with credentials",
			input:         "https://user:pass@example.com/callback",
			expectedParts: []string{"https", "user", "pass", "example", "com"},
			expectedPath:  "/callback",
		},
		{
			name:          "URL without path",
			input:         "https://example.com",
			expectedParts: []string{"https", "example", "com"},
			expectedPath:  "",
		},
		{
			name:          "URL with deep path",
			input:         "https://example.com/api/v1/callback",
			expectedParts: []string{"https", "example", "com"},
			expectedPath:  "/api/v1/callback",
		},
		{
			name:          "URL with path and query",
			input:         "https://example.com/callback?code=123",
			expectedParts: []string{"https", "example", "com"},
			expectedPath:  "/callback?code=123",
		},
		{
			name:          "URL with trailing slash",
			input:         "https://example.com/",
			expectedParts: []string{"https", "example", "com"},
			expectedPath:  "/",
		},
		{
			name:          "URL with multiple subdomains",
			input:         "https://api.v1.staging.example.com/callback",
			expectedParts: []string{"https", "api", "v1", "staging", "example", "com"},
			expectedPath:  "/callback",
		},
		{
			name:          "URL with port and credentials",
			input:         "https://user:pass@example.com:8080/callback",
			expectedParts: []string{"https", "user", "pass", "example", "com", "8080"},
			expectedPath:  "/callback",
		},
		{
			name:          "scheme with authority separator but no slash",
			input:         "http://example.com",
			expectedParts: []string{"http", "example", "com"},
			expectedPath:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parts, path := splitParts(tt.input)
			assert.Equal(t, tt.expectedParts, parts, "parts mismatch")
			assert.Equal(t, tt.expectedPath, path, "path mismatch")
		})
	}
}
