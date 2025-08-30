package bootstrap

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsSqliteInMemory(t *testing.T) {
	tests := []struct {
		name     string
		connStr  string
		expected bool
	}{
		{
			name:     "memory database with :memory:",
			connStr:  ":memory:",
			expected: true,
		},
		{
			name:     "memory database with file::memory:",
			connStr:  "file::memory:",
			expected: true,
		},
		{
			name:     "memory database with :MEMORY: (uppercase)",
			connStr:  ":MEMORY:",
			expected: true,
		},
		{
			name:     "memory database with FILE::MEMORY: (uppercase)",
			connStr:  "FILE::MEMORY:",
			expected: true,
		},
		{
			name:     "memory database with mixed case",
			connStr:  ":Memory:",
			expected: true,
		},
		{
			name:     "has mode=memory",
			connStr:  "file:data?mode=memory",
			expected: true,
		},
		{
			name:     "file database",
			connStr:  "data.db",
			expected: false,
		},
		{
			name:     "file database with path",
			connStr:  "/path/to/data.db",
			expected: false,
		},
		{
			name:     "file database with file: prefix",
			connStr:  "file:data.db",
			expected: false,
		},
		{
			name:     "empty string",
			connStr:  "",
			expected: false,
		},
		{
			name:     "string containing memory but not at start",
			connStr:  "data:memory:.db",
			expected: false,
		},
		{
			name:     "has mode=ro",
			connStr:  "file:data?mode=ro",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isSqliteInMemory(tt.connStr)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConvertSqlitePragmaArgs(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "basic file path",
			input:    "file:test.db",
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resultURL, _ := url.Parse(tt.input)
			convertSqlitePragmaArgs(resultURL)

			// Parse both URLs to compare components independently
			expectedURL, err := url.Parse(tt.expected)
			require.NoError(t, err)

			// Compare scheme and path components
			compareQueryStrings(t, expectedURL, resultURL)
		})
	}
}

