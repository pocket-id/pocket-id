package appconfig

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/italypaleale/francis/host/local"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/dto"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	testutils "github.com/pocket-id/pocket-id/backend/internal/utils/testing"
)

// setUIConfigDisabled sets common.EnvConfig.UiConfigDisabled for the duration of the test, restoring the previous global afterwards
func setUIConfigDisabled(t *testing.T, disabled bool) {
	t.Helper()

	original := common.EnvConfig
	t.Cleanup(func() {
		common.EnvConfig = original
	})
	common.EnvConfig.UiConfigDisabled = disabled
}

// newActorBackedService creates an AppConfigService wired to an in-memory test actor host.
// The AppConfig singleton actor is registered and bootstrapped from db, which is also used to load any legacy config.
func newActorBackedService(t *testing.T, db *gorm.DB) *AppConfigService {
	t.Helper()

	var svc *AppConfigService
	testutils.NewActorHostForTest(t, func(t *testing.T, h *local.Host) {
		var err error
		svc, err = NewService(t.Context(), h, db)
		require.NoError(t, err)
	})
	require.NotNil(t, svc)

	// The singleton actor is bootstrapped asynchronously once the host is ready.
	// Before bootstrap runs, the actor has no state and GetConfig decodes it into a non-nil but zero config, so wait until a non-zero (bootstrapped) config is available before returning.
	require.Eventually(t, func() bool {
		cfg, err := svc.GetConfig(t.Context())
		return err == nil && cfg != nil && *cfg != (AppConfigModel{})
	}, 10*time.Second, 20*time.Millisecond, "config actor was not bootstrapped in time")

	return svc
}

// seedLegacyConfig writes a legacy config blob to the kv table so the AppConfig actor bootstraps from it.
func seedLegacyConfig(t *testing.T, db *gorm.DB, values map[string]string) {
	t.Helper()

	blob, err := json.Marshal(values)
	require.NoError(t, err)

	value := string(blob)
	err = db.Create(&model.KV{Key: "config_migrated", Value: &value}).Error
	require.NoError(t, err)
}

// findConfigValue returns the value for key in a slice of AppConfigVariable, and whether it was found.
func findConfigValue(vars []AppConfigVariable, key string) (string, bool) {
	for _, v := range vars {
		if v.Key == key {
			return v.Value, true
		}
	}
	return "", false
}

func TestService_NewService(t *testing.T) {
	t.Run("bootstraps the default config when the database is empty", func(t *testing.T) {
		setUIConfigDisabled(t, false)
		db := testutils.NewDatabaseForTest(t)
		svc := newActorBackedService(t, db)

		cfg, err := svc.GetConfig(t.Context())
		require.NoError(t, err)
		require.NotNil(t, cfg)
		assert.Equal(t, *getDefaultConfig(), *cfg)
	})

	t.Run("bootstraps from the legacy config in the database", func(t *testing.T) {
		setUIConfigDisabled(t, false)
		db := testutils.NewDatabaseForTest(t)
		seedLegacyConfig(t, db, map[string]string{
			"appName":     "Legacy App",
			"ldapEnabled": "true",
		})

		svc := newActorBackedService(t, db)

		cfg, err := svc.GetConfig(t.Context())
		require.NoError(t, err)
		assert.Equal(t, AppConfigValue("Legacy App"), cfg.AppName)
		assert.Equal(t, AppConfigValue("true"), cfg.LdapEnabled)
		// Keys not present in the legacy config keep their defaults
		assert.Equal(t, getDefaultConfig().SessionDuration, cfg.SessionDuration)
	})

	t.Run("loads config from the environment when the UI config is disabled", func(t *testing.T) {
		setUIConfigDisabled(t, true)

		// No actor host or database is needed when the UI config is disabled
		svc, err := NewService(t.Context(), nil, nil)
		require.NoError(t, err)
		require.NotNil(t, svc)

		cfg, err := svc.GetConfig(t.Context())
		require.NoError(t, err)
		assert.Equal(t, *getDefaultConfig(), *cfg)
	})
}

