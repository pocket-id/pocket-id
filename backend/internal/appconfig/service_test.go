package appconfig

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/italypaleale/francis/actor"
	"github.com/italypaleale/francis/host/local"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pocket-id/pocket-id/backend/internal/apperror"
	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/dto"
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

// newActorBackedService creates an AppConfigService wired to an in-memory test actor host, on which the AppConfig singleton actor is registered and bootstrapped
func newActorBackedService(t *testing.T) *AppConfigService {
	t.Helper()

	svc, _ := newActorBackedServiceWithHost(t)
	return svc
}

// newActorBackedServiceWithHost is newActorBackedService that also returns the actor host
func newActorBackedServiceWithHost(t *testing.T) (*AppConfigService, *local.Host) {
	t.Helper()

	var svc *AppConfigService
	h := testutils.NewActorHostForTest(t, func(t *testing.T, h *local.Host) {
		var err error
		svc, err = NewService(t.Context(), h)
		require.NoError(t, err)
	})
	require.NotNil(t, svc)

	// The singleton actor is bootstrapped asynchronously once the host is ready.
	// Before bootstrap runs, the actor has no state and GetConfig decodes it into a non-nil but zero config, so wait until a non-zero (bootstrapped) config is available before returning.
	require.Eventually(t, func() bool {
		cfg, err := svc.GetConfig(t.Context())
		return err == nil && cfg != nil && *cfg != (AppConfigModel{})
	}, 10*time.Second, 20*time.Millisecond, "config actor was not bootstrapped in time")

	return svc, h
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
	t.Run("bootstraps the default config when there is no state", func(t *testing.T) {
		setUIConfigDisabled(t, false)
		svc := newActorBackedService(t)

		cfg, err := svc.GetConfig(t.Context())
		require.NoError(t, err)
		require.NotNil(t, cfg)
		assert.Equal(t, *getDefaultConfig(), *cfg)
	})

	t.Run("loads config from the environment when the UI config is disabled", func(t *testing.T) {
		setUIConfigDisabled(t, true)
		t.Setenv("APP_NAME", "Environment App")
		t.Setenv("SIGNUP_DEFAULT_CUSTOM_CLAIMS", `[{"key":"role","value":"user"}]`)

		// No actor host is needed when the UI config is disabled
		svc, err := NewService(t.Context(), nil)
		require.NoError(t, err)
		require.NotNil(t, svc)

		cfg, err := svc.GetConfig(t.Context())
		require.NoError(t, err)
		assert.Equal(t, AppConfigValue("Environment App"), cfg.AppName)
		assert.JSONEq(t, `[{"key":"role","value":"user"}]`, string(cfg.SignupDefaultCustomClaims))
	})

	t.Run("rejects invalid environment application configuration", func(t *testing.T) {
		tests := []struct {
			name       string
			envName    string
			value      string
			wantDetail string
		}{
			{
				name:       "malformed custom claims JSON",
				envName:    "SIGNUP_DEFAULT_CUSTOM_CLAIMS",
				value:      `["immich_role": "user"]`,
				wantDetail: `JSON array of objects with string "key" and "value" properties`,
			},
			{
				name:       "custom claim missing value",
				envName:    "SIGNUP_DEFAULT_CUSTOM_CLAIMS",
				value:      `[{"key":"role"}]`,
				wantDetail: `JSON array of objects with string "key" and "value" properties`,
			},
			{
				name:       "user group IDs with the wrong shape",
				envName:    "SIGNUP_DEFAULT_USER_GROUP_IDS",
				value:      `{"group":"id"}`,
				wantDetail: "JSON array of strings",
			},
			{
				name:       "unsafe CIMD URL pattern",
				envName:    "CIMD_URL_ALLOWLIST",
				value:      `["javascript:alert(1)"]`,
				wantDetail: "JSON array of valid callback URL patterns",
			},
			{
				name:       "invalid enum",
				envName:    "WEBAUTHN_USER_VERIFICATION",
				value:      "sometimes",
				wantDetail: "is invalid",
			},
			{
				name:       "invalid boolean type",
				envName:    "REQUIRE_USER_EMAIL",
				value:      "hello",
				wantDetail: "must be either true or false",
			},
			{
				name:       "invalid integer type",
				envName:    "SESSION_DURATION",
				value:      "hello",
				wantDetail: "must be an integer",
			},
		}

		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				setUIConfigDisabled(t, true)
				t.Setenv(test.envName, test.value)

				svc, err := NewService(t.Context(), nil)
				require.Error(t, err)
				assert.Nil(t, svc)
				require.ErrorContains(t, err, test.envName)
				require.ErrorContains(t, err, test.wantDetail)
			})
		}
	})
}

