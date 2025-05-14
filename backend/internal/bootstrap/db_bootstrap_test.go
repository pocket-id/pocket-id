package bootstrap

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseSqliteConnectionString(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expected      string
		expectedError bool
	}{
		{
			name:     "basic file path",
			input:    "file:test.db",
			expected: "file:test.db",
		},
		{
			name:     "adds file: prefix if missing",
			input:    "test.db",
			expected: "file:test.db",
		},
		{
			name:     "converts _busy_timeout to pragma",
			input:    "file:test.db?_busy_timeout=5000",
			expected: "file:test.db?_pragma=busy_timeout%285000%29",
		},
		{
			name:     "converts _timeout to pragma",
			input:    "file:test.db?_timeout=5000",
			expected: "file:test.db?_pragma=busy_timeout%285000%29",
		},
		{
			name:     "converts _foreign_keys to pragma",
			input:    "file:test.db?_foreign_keys=1",
			expected: "file:test.db?_pragma=foreign_keys%281%29",
		},
		{
			name:     "converts _fk to pragma",
			input:    "file:test.db?_fk=1",
			expected: "file:test.db?_pragma=foreign_keys%281%29",
		},
		{
			name:     "converts _synchronous to pragma",
			input:    "file:test.db?_synchronous=NORMAL",
			expected: "file:test.db?_pragma=synchronous%28NORMAL%29",
		},
		{
			name:     "converts _sync to pragma",
			input:    "file:test.db?_sync=NORMAL",
			expected: "file:test.db?_pragma=synchronous%28NORMAL%29",
		},
		{
			name:     "converts _auto_vacuum to pragma",
			input:    "file:test.db?_auto_vacuum=FULL",
			expected: "file:test.db?_pragma=auto_vacuum%28FULL%29",
		},
		{
			name:     "converts _vacuum to pragma",
			input:    "file:test.db?_vacuum=FULL",
			expected: "file:test.db?_pragma=auto_vacuum%28FULL%29",
		},
		{
			name:     "converts _case_sensitive_like to pragma",
			input:    "file:test.db?_case_sensitive_like=1",
			expected: "file:test.db?_pragma=case_sensitive_like%281%29",
		},
		{
			name:     "converts _cslike to pragma",
			input:    "file:test.db?_cslike=1",
			expected: "file:test.db?_pragma=case_sensitive_like%281%29",
		},
		{
			name:     "converts _locking_mode to pragma",
			input:    "file:test.db?_locking_mode=EXCLUSIVE",
			expected: "file:test.db?_pragma=locking_mode%28EXCLUSIVE%29",
		},
		{
			name:     "converts _locking to pragma",
			input:    "file:test.db?_locking=EXCLUSIVE",
			expected: "file:test.db?_pragma=locking_mode%28EXCLUSIVE%29",
		},
		{
			name:     "converts _secure_delete to pragma",
			input:    "file:test.db?_secure_delete=1",
			expected: "file:test.db?_pragma=secure_delete%281%29",
		},
		{
			name:     "preserves unrecognized parameters",
			input:    "file:test.db?mode=rw&cache=shared",
			expected: "file:test.db?cache=shared&mode=rw",
		},
		{
			name:     "handles multiple parameters",
			input:    "file:test.db?_fk=1&mode=rw&_timeout=5000",
			expected: "file:test.db?_pragma=foreign_keys%281%29&_pragma=busy_timeout%285000%29&mode=rw",
		},
		{
			name:          "invalid URL format",
			input:         "file:invalid#$%^&*@test.db",
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseSqliteConnectionString(tt.input)

			if tt.expectedError {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)

			// Parse both URLs to compare components independently
			expectedURL, err := url.Parse(tt.expected)
			require.NoError(t, err)

			resultURL, err := url.Parse(result)
			require.NoError(t, err)

			// Compare scheme and path components
			assert.Equal(t, expectedURL.Scheme, resultURL.Scheme)
			assert.Equal(t, expectedURL.Path, resultURL.Path)

			// Compare query parameters regardless of order
			expectedQuery := expectedURL.Query()
			resultQuery := resultURL.Query()

			assert.Equal(t, len(expectedQuery), len(resultQuery))

			for key, expectedValues := range expectedQuery {
				resultValues, ok := resultQuery[key]
				_ = assert.True(t, ok) &&
					assert.ElementsMatch(t, expectedValues, resultValues)
			}
		})
	}
}
