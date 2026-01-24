package utils

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBearerAuth(t *testing.T) {
	tests := []struct {
		name          string
		authHeader    string
		expectedToken string
		expectedFound bool
	}{
		{
			name:          "Valid bearer token",
			authHeader:    "Bearer token123",
			expectedToken: "token123",
			expectedFound: true,
		},
		{
			name:          "Valid bearer token with mixed case",
			authHeader:    "beARer token456",
			expectedToken: "token456",
			expectedFound: true,
		},
		{
			name:          "No bearer prefix",
			authHeader:    "Basic dXNlcjpwYXNz",
			expectedToken: "",
			expectedFound: false,
		},
		{
			name:          "Empty auth header",
			authHeader:    "",
			expectedToken: "",
			expectedFound: false,
		},
		{
			name:          "Bearer prefix only",
			authHeader:    "Bearer ",
			expectedToken: "",
			expectedFound: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "http://example.com", nil)
			require.NoError(t, err, "Failed to create request")

			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}

			token, found := BearerAuth(req)

			assert.Equal(t, tt.expectedFound, found)
			assert.Equal(t, tt.expectedToken, token)
		})
	}
}

func TestOAuthClientBasicAuth(t *testing.T) {
	tests := []struct {
		name                 string
		authHeader           string
		expectedClientID     string
		expectedClientSecret string
		expectedOk           bool
	}{
		{
			name:                 "Valid client ID and secret in header (example from RFC 6749)",
			authHeader:           "Basic czZCaGRSa3F0Mzo3RmpmcDBaQnIxS3REUmJuZlZkbUl3",
			expectedClientID:     "s6BhdRkqt3",
			expectedClientSecret: "7Fjfp0ZBr1KtDRbnfVdmIw",
			expectedOk:           true,
		},
		{
			name:             "Valid client ID and secret in header (escaped values)",
			authHeader:       "Basic ZTUwOTcyYmQtNmUzMi00OTU3LWJhZmMtMzU0MTU3ZjI1NDViOislMjUlMjYlMkIlQzIlQTMlRTIlODIlQUM=",
			expectedClientID: "e50972bd-6e32-4957-bafc-354157f2545b",
			// This is the example string from RFC 6749, Appendix B.
			expectedClientSecret: " %&+£€",
			expectedOk:           true,
		},
		{
			name:                 "Empty auth header",
			authHeader:           "",
			expectedClientID:     "",
			expectedClientSecret: "",
			expectedOk:           false,
		},
		{
			name:                 "Basic prefix only",
			authHeader:           "Basic ",
			expectedClientID:     "",
			expectedClientSecret: "",
			expectedOk:           false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "http://example.com", nil)
			require.NoError(t, err, "Failed to create request")

			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}

			clientId, clientSecret, ok := OAuthClientBasicAuth(req)

			assert.Equal(t, tt.expectedOk, ok)

			if tt.expectedOk {
				assert.Equal(t, tt.expectedClientID, clientId)
				assert.Equal(t, tt.expectedClientSecret, clientSecret)
			}
		})
	}
}