func TestService_GetConfig(t *testing.T) {
	t.Run("returns a fresh copy on each call", func(t *testing.T) {
		setUIConfigDisabled(t, false)
		db := testutils.NewDatabaseForTest(t)
		svc := newActorBackedService(t, db)

		first, err := svc.GetConfig(t.Context())
		require.NoError(t, err)

		// Mutating the returned config must not affect what the service returns later
		first.AppName = "Mutated"

		second, err := svc.GetConfig(t.Context())
		require.NoError(t, err)
		assert.Equal(t, getDefaultConfig().AppName, second.AppName)
	})

	t.Run("returns the env config when the UI config is disabled", func(t *testing.T) {
		setUIConfigDisabled(t, true)
		svc := NewTestAppConfigService(&AppConfigModel{AppName: "From Env"})

		cfg, err := svc.GetConfig(t.Context())
		require.NoError(t, err)
		assert.Equal(t, AppConfigValue("From Env"), cfg.AppName)
	})
}

func TestService_UpdateAppConfig(t *testing.T) {
	t.Run("replaces the configuration and returns all variables", func(t *testing.T) {
		setUIConfigDisabled(t, false)
		db := testutils.NewDatabaseForTest(t)
		svc := newActorBackedService(t, db)

		input := dto.AppConfigUpdateDto{
			AppName:         "Replaced App",
			SessionDuration: "120",
			LdapEnabled:     "true",
			SmtpTls:         "tls",
		}
		res, err := svc.UpdateAppConfig(t.Context(), input)
		require.NoError(t, err)

		// The returned slice includes all variables, both public and private
		got, ok := findConfigValue(res, "appName")
		require.True(t, ok)
		assert.Equal(t, "Replaced App", got)
		got, ok = findConfigValue(res, "smtpTls")
		require.True(t, ok, "the returned slice should include private variables")
		assert.Equal(t, "tls", got)

		// The change is persisted and visible on subsequent reads
		cfg, err := svc.GetConfig(t.Context())
		require.NoError(t, err)
		assert.Equal(t, AppConfigValue("Replaced App"), cfg.AppName)
		assert.Equal(t, AppConfigValue("120"), cfg.SessionDuration)
		assert.Equal(t, AppConfigValue("true"), cfg.LdapEnabled)
	})

	t.Run("resets fields omitted from the DTO to their defaults", func(t *testing.T) {
		setUIConfigDisabled(t, false)
		db := testutils.NewDatabaseForTest(t)
		svc := newActorBackedService(t, db)

		// First set some non-default values
		_, err := svc.UpdateAppConfig(t.Context(), dto.AppConfigUpdateDto{
			AppName:     "First",
			LdapEnabled: "true",
			SmtpTls:     "tls",
		})
		require.NoError(t, err)

		// Replace again with only AppName set: the rest must reset to their defaults
		_, err = svc.UpdateAppConfig(t.Context(), dto.AppConfigUpdateDto{AppName: "Second"})
		require.NoError(t, err)

		cfg, err := svc.GetConfig(t.Context())
		require.NoError(t, err)
		assert.Equal(t, AppConfigValue("Second"), cfg.AppName)
		assert.Equal(t, getDefaultConfig().LdapEnabled, cfg.LdapEnabled)
		assert.Equal(t, getDefaultConfig().SmtpTls, cfg.SmtpTls)
	})

	t.Run("returns UiConfigDisabledError when the UI config is disabled", func(t *testing.T) {
		setUIConfigDisabled(t, true)
		svc := NewTestAppConfigService(nil)

		_, err := svc.UpdateAppConfig(t.Context(), dto.AppConfigUpdateDto{AppName: "X"})
		require.Error(t, err)
		var target *common.UiConfigDisabledError
		assert.ErrorAs(t, err, &target)
	})
}