func TestService_GetConfig(t *testing.T) {
	t.Run("returns a fresh copy on each call", func(t *testing.T) {
		setUIConfigDisabled(t, false)
		svc := newActorBackedService(t)

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
		svc := newActorBackedService(t)

		input := AppConfigUpdateDto{
			AppName:         "Replaced App",
			SessionDuration: "120",
			LdapEnabled:     "true",
			SmtpTls:         "tls",
		}
		res, err := svc.UpdateAppConfig(t.Context(), dto.Replace(input))
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
		svc := newActorBackedService(t)

		// First set some non-default values
		_, err := svc.UpdateAppConfig(t.Context(), dto.Replace(AppConfigUpdateDto{
			AppName:     "First",
			LdapEnabled: "true",
			SmtpTls:     "tls",
		}))
		require.NoError(t, err)

		// Replace again with only AppName set: the rest must reset to their defaults
		_, err = svc.UpdateAppConfig(t.Context(), dto.Replace(AppConfigUpdateDto{AppName: "Second"}))
		require.NoError(t, err)

		cfg, err := svc.GetConfig(t.Context())
		require.NoError(t, err)
		assert.Equal(t, AppConfigValue("Second"), cfg.AppName)
		assert.Equal(t, getDefaultConfig().LdapEnabled, cfg.LdapEnabled)
		assert.Equal(t, getDefaultConfig().SmtpTls, cfg.SmtpTls)
	})

	t.Run("returns a UI-config-disabled error when the UI config is disabled", func(t *testing.T) {
		setUIConfigDisabled(t, true)
		svc := NewTestAppConfigService(nil)

		_, err := svc.UpdateAppConfig(t.Context(), dto.Replace(AppConfigUpdateDto{AppName: "X"}))
		require.Error(t, err)
		assert.True(t, apperror.IsCode(err, apperror.CodeUIConfigDisabled))
	})

	t.Run("redacts saved secrets in its response and the list, leaving unset ones empty", func(t *testing.T) {
		setUIConfigDisabled(t, false)
		svc := newActorBackedService(t)

		res, err := svc.UpdateAppConfig(t.Context(), dto.Replace(AppConfigUpdateDto{SmtpPassword: "smtp-secret"}))
		require.NoError(t, err)
		vars, err := svc.ListAppConfig(t.Context(), true)
		require.NoError(t, err)

		for _, got := range [][]AppConfigVariable{res, vars} {
			smtpPassword, _ := findConfigValue(got, "smtpPassword")
			assert.Equal(t, redactedValue, smtpPassword)
			ldapBindPassword, ok := findConfigValue(got, "ldapBindPassword")
			require.True(t, ok)
			assert.Empty(t, ldapBindPassword)
		}

		cfg, err := svc.GetConfig(t.Context())
		require.NoError(t, err)
		assert.Equal(t, AppConfigValue("smtp-secret"), cfg.SmtpPassword)
	})

	t.Run("a redacted secret keeps the saved one and a new secret replaces it", func(t *testing.T) {
		setUIConfigDisabled(t, false)
		svc := newActorBackedService(t)

		_, err := svc.UpdateAppConfig(t.Context(), dto.Replace(AppConfigUpdateDto{SmtpPassword: "smtp-secret"}))
		require.NoError(t, err)

		_, err = svc.UpdateAppConfig(t.Context(), dto.Replace(AppConfigUpdateDto{SmtpPassword: redactedValue}))
		require.NoError(t, err)
		cfg, err := svc.GetConfig(t.Context())
		require.NoError(t, err)
		assert.Equal(t, AppConfigValue("smtp-secret"), cfg.SmtpPassword)

		_, err = svc.UpdateAppConfig(t.Context(), dto.Replace(AppConfigUpdateDto{SmtpPassword: "new-secret"}))
		require.NoError(t, err)
		cfg, err = svc.GetConfig(t.Context())
		require.NoError(t, err)
		assert.Equal(t, AppConfigValue("new-secret"), cfg.SmtpPassword)
	})
}

