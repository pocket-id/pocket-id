package bootstrap

import (
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseSqliteConnString(t *testing.T) {
	// Helper function to parse the connection string and extract query parameters
	parseAndGetQuery := func(t *testing.T, connString string) url.Values {
		t.Helper()

		// Parse the URL
		u, err := url.Parse(connString)
		require.NoError(t, err, "Failed to parse URL")

		// Parse the query string
		query, err := url.ParseQuery(u.RawQuery)
		require.NoError(t, err, "Failed to parse query string")

		return query
	}

	t.Run("simple file path", func(t *testing.T) {
		input := "/path/to/db.sqlite"
		result := parseSqliteConnString(input)

		// Check file: prefix
		assert.True(t, strings.HasPrefix(result, "file:/path/to/db.sqlite?"), "Should add file: prefix")

		// Parse and check query parameters
		query := parseAndGetQuery(t, result)
		assert.Equal(t, "immediate", query.Get("_txlock"), "Should set _txlock=immediate")
		assert.Equal(t, "2500", query.Get("_busy_timeout"), "Should set busy_timeout")
		assert.Equal(t, "WAL", query.Get("_journal_mode"), "Should set journal_mode=WAL for regular DBs")
	})

	t.Run("path with existing file: prefix", func(t *testing.T) {
		input := "file:/path/to/db.sqlite"
		result := parseSqliteConnString(input)

		// Check file: prefix
		assert.True(t, strings.HasPrefix(result, "file:/path/to/db.sqlite?"), "Should preserve file: prefix")

		// Parse and check query parameters
		query := parseAndGetQuery(t, result)
		assert.Equal(t, "immediate", query.Get("_txlock"), "Should set _txlock=immediate")
		assert.Equal(t, "2500", query.Get("_busy_timeout"), "Should set busy_timeout")
		assert.Equal(t, "WAL", query.Get("_journal_mode"), "Should set journal_mode=WAL for regular DBs")
	})

	t.Run("with existing query parameters", func(t *testing.T) {
		input := "/path/to/db.sqlite?cache=private&_foo=5000"
		result := parseSqliteConnString(input)

		// Parse and check query parameters
		query := parseAndGetQuery(t, result)
		assert.Equal(t, "private", query.Get("cache"), "Should preserve existing cache parameter")
		assert.Equal(t, "5000", query.Get("_foo"), "Should preserve existing _foo parameter")
		assert.Equal(t, "immediate", query.Get("_txlock"), "Should set _txlock=immediate")
		assert.Equal(t, "2500", query.Get("_busy_timeout"), "Should set busy_timeout")
		assert.Equal(t, "WAL", query.Get("_journal_mode"), "Should set journal_mode=WAL for regular DBs")
	})

	t.Run("with read-only mode", func(t *testing.T) {
		input := "/path/to/db.sqlite?mode=ro"
		result := parseSqliteConnString(input)

		// Parse and check query parameters
		query := parseAndGetQuery(t, result)
		assert.Equal(t, "ro", query.Get("mode"), "Should preserve mode=ro")
		assert.Equal(t, "immediate", query.Get("_txlock"), "Should set _txlock=immediate")
		assert.Equal(t, "2500", query.Get("_busy_timeout"), "Should set busy_timeout")
		assert.Equal(t, "DELETE", query.Get("_journal_mode"), "Should set journal_mode=DELETE for read-only DBs")
	})

	t.Run("with immutable flag", func(t *testing.T) {
		input := "/path/to/db.sqlite?immutable=1"
		result := parseSqliteConnString(input)

		// Parse and check query parameters
		query := parseAndGetQuery(t, result)
		assert.Equal(t, "1", query.Get("immutable"), "Should preserve immutable=1")
		assert.Equal(t, "immediate", query.Get("_txlock"), "Should set _txlock=immediate")
		assert.Equal(t, "2500", query.Get("_busy_timeout"), "Should set busy_timeout")
		assert.Equal(t, "DELETE", query.Get("_journal_mode"), "Should set journal_mode=DELETE for immutable DBs")
	})

	t.Run("with existing _txlock parameter", func(t *testing.T) {
		input := "/path/to/db.sqlite?_txlock=deferred"
		result := parseSqliteConnString(input)

		// Parse and check query parameters
		query := parseAndGetQuery(t, result)
		assert.Equal(t, "deferred", query.Get("_txlock"), "Should preserve existing _txlock parameter")
	})
}
