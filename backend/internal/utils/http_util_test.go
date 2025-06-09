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