func TestService_UpdateAppConfigValues(t *testing.T) {
	t.Run("updates a subset of keys and leaves the rest unchanged", func(t *testing.T) {
		setUIConfigDisabled(t, false)
		svc := newActorBackedService(t)

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
		svc := newActorBackedService(t)

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
		svc := newActorBackedService(t)

		err := svc.UpdateAppConfigValues(t.Context(), "appName")
		require.Error(t, err)
		assert.ErrorContains(t, err, "invalid number of arguments received")
	})

	t.Run("an unknown key returns an error and does not change the config", func(t *testing.T) {
		setUIConfigDisabled(t, false)
		svc := newActorBackedService(t)

		err := svc.UpdateAppConfigValues(t.Context(), "thisKeyDoesNotExist", "value")
		require.Error(t, err)

		// The config must not have been modified
		cfg, err := svc.GetConfig(t.Context())
		require.NoError(t, err)
		assert.Equal(t, *getDefaultConfig(), *cfg)
	})

	t.Run("returns a UI-config-disabled error when the UI config is disabled", func(t *testing.T) {
		setUIConfigDisabled(t, true)
		svc := NewTestAppConfigService(nil)

		// An even number of arguments so the count check passes and we reach the UI-config check
		err := svc.UpdateAppConfigValues(t.Context(), "appName", "X")
		require.Error(t, err)
		assert.True(t, apperror.IsCode(err, apperror.CodeUIConfigDisabled))
	})
}

func TestService_ListAppConfig(t *testing.T) {
	t.Run("returns only public variables when showAll is false", func(t *testing.T) {
		setUIConfigDisabled(t, false)
		svc := newActorBackedService(t)

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
		svc := newActorBackedService(t)

		vars, err := svc.ListAppConfig(t.Context(), true)
		require.NoError(t, err)

		_, ok := findConfigValue(vars, "appName")
		assert.True(t, ok)
		_, ok = findConfigValue(vars, "smtpHost")
		assert.True(t, ok, "private variables should be included when showAll is true")
	})

	t.Run("reflects updates made through the service", func(t *testing.T) {
		setUIConfigDisabled(t, false)
		svc := newActorBackedService(t)

		err := svc.UpdateAppConfigValues(t.Context(), "appName", "Listed App")
		require.NoError(t, err)

		vars, err := svc.ListAppConfig(t.Context(), true)
		require.NoError(t, err)

		got, ok := findConfigValue(vars, "appName")
		require.True(t, ok)
		assert.Equal(t, "Listed App", got)
	})

	t.Run("redacts sensitive values set in the environment", func(t *testing.T) {
		setUIConfigDisabled(t, true)
		t.Setenv("SMTP_PASSWORD", "super-secret")
		svc, err := NewService(t.Context(), nil)
		require.NoError(t, err)

		vars, err := svc.ListAppConfig(t.Context(), true)
		require.NoError(t, err)

		got, ok := findConfigValue(vars, "smtpPassword")
		require.True(t, ok)
		assert.Equal(t, "XXXXXXXXXX", got)
	})
}

func TestService_EnvOverrides(t *testing.T) {
	t.Run("environment values take precedence over the saved ones and are never saved", func(t *testing.T) {
		setUIConfigDisabled(t, false)
		t.Setenv("APP_NAME", "Env App")
		t.Setenv("SMTP_PASSWORD", "env-secret")
		svc := newActorBackedService(t)

		res, err := svc.UpdateAppConfig(t.Context(), dto.Replace(AppConfigUpdateDto{
			AppName:         "UI App",
			SessionDuration: "120",
			SmtpPassword:    "XXXXXXXXXX",
		}))
		require.NoError(t, err)
		appName, _ := findConfigValue(res, "appName")
		assert.Equal(t, "Env App", appName)

		cfg, err := svc.GetConfig(t.Context())
		require.NoError(t, err)
		assert.Equal(t, AppConfigValue("Env App"), cfg.AppName)
		assert.Equal(t, AppConfigValue("env-secret"), cfg.SmtpPassword)
		assert.Equal(t, AppConfigValue("120"), cfg.SessionDuration)

		saved, err := svc.getSavedConfig(t.Context())
		require.NoError(t, err)
		assert.Equal(t, getDefaultConfig().AppName, saved.AppName)
		assert.Empty(t, saved.SmtpPassword)

		vars, err := svc.ListAppConfig(t.Context(), true)
		require.NoError(t, err)
		for _, v := range vars {
			assert.Equal(t, v.Key == "appName" || v.Key == "smtpPassword", v.EnvOverride, v.Key)
		}
	})

	t.Run("cross-property rules are left to saving when the UI config is enabled", func(t *testing.T) {
		setUIConfigDisabled(t, false)
		t.Setenv("EMAIL_VERIFICATION", "required")
		svc := newActorBackedService(t)

		cfg, err := svc.GetConfig(t.Context())
		require.NoError(t, err)
		assert.Equal(t, AppConfigValue(EmailVerificationRequired), cfg.EmailVerification)
	})
}