func TestAddSqliteDefaultParameters(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		isMemoryDB  bool
		expected    string
		expectError bool
	}{
		{
			name:       "basic file database",
			input:      "file:test.db",
			isMemoryDB: false,
			expected:   "file:test.db?_pragma=busy_timeout%282500%29&_pragma=foreign_keys%281%29&_pragma=journal_mode%28WAL%29&_txlock=immediate",
		},
		{
			name:       "in-memory database",
			input:      "file::memory:",
			isMemoryDB: true,
			expected:   "file::memory:?_pragma=busy_timeout%282500%29&_pragma=foreign_keys%281%29&_pragma=journal_mode%28MEMORY%29&_txlock=immediate&cache=shared",
		},
		{
			name:       "read-only database with mode=ro",
			input:      "file:test.db?mode=ro",
			isMemoryDB: false,
			expected:   "file:test.db?_pragma=busy_timeout%282500%29&_pragma=foreign_keys%281%29&_pragma=journal_mode%28DELETE%29&_txlock=immediate&mode=ro",
		},
		{
			name:       "immutable database",
			input:      "file:test.db?immutable=1",
			isMemoryDB: false,
			expected:   "file:test.db?_pragma=busy_timeout%282500%29&_pragma=foreign_keys%281%29&_pragma=journal_mode%28DELETE%29&_txlock=immediate&immutable=1",
		},
		{
			name:       "database with existing _txlock",
			input:      "file:test.db?_txlock=deferred",
			isMemoryDB: false,
			expected:   "file:test.db?_pragma=busy_timeout%282500%29&_pragma=foreign_keys%281%29&_pragma=journal_mode%28WAL%29&_txlock=deferred",
		},
		{
			name:       "database with existing busy_timeout pragma",
			input:      "file:test.db?_pragma=busy_timeout%285000%29",
			isMemoryDB: false,
			expected:   "file:test.db?_pragma=busy_timeout%285000%29&_pragma=foreign_keys%281%29&_pragma=journal_mode%28WAL%29&_txlock=immediate",
		},
		{
			name:       "database with existing journal_mode pragma",
			input:      "file:test.db?_pragma=journal_mode%28DELETE%29",
			isMemoryDB: false,
			expected:   "file:test.db?_pragma=busy_timeout%282500%29&_pragma=foreign_keys%281%29&_pragma=journal_mode%28DELETE%29&_txlock=immediate",
		},
		{
			name:        "database with forbidden foreign_keys pragma",
			input:       "file:test.db?_pragma=foreign_keys%280%29",
			isMemoryDB:  false,
			expectError: true,
		},
		{
			name:       "database with multiple existing pragmas",
			input:      "file:test.db?_pragma=busy_timeout%283000%29&_pragma=journal_mode%28TRUNCATE%29&_pragma=synchronous%28NORMAL%29",
			isMemoryDB: false,
			expected:   "file:test.db?_pragma=busy_timeout%283000%29&_pragma=foreign_keys%281%29&_pragma=journal_mode%28TRUNCATE%29&_pragma=synchronous%28NORMAL%29&_txlock=immediate",
		},
		{
			name:       "in-memory database with cache already set",
			input:      "file::memory:?cache=private",
			isMemoryDB: true,
			expected:   "file::memory:?_pragma=busy_timeout%282500%29&_pragma=foreign_keys%281%29&_pragma=journal_mode%28MEMORY%29&_txlock=immediate&cache=shared",
		},
		{
			name:       "database with mode=rw (not read-only)",
			input:      "file:test.db?mode=rw",
			isMemoryDB: false,
			expected:   "file:test.db?_pragma=busy_timeout%282500%29&_pragma=foreign_keys%281%29&_pragma=journal_mode%28WAL%29&_txlock=immediate&mode=rw",
		},
		{
			name:       "database with immutable=0 (not immutable)",
			input:      "file:test.db?immutable=0",
			isMemoryDB: false,
			expected:   "file:test.db?_pragma=busy_timeout%282500%29&_pragma=foreign_keys%281%29&_pragma=journal_mode%28WAL%29&_txlock=immediate&immutable=0",
		},
		{
			name:       "database with mixed case mode=RO",
			input:      "file:test.db?mode=RO",
			isMemoryDB: false,
			expected:   "file:test.db?_pragma=busy_timeout%282500%29&_pragma=foreign_keys%281%29&_pragma=journal_mode%28DELETE%29&_txlock=immediate&mode=ro",
		},
		{
			name:       "database with mixed case immutable=1",
			input:      "file:test.db?immutable=1",
			isMemoryDB: false,
			expected:   "file:test.db?_pragma=busy_timeout%282500%29&_pragma=foreign_keys%281%29&_pragma=journal_mode%28DELETE%29&_txlock=immediate&immutable=1",
		},
		{
			name:       "complex database configuration",
			input:      "file:test.db?cache=shared&mode=rwc&_txlock=immediate&_pragma=synchronous%28FULL%29",
			isMemoryDB: false,
			expected:   "file:test.db?_pragma=busy_timeout%282500%29&_pragma=foreign_keys%281%29&_pragma=journal_mode%28WAL%29&_pragma=synchronous%28FULL%29&_txlock=immediate&cache=shared&mode=rwc",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resultURL, err := url.Parse(tt.input)
			require.NoError(t, err)

			err = addSqliteDefaultParameters(resultURL, tt.isMemoryDB)

			if tt.expectError {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)

			expectedURL, err := url.Parse(tt.expected)
			require.NoError(t, err)

			compareQueryStrings(t, expectedURL, resultURL)
		})
	}
}

func compareQueryStrings(t *testing.T, expectedURL *url.URL, resultURL *url.URL) {
	t.Helper()

	// Compare scheme and path components
	assert.Equal(t, expectedURL.Scheme, resultURL.Scheme)
	assert.Equal(t, expectedURL.Path, resultURL.Path)

	// Compare query parameters regardless of order
	expectedQuery := expectedURL.Query()
	resultQuery := resultURL.Query()

	assert.Len(t, expectedQuery, len(resultQuery))

	for key, expectedValues := range expectedQuery {
		resultValues, ok := resultQuery[key]
		_ = assert.True(t, ok) &&
			assert.ElementsMatch(t, expectedValues, resultValues)
	}
}
