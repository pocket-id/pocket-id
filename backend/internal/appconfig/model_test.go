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
		m.Replace(input)

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
		m.Replace(input)

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

		m.Replace(dto.AppConfigUpdateDto{})

		assert.Equal(t, *getDefaultConfig(), m)
	})

	t.Run("provided values overwrite existing non-default values", func(t *testing.T) {
		m := getDefaultConfig()
		m.AppName = "Old Name"
		m.LdapEnabled = "true"

		input := dto.AppConfigUpdateDto{}
		input.AppName = "New Name"

		m.Replace(input)

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
		m.Replace(input)

		assert.Equal(t, "120", m.SessionDuration)
		assert.Equal(t, "true", m.LdapEnabled)
	})
}

func TestAppConfigModel_Clone(t *testing.T) {
	t.Run("clones every property", func(t *testing.T) {
		// Populate every property with a unique marker so we can assert each one is copied
		var original AppConfigModel
		rv := reflect.ValueOf(&original).Elem()
		rt := rv.Type()
		for i := range rt.NumField() {
			key, _, _ := strings.Cut(rt.Field(i).Tag.Get("json"), ",")
			rv.Field(i).SetString("marker-" + key)
		}

		clone := original.Clone()

		require.NotNil(t, clone)
		// The clone must be a distinct object with equal contents
		assert.NotSame(t, &original, clone)
		assert.Equal(t, original, *clone)
	})

	t.Run("mutating the clone does not affect the original", func(t *testing.T) {
		original := getDefaultConfig()

		clone := original.Clone()
		clone.AppName = "Changed"
		clone.LdapEnabled = "true"

		// The original keeps its values
		assert.Equal(t, getDefaultConfig().AppName, original.AppName)
		assert.Equal(t, getDefaultConfig().LdapEnabled, original.LdapEnabled)

		// The clone holds the new values
		assert.Equal(t, AppConfigValue("Changed"), clone.AppName)
		assert.Equal(t, AppConfigValue("true"), clone.LdapEnabled)
	})

	t.Run("mutating the original does not affect the clone", func(t *testing.T) {
		original := getDefaultConfig()

		clone := original.Clone()
		original.AppName = "Changed"

		assert.Equal(t, getDefaultConfig().AppName, clone.AppName)
	})

	t.Run("cloning a nil receiver returns nil", func(t *testing.T) {
		var m *AppConfigModel
		assert.Nil(t, m.Clone())
	})
}

func TestAppConfigModel_Update(t *testing.T) {
	t.Run("updates a single property", func(t *testing.T) {
		m := getDefaultConfig()

		err := m.Update(map[string]string{"appName": "My App"})
		require.NoError(t, err)

		assert.Equal(t, "My App", m.AppName)
	})

	t.Run("updates multiple properties and leaves others untouched", func(t *testing.T) {
		m := getDefaultConfig()

		err := m.Update(map[string]string{"appName": "My App", "homePageUrl": "/home", "ldapEnabled": "true"})
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

		err := m.Update(map[string]string{"smtpTls": "", "sessionDuration": ""})
		require.NoError(t, err)

		assert.Equal(t, getDefaultConfig().SmtpTls, m.SmtpTls)
		assert.Equal(t, getDefaultConfig().SessionDuration, m.SessionDuration)
	})

	t.Run("stores raw string values without type coercion", func(t *testing.T) {
		m := getDefaultConfig()

		err := m.Update(map[string]string{"sessionDuration": "120", "disableAnimations": "true"})
		require.NoError(t, err)

		assert.Equal(t, "120", m.SessionDuration)
		assert.Equal(t, "true", m.DisableAnimations)
	})

	t.Run("an empty map is a no-op", func(t *testing.T) {
		m := getDefaultConfig()
		before := *m

		err := m.Update(nil)
		require.NoError(t, err)

		assert.Equal(t, before, *m)
	})

	t.Run("an unknown key returns AppConfigKeyNotFoundError", func(t *testing.T) {
		m := getDefaultConfig()

		err := m.Update(map[string]string{"thisKeyDoesNotExist": "value"})
		require.Error(t, err)
		assert.EqualError(t, err, "cannot find config key 'thisKeyDoesNotExist'")

		notFound, ok := errors.AsType[AppConfigKeyNotFoundError](err)
		require.True(t, ok)
		assert.Equal(t, "thisKeyDoesNotExist", notFound.field)
	})
}