func TestService_SecretEncryption(t *testing.T) {
	// loadSavedState returns the actor state as it is saved in the database
	loadSavedState := func(t *testing.T, h *local.Host) AppConfigModel {
		t.Helper()

		var state AppConfigModel
		err := h.GetState(t.Context(), AppConfigActorType, actor.SingletonActorID, &state)
		require.NoError(t, err)
		return state
	}

	t.Run("secrets are encrypted in the database", func(t *testing.T) {
		setUIConfigDisabled(t, false)
		svc, h := newActorBackedServiceWithHost(t)

		_, err := svc.UpdateAppConfig(t.Context(), dto.Replace(AppConfigUpdateDto{
			SmtpPassword:     "smtp-secret",
			LdapBindPassword: "ldap-secret",
			SmtpUser:         "smtp-user",
		}))
		require.NoError(t, err)

		saved := loadSavedState(t, h)
		assert.True(t, strings.HasPrefix(saved.SmtpPassword.String(), encryptedSecretPrefix))
		assert.True(t, strings.HasPrefix(saved.LdapBindPassword.String(), encryptedSecretPrefix))
		assert.NotContains(t, saved.SmtpPassword.String(), "smtp-secret")
		assert.Equal(t, AppConfigValue("smtp-user"), saved.SmtpUser)

		cfg, err := svc.GetConfig(t.Context())
		require.NoError(t, err)
		assert.Equal(t, AppConfigValue("smtp-secret"), cfg.SmtpPassword)
		assert.Equal(t, AppConfigValue("ldap-secret"), cfg.LdapBindPassword)
	})
}

func TestService_CIMDURLAllowlist(t *testing.T) {
	t.Run("defaults to empty", func(t *testing.T) {
		setUIConfigDisabled(t, false)
		svc := newActorBackedService(t)

		assert.Empty(t, svc.GetCIMDURLAllowlist())
	})

	t.Run("round-trips a valid allowlist", func(t *testing.T) {
		setUIConfigDisabled(t, false)
		svc := newActorBackedService(t)

		_, err := svc.UpdateAppConfig(t.Context(), dto.Replace(AppConfigUpdateDto{
			AppName:          "App",
			SessionDuration:  "60",
			CIMDURLAllowlist: `["https://app.example.com/**","https://*.trusted.com/oauth"]`,
		}))
		require.NoError(t, err)

		assert.Equal(t,
			[]string{"https://app.example.com/**", "https://*.trusted.com/oauth"},
			svc.GetCIMDURLAllowlist(),
		)
	})

	t.Run("rejects an invalid pattern", func(t *testing.T) {
		setUIConfigDisabled(t, false)
		svc := newActorBackedService(t)

		_, err := svc.UpdateAppConfig(t.Context(), dto.Replace(AppConfigUpdateDto{
			AppName:          "App",
			SessionDuration:  "60",
			CIMDURLAllowlist: `["javascript:alert(1)"]`,
		}))
		require.Error(t, err)
	})

	t.Run("returns empty on malformed value", func(t *testing.T) {
		setUIConfigDisabled(t, false)
		svc := newActorBackedService(t)

		require.NoError(t, svc.UpdateAppConfigValues(t.Context(), "cimdUrlAllowlist", "not-json"))
		assert.Empty(t, svc.GetCIMDURLAllowlist())
	})
}

