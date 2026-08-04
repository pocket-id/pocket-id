package bootstrap

import (
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pocket-id/pocket-id/backend/internal/common"
)

// TestSqlInstrumentOptions covers the wiring that makes the instrumentation on the connection the single source of SQL logs.
// Gorm is silent, so anything not enabled here is not logged by anyone.
func TestSqlInstrumentOptions(t *testing.T) {
	prevConfig := common.EnvConfig
	t.Cleanup(func() {
		common.EnvConfig = prevConfig
	})

	t.Run("statements are logged at debug level", func(t *testing.T) {
		common.EnvConfig.LogLevel = "debug"

		opts := sqlInstrumentOptions()
		assert.True(t, opts.QueryLoggingEnabled(), "statements must be logged at debug level")
		assert.NotNil(t, opts.Logger(), "a logger is required for any statement log to be emitted")
	})

	t.Run("statements are not logged at other levels", func(t *testing.T) {
		for _, level := range []string{"info", "warn", "error"} {
			common.EnvConfig.LogLevel = level

			assert.Falsef(t, sqlInstrumentOptions().QueryLoggingEnabled(), "statements must not be logged at %s level", level)
		}
	})

	t.Run("slow statements are reported at every level", func(t *testing.T) {
		for _, level := range []string{"debug", "info", "warn", "error"} {
			common.EnvConfig.LogLevel = level

			assert.Equalf(t, 250*time.Millisecond, sqlInstrumentOptions().SlowQueryThreshold(), "slow statements must be reported at %s level", level)
		}
	})

	t.Run("query parameters are omitted unless LOG_QUERY_ARGS is set", func(t *testing.T) {
		common.EnvConfig.LogLevel = "debug"
		common.EnvConfig.LogQueryArgs = false

		assert.False(t, sqlInstrumentOptions().QueryParametersIncluded(), "parameter values must not be included by default, not even at debug level")
	})

	t.Run("query parameters are included when LOG_QUERY_ARGS is set", func(t *testing.T) {
		// The option is independent of the log level, because it also controls whether parameters are attached to trace spans
		for _, level := range []string{"debug", "info", "warn", "error"} {
			common.EnvConfig.LogLevel = level
			common.EnvConfig.LogQueryArgs = true

			assert.Truef(t, sqlInstrumentOptions().QueryParametersIncluded(), "parameter values must be included at %s level", level)
		}
	})
}

func TestAddSqliteDatetimeParams(t *testing.T) {
	tests := []struct {
		name       string
		connString string
		want       url.Values
	}{
		{
			name:       "adds all params to a bare path",
			connString: "data/pocket-id.db",
			want: url.Values{
				"_texttotime":  {"1"},
				"_inttotime":   {"1"},
				"_time_format": {"sqlite"},
			},
		},
		{
			name:       "preserves existing query string params",
			connString: "file:data/pocket-id.db?_pragma=journal_mode(WAL)&_txlock=immediate",
			want: url.Values{
				"_pragma":      {"journal_mode(WAL)"},
				"_txlock":      {"immediate"},
				"_texttotime":  {"1"},
				"_inttotime":   {"1"},
				"_time_format": {"sqlite"},
			},
		},
		{
			name:       "does not override params set explicitly",
			connString: "file:data/pocket-id.db?_time_format=unix&_inttotime=0",
			want: url.Values{
				"_texttotime":  {"1"},
				"_inttotime":   {"0"},
				"_time_format": {"unix"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := addSqliteDatetimeParams(tt.connString)

			path, rawQuery, found := strings.Cut(got, "?")
			require.True(t, found, "result is missing a query string")

			expectedPath, _, _ := strings.Cut(tt.connString, "?")
			assert.Equal(t, expectedPath, path, "path was modified")

			qs, err := url.ParseQuery(rawQuery)
			require.NoError(t, err)
			assert.Equal(t, tt.want, qs)
		})
	}

	t.Run("returns an empty connection string untouched", func(t *testing.T) {
		assert.Empty(t, addSqliteDatetimeParams(""))
	})
}

// TestConnectDatabaseSqlite checks that the connection Pocket ID now opens itself, so it can be instrumented, still behaves like the one Gorm used to open for us.
// The datetime parameters are the part at risk: without them modernc.org/sqlite returns strings, not time.Time, for datetime columns.
//
// This is the only test that may call ConnectDatabase: registering the custom SQLite functions a second time panics, so the function can only run once per process.
func TestConnectDatabaseSqlite(t *testing.T) {
	prevConfig := common.EnvConfig
	t.Cleanup(func() {
		common.EnvConfig = prevConfig
	})

	common.EnvConfig.DbProvider = common.DbProviderSqlite
	common.EnvConfig.DbConnectionString = "file:test-connect-database?mode=memory"

	db, pg, err := ConnectDatabase(t.Context())
	require.NoError(t, err, "Failed to connect to database")
	require.Nil(t, pg, "Postgres pool must be nil for SQLite")

	sqlDB, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = sqlDB.Close()
	})

	t.Run("datetime columns round-trip as time.Time", func(t *testing.T) {
		type record struct {
			ID        string `gorm:"primaryKey"`
			CreatedAt time.Time
		}

		err := db.AutoMigrate(&record{})
		require.NoError(t, err, "Failed to migrate the test table")

		created := time.Now().UTC().Truncate(time.Second)
		err = db.Create(&record{ID: "1", CreatedAt: created}).Error
		require.NoError(t, err)

		var got record
		err = db.First(&got, "id = ?", "1").Error
		require.NoError(t, err)
		assert.True(t, created.Equal(got.CreatedAt), "expected %v, got %v", created, got.CreatedAt)
	})

	t.Run("foreign keys are enabled", func(t *testing.T) {
		var foreignKeys int
		err := sqlDB.QueryRowContext(t.Context(), "PRAGMA foreign_keys").Scan(&foreignKeys)
		require.NoError(t, err)
		assert.Equal(t, 1, foreignKeys)
	})

	t.Run("in-memory databases are capped at one connection", func(t *testing.T) {
		// In-memory databases only see the whole data through a single connection
		assert.Equal(t, 1, sqlDB.Stats().MaxOpenConnections)
	})
}
