package appconfig

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pocket-id/pocket-id/backend/internal/dto"
)

// dtoWithMarkerValues returns a DTO where every field is set to a unique, non-empty marker derived from its "json" key, so we can assert each value lands in the right place.
func dtoWithMarkerValues() dto.AppConfigUpdateDto {
	var input dto.AppConfigUpdateDto
	rv := reflect.ValueOf(&input).Elem()
	rt := rv.Type()
	for i := range rt.NumField() {
		key, _, _ := strings.Cut(rt.Field(i).Tag.Get("json"), ",")
		rv.Field(i).SetString("marker-" + key)
	}
	return input
}

func TestAppConfigModel_Replace(t *testing.T) {
	t.Run("populates every property from the DTO", func(t *testing.T) {
		input := dtoWithMarkerValues()

		var m AppConfigModel
		err := m.Replace(input)
		require.NoError(t, err)

		// Each model property must hold the marker built from its own "json" key.
		// This also asserts that the model and the DTO share the same set of keys.
		rv := reflect.ValueOf(&m).Elem()
		rt := rv.Type()
		for i := range rt.NumField() {
			key, _, _ := strings.Cut(rt.Field(i).Tag.Get("json"), ",")
			assert.Equalf(t, "marker-"+key, rv.Field(i).String(), "property %s (key %q)", rt.Field(i).Name, key)
		}
	})

	t.Run("empty values fall back to their default", func(t *testing.T) {
		defaults := getDefaultConfig()

		// Start from all-markers, then blank out a few properties whose default is non-empty
		input := dtoWithMarkerValues()
		input.AppName = ""
		input.SessionDuration = ""
		input.SmtpTls = ""
		input.LdapUserSearchFilter = ""

		var m AppConfigModel
		err := m.Replace(input)
		require.NoError(t, err)

		// Blanked properties are reset to their default
		assert.Equal(t, defaults.AppName, m.AppName)
		assert.Equal(t, defaults.SessionDuration, m.SessionDuration)
		assert.Equal(t, defaults.SmtpTls, m.SmtpTls)
		assert.Equal(t, defaults.LdapUserSearchFilter, m.LdapUserSearchFilter)

		// A property that was provided keeps the provided value
		assert.Equal(t, "marker-homePageUrl", m.HomePageURL)
	})

	t.Run("an empty DTO resets every property to its default", func(t *testing.T) {
		// Pre-populate with junk to prove Replace overwrites existing state
		m := AppConfigModel{
			AppName:     "Custom Name",
			LdapEnabled: "true",
			SmtpHost:    "smtp.example.com",
		}

		err := m.Replace(dto.AppConfigUpdateDto{})
		require.NoError(t, err)

		assert.Equal(t, *getDefaultConfig(), m)
	})

	t.Run("provided values overwrite existing non-default values", func(t *testing.T) {
		m := getDefaultConfig()
		m.AppName = "Old Name"
		m.LdapEnabled = "true"

		input := dto.AppConfigUpdateDto{}
		input.AppName = "New Name"

		err := m.Replace(input)
		require.NoError(t, err)

		// Explicitly provided value wins
		assert.Equal(t, "New Name", m.AppName)
		// Everything else in the DTO was empty, so it is reset to the default
		assert.Equal(t, getDefaultConfig().LdapEnabled, m.LdapEnabled)
	})

	t.Run("stores raw string values without type coercion", func(t *testing.T) {
		input := dto.AppConfigUpdateDto{}
		input.SessionDuration = "120" // int-tagged property
		input.LdapEnabled = "true"    // bool-tagged property

		var m AppConfigModel
		err := m.Replace(input)
		require.NoError(t, err)

		assert.Equal(t, "120", m.SessionDuration)
		assert.Equal(t, "true", m.LdapEnabled)
	})
}

func TestAppConfigModel_Update(t *testing.T) {
	t.Run("updates a single property", func(t *testing.T) {
		m := getDefaultConfig()

		err := m.Update("appName", "My App")
		require.NoError(t, err)

		assert.Equal(t, "My App", m.AppName)
	})

	t.Run("updates multiple properties and leaves others untouched", func(t *testing.T) {
		m := getDefaultConfig()

		err := m.Update("appName", "My App", "homePageUrl", "/home", "ldapEnabled", "true")
		require.NoError(t, err)

		assert.Equal(t, "My App", m.AppName)
		assert.Equal(t, "/home", m.HomePageURL)
		assert.Equal(t, "true", m.LdapEnabled)
		// A property that was not part of the update keeps its previous value
		assert.Equal(t, getDefaultConfig().SessionDuration, m.SessionDuration)
	})

	t.Run("an empty value resets the property to its default", func(t *testing.T) {
		m := getDefaultConfig()
		m.SmtpTls = "tls"         // default is "none"
		m.SessionDuration = "120" // default is "60"

		err := m.Update("smtpTls", "", "sessionDuration", "")
		require.NoError(t, err)

		assert.Equal(t, getDefaultConfig().SmtpTls, m.SmtpTls)
		assert.Equal(t, getDefaultConfig().SessionDuration, m.SessionDuration)
	})

	t.Run("stores raw string values without type coercion", func(t *testing.T) {
		m := getDefaultConfig()

		err := m.Update("sessionDuration", "120", "disableAnimations", "true")
		require.NoError(t, err)

		assert.Equal(t, "120", m.SessionDuration)
		assert.Equal(t, "true", m.DisableAnimations)
	})

	t.Run("later value wins for a repeated key", func(t *testing.T) {
		m := getDefaultConfig()

		err := m.Update("appName", "First", "appName", "Second")
		require.NoError(t, err)

		assert.Equal(t, "Second", m.AppName)
	})

	t.Run("no arguments is a no-op", func(t *testing.T) {
		m := getDefaultConfig()
		before := *m

		err := m.Update()
		require.NoError(t, err)

		assert.Equal(t, before, *m)
	})

	t.Run("an odd number of arguments returns an error", func(t *testing.T) {
		m := getDefaultConfig()
		before := *m

		err := m.Update("appName")
		require.Error(t, err)
		assert.EqualError(t, err, "invalid number of arguments received")

		// The config must not have been modified
		assert.Equal(t, before, *m)
	})

	t.Run("an unknown key returns AppConfigKeyNotFoundError", func(t *testing.T) {
		m := getDefaultConfig()

		err := m.Update("thisKeyDoesNotExist", "value")
		require.Error(t, err)
		assert.EqualError(t, err, "cannot find config key 'thisKeyDoesNotExist'")

		notFound, ok := errors.AsType[AppConfigKeyNotFoundError](err)
		require.True(t, ok)
		assert.Equal(t, "thisKeyDoesNotExist", notFound.field)
	})
}