func TestService_LoadEnvConfig(t *testing.T) {
	loadEnvConfig := func() (*AppConfigModel, error) {
		overrides, err := loadEnvOverrides()
		if err != nil {
			return nil, err
		}
		return newEnvConfig(overrides)
	}

	t.Run("required verification rejects conflicting settings", func(t *testing.T) {
		for _, setting := range []struct{ name, value string }{
			{"EMAILS_VERIFIED", "true"},
			{"REQUIRE_USER_EMAIL", "false"},
			{"SMTP_HOST", ""},
			{"SMTP_FROM", ""},
		} {
			t.Run(setting.name, func(t *testing.T) {
				setUIConfigDisabled(t, true)
				t.Setenv("EMAIL_VERIFICATION", "required")
				t.Setenv("EMAILS_VERIFIED", "false")
				t.Setenv("REQUIRE_USER_EMAIL", "true")
				t.Setenv("SMTP_HOST", "smtp.example.com")
				t.Setenv("SMTP_FROM", "pocket-id@example.com")
				t.Setenv(setting.name, setting.value)

				_, err := loadEnvConfig()
				require.ErrorContains(t, err, "EMAIL_VERIFICATION requires user emails, an SMTP host and sender address, and forbids emails verified by default")
			})
		}
	})

	t.Run("required verification accepts consistent settings", func(t *testing.T) {
		t.Setenv("EMAIL_VERIFICATION", "required")
		t.Setenv("SMTP_HOST", "smtp.example.com")
		t.Setenv("SMTP_FROM", "pocket-id@example.com")

		cfg, err := loadEnvConfig()
		require.NoError(t, err)
		assert.Equal(t, AppConfigValue(EmailVerificationRequired), cfg.EmailVerification)
	})

	// writeSecretFile writes content to a file in a temporary directory that is removed when the test ends, returning its path
	writeSecretFile := func(t *testing.T, content string) string {
		t.Helper()

		fileName := filepath.Join(t.TempDir(), "secret.txt")
		err := os.WriteFile(fileName, []byte(content), 0600)
		require.NoError(t, err)

		return fileName
	}

	t.Run("loads a sensitive value from a file", func(t *testing.T) {
		t.Setenv("LDAP_BIND_PASSWORD_FILE", writeSecretFile(t, "my-ldap-password"))

		cfg, err := loadEnvConfig()
		require.NoError(t, err)

		assert.Equal(t, AppConfigValue("my-ldap-password"), cfg.LdapBindPassword)
	})

	t.Run("trims whitespace from values loaded from a file", func(t *testing.T) {
		// Files containing secrets normally end with a trailing newline, which must not be part of the value
		t.Setenv("LDAP_BIND_PASSWORD_FILE", writeSecretFile(t, "my-ldap-password\n"))
		t.Setenv("SMTP_PASSWORD_FILE", writeSecretFile(t, "  my-smtp-password  \r\n"))

		cfg, err := loadEnvConfig()
		require.NoError(t, err)

		assert.Equal(t, AppConfigValue("my-ldap-password"), cfg.LdapBindPassword)
		assert.Equal(t, AppConfigValue("my-smtp-password"), cfg.SmtpPassword)
	})

	t.Run("the file takes precedence over the environment variable", func(t *testing.T) {
		t.Setenv("LDAP_BIND_PASSWORD", "from-env")
		t.Setenv("LDAP_BIND_PASSWORD_FILE", writeSecretFile(t, "from-file\n"))

		cfg, err := loadEnvConfig()
		require.NoError(t, err)

		assert.Equal(t, AppConfigValue("from-file"), cfg.LdapBindPassword)
	})

	t.Run("values that are not sensitive are not loaded from a file", func(t *testing.T) {
		t.Setenv("LDAP_BIND_DN_FILE", writeSecretFile(t, "cn=from-file"))

		cfg, err := loadEnvConfig()
		require.NoError(t, err)

		assert.Empty(t, cfg.LdapBindDn)
	})

	t.Run("returns an error when the file cannot be read", func(t *testing.T) {
		t.Setenv("LDAP_BIND_PASSWORD_FILE", filepath.Join(t.TempDir(), "does-not-exist.txt"))

		_, err := loadEnvConfig()
		require.Error(t, err)
		assert.ErrorContains(t, err, "LDAP_BIND_PASSWORD_FILE")
	})
}