func TestService_UpdateAppConfigValues(t *testing.T) {
	t.Run("updates a subset of keys and leaves the rest unchanged", func(t *testing.T) {
		setUIConfigDisabled(t, false)
		db := testutils.NewDatabaseForTest(t)
		svc := newActorBackedService(t, db)

		err := svc.UpdateAppConfigValues(t.Context(), "appName", "Updated", "sessionDuration", "120")
		require.NoError(t, err)

		cfg, err := svc.GetConfig(t.Context())
		require.NoError(t, err)
		assert.Equal(t, AppConfigValue("Updated"), cfg.AppName)
		assert.Equal(t, AppConfigValue("120"), cfg.SessionDuration)
		// A key that was not part of the update keeps its default
		assert.Equal(t, getDefaultConfig().LdapEnabled, cfg.LdapEnabled)
	})

	t.Run("an empty value resets the property to its default", func(t *testing.T) {
		setUIConfigDisabled(t, false)
		db := testutils.NewDatabaseForTest(t)
		svc := newActorBackedService(t, db)

		// Set a non-default value first
		err := svc.UpdateAppConfigValues(t.Context(), "sessionDuration", "120")
		require.NoError(t, err)

		// Then reset it with an empty value
		err = svc.UpdateAppConfigValues(t.Context(), "sessionDuration", "")
		require.NoError(t, err)

		cfg, err := svc.GetConfig(t.Context())
		require.NoError(t, err)
		assert.Equal(t, getDefaultConfig().SessionDuration, cfg.SessionDuration)
	})

	t.Run("an odd number of arguments returns an error", func(t *testing.T) {
		setUIConfigDisabled(t, false)
		db := testutils.NewDatabaseForTest(t)
		svc := newActorBackedService(t, db)

		err := svc.UpdateAppConfigValues(t.Context(), "appName")
		require.Error(t, err)
		assert.ErrorContains(t, err, "invalid number of arguments received")
	})

	t.Run("an unknown key returns an error and does not change the config", func(t *testing.T) {
		setUIConfigDisabled(t, false)
		db := testutils.NewDatabaseForTest(t)
		svc := newActorBackedService(t, db)

		err := svc.UpdateAppConfigValues(t.Context(), "thisKeyDoesNotExist", "value")
		require.Error(t, err)

		// The config must not have been modified
		cfg, err := svc.GetConfig(t.Context())
		require.NoError(t, err)
		assert.Equal(t, *getDefaultConfig(), *cfg)
	})

	t.Run("returns UiConfigDisabledError when the UI config is disabled", func(t *testing.T) {
		setUIConfigDisabled(t, true)
		svc := NewTestAppConfigService(nil)

		// An even number of arguments so the count check passes and we reach the UI-config check
		err := svc.UpdateAppConfigValues(t.Context(), "appName", "X")
		require.Error(t, err)
		var target *common.UiConfigDisabledError
		assert.ErrorAs(t, err, &target)
	})
}

func TestService_ListAppConfig(t *testing.T) {
	t.Run("returns only public variables when showAll is false", func(t *testing.T) {
		setUIConfigDisabled(t, false)
		db := testutils.NewDatabaseForTest(t)
		svc := newActorBackedService(t, db)

		vars, err := svc.ListAppConfig(t.Context(), false)
		require.NoError(t, err)

		// appName is public and must be present
		_, ok := findConfigValue(vars, "appName")
		assert.True(t, ok, "public variable appName should be present")
		// smtpHost is not public and must be excluded
		_, ok = findConfigValue(vars, "smtpHost")
		assert.False(t, ok, "private variable smtpHost should be excluded")
	})

	t.Run("returns all variables when showAll is true", func(t *testing.T) {
		setUIConfigDisabled(t, false)
		db := testutils.NewDatabaseForTest(t)
		svc := newActorBackedService(t, db)

		vars, err := svc.ListAppConfig(t.Context(), true)
		require.NoError(t, err)

		_, ok := findConfigValue(vars, "appName")
		assert.True(t, ok)
		_, ok = findConfigValue(vars, "smtpHost")
		assert.True(t, ok, "private variables should be included when showAll is true")
	})

	t.Run("reflects updates made through the service", func(t *testing.T) {
		setUIConfigDisabled(t, false)
		db := testutils.NewDatabaseForTest(t)
		svc := newActorBackedService(t, db)

		err := svc.UpdateAppConfigValues(t.Context(), "appName", "Listed App")
		require.NoError(t, err)

		vars, err := svc.ListAppConfig(t.Context(), true)
		require.NoError(t, err)

		got, ok := findConfigValue(vars, "appName")
		require.True(t, ok)
		assert.Equal(t, "Listed App", got)
	})

	t.Run("redacts sensitive values when the UI config is disabled", func(t *testing.T) {
		setUIConfigDisabled(t, true)
		svc := NewTestAppConfigService(&AppConfigModel{
			SmtpPassword: "super-secret",
		})

		vars, err := svc.ListAppConfig(t.Context(), true)
		require.NoError(t, err)

		got, ok := findConfigValue(vars, "smtpPassword")
		require.True(t, ok)
		assert.Equal(t, "XXXXXXXXXX", got)
	})
}
