package service

import (
	"sync/atomic"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/pocket-id/pocket-id/backend/internal/model"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
	"github.com/stretchr/testify/require"
)

// NewTestAppConfigService is a function used by tests to create AppConfigService objects with pre-defined configuration values
func NewTestAppConfigService(config *model.AppConfig) *AppConfigService {
	service := &AppConfigService{
		dbConfig: atomic.Pointer[model.AppConfig]{},
	}
	service.dbConfig.Store(config)

	return service
}

func TestLoadDbConfig(t *testing.T) {
	t.Run("empty config table", func(t *testing.T) {
		db := newAppConfigTestDatabaseForTest(t)
		service := &AppConfigService{
			db: db,
		}

		// Load the config
		err := service.LoadDbConfig(t.Context())
		require.NoError(t, err)

		// Config should be equal to default config
		require.EqualValues(t, service.GetDbConfig(), service.getDefaultDbConfig())
	})

	t.Run("loads value from config table", func(t *testing.T) {
		db := newAppConfigTestDatabaseForTest(t)

		// Populate the config table with some initial values
		err := db.
			Create([]model.AppConfigVariable{
				// Should be set to the default value because it's an empty string
				{Key: "appName", Value: ""},
				// Overrides default value
				{Key: "sessionDuration", Value: "5"},
				// Does not have a default value
				{Key: "smtpHost", Value: "example"},
			}).
			Error
		require.NoError(t, err)

		// Load the config
		service := &AppConfigService{
			db: db,
		}
		err = service.LoadDbConfig(t.Context())
		require.NoError(t, err)

		// Values should match expected ones
		expect := service.getDefaultDbConfig()
		expect.SessionDuration.Value = "5"
		expect.SmtpHost.Value = "example"
		require.EqualValues(t, service.GetDbConfig(), expect)
	})
}

// Implements gorm's logger.Writer interface
type testLoggerAdapter struct {
	t *testing.T
}

func (l testLoggerAdapter) Printf(format string, args ...any) {
	l.t.Logf(format, args...)
}

func newAppConfigTestDatabaseForTest(t *testing.T) *gorm.DB {
	t.Helper()

	// Get a name for this in-memory database that is specific to the test
	dbName := utils.CreateSha256Hash(t.Name())

	// Connect to a new in-memory SQL database
	db, err := gorm.Open(
		sqlite.Open("file:"+dbName+"?mode=memory&cache=shared"),
		&gorm.Config{
			TranslateError: true,
			Logger: logger.New(
				testLoggerAdapter{t: t},
				logger.Config{
					SlowThreshold:             200 * time.Millisecond,
					LogLevel:                  logger.Info,
					IgnoreRecordNotFoundError: false,
					ParameterizedQueries:      false,
					Colorful:                  false,
				},
			),
		})
	require.NoError(t, err, "Failed to connect to test database")

	// Create the app_config_variables table
	err = db.Exec(`
CREATE TABLE app_config_variables
(
    key           VARCHAR(100) NOT NULL PRIMARY KEY,
    value         TEXT NOT NULL
)
`).Error
	require.NoError(t, err, "Failed to create test config table")

	return db
}
