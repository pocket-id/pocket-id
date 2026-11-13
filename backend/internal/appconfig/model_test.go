package appconfig

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vmihailenco/msgpack/v5"

	"github.com/pocket-id/pocket-id/backend/internal/dto"
)

func TestAppConfigModel_FieldTags(t *testing.T) {
	modelType := reflect.TypeFor[AppConfigModel]()
	jsonNames := make(map[string]string, modelType.NumField())
	envNames := make(map[string]string, modelType.NumField())
	for i := range modelType.NumField() {
		field := modelType.Field(i)
		t.Run(field.Name, func(t *testing.T) {
			jsonTag, ok := field.Tag.Lookup("json")
			require.True(t, ok)
			jsonName, _, _ := strings.Cut(jsonTag, ",")
			require.NotEmpty(t, jsonName)
			require.NotEqual(t, "-", jsonName)
			require.NotContains(t, jsonNames, jsonName)
			jsonNames[jsonName] = field.Name

			envName, ok := field.Tag.Lookup("env")
			require.True(t, ok)
			require.NotEmpty(t, envName)
			require.NotContains(t, envNames, envName)
			envNames[envName] = field.Name
		})
	}
}

func TestAppConfigModel_TypeTagsHaveDTOValidation(t *testing.T) {
	validatorsByType := map[string]string{
		"bool": "boolean_string",
		"int":  "integer_string",
	}

	dtoType := reflect.TypeFor[dto.AppConfigUpdateDto]()
	dtoBindings := make(map[string][]string, dtoType.NumField())
	for i := range dtoType.NumField() {
		field := dtoType.Field(i)
		jsonName, _, _ := strings.Cut(field.Tag.Get("json"), ",")
		dtoBindings[jsonName] = strings.Split(field.Tag.Get("binding"), ",")
	}

	modelType := reflect.TypeFor[AppConfigModel]()
	for i := range modelType.NumField() {
		field := modelType.Field(i)
		configType := field.Tag.Get("type")
		if configType == "" {
			continue
		}

		t.Run(field.Name, func(t *testing.T) {
			expectedValidator, ok := validatorsByType[configType]
			require.True(t, ok)

			jsonName, _, _ := strings.Cut(field.Tag.Get("json"), ",")
			bindings, ok := dtoBindings[jsonName]
			require.True(t, ok)
			require.Contains(t, bindings, expectedValidator)
		})
	}
}

func TestAppConfigModel_ApplyDefaults(t *testing.T) {
	m := &AppConfigModel{AppName: "Custom Name"}

	assert.True(t, m.applyDefaults())
	assert.Equal(t, AppConfigValue("Custom Name"), m.AppName)

	defaults := reflect.ValueOf(getDefaultConfig()).Elem()
	values := reflect.ValueOf(m).Elem()
	modelType := values.Type()
	for i := range values.NumField() {
		if modelType.Field(i).Name == "AppName" {
			continue
		}
		assert.Equal(t, defaults.Field(i).Interface(), values.Field(i).Interface(), modelType.Field(i).Name)
	}

	assert.False(t, m.applyDefaults())
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

		assert.Equal(t, AppConfigValue("My App"), m.AppName)
	})

	t.Run("updates multiple properties and leaves others untouched", func(t *testing.T) {
		m := getDefaultConfig()

		err := m.Update(map[string]string{"appName": "My App", "homePageUrl": "/home", "ldapEnabled": "true"})
		require.NoError(t, err)

		assert.Equal(t, AppConfigValue("My App"), m.AppName)
		assert.Equal(t, AppConfigValue("/home"), m.HomePageURL)
		assert.Equal(t, AppConfigValue("true"), m.LdapEnabled)
		// A property that was not part of the update keeps its previous value
		assert.Equal(t, getDefaultConfig().SessionDuration, m.SessionDuration)
	})

	t.Run("an empty value resets the property to its default", func(t *testing.T) {
		m := getDefaultConfig()
		m.SmtpTls = "tls"         // default is "none"
		m.SessionDuration = "120" // default is "1440"

		err := m.Update(map[string]string{"smtpTls": "", "sessionDuration": ""})
		require.NoError(t, err)

		assert.Equal(t, getDefaultConfig().SmtpTls, m.SmtpTls)
		assert.Equal(t, getDefaultConfig().SessionDuration, m.SessionDuration)
	})

	t.Run("stores raw string values without type coercion", func(t *testing.T) {
		m := getDefaultConfig()

		err := m.Update(map[string]string{"sessionDuration": "120", "allowOwnAccountEdit": "false"})
		require.NoError(t, err)

		assert.Equal(t, AppConfigValue("120"), m.SessionDuration)
		assert.Equal(t, AppConfigValue("false"), m.AllowOwnAccountEdit)
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
		require.EqualError(t, err, "cannot find config key 'thisKeyDoesNotExist'")

		notFound, ok := errors.AsType[AppConfigKeyNotFoundError](err)
		require.True(t, ok)
		assert.Equal(t, "thisKeyDoesNotExist", notFound.field)
	})
}

func TestAppConfigModel_DecodeMsgpack(t *testing.T) {
	t.Run("maps the legacy toggles to modes", func(t *testing.T) {
		tests := map[string]struct {
			notification      AppConfigValue
			emailVerification AppConfigValue
		}{
			"true":  {LoginNotificationBrowserRecognition, EmailVerificationOptional},
			"false": {LoginNotificationDisabled, EmailVerificationDisabled},
		}
		for enabled, expected := range tests {
			data, err := msgpack.Marshal(map[string]string{
				"AppName":                       "Legacy App",
				"EmailLoginNotificationEnabled": enabled,
				"EmailVerificationEnabled":      enabled,
			})
			require.NoError(t, err)

			var m AppConfigModel
			err = msgpack.Unmarshal(data, &m)
			require.NoError(t, err)

			assert.Equal(t, AppConfigValue("Legacy App"), m.AppName)
			assert.Equal(t, expected.notification, m.EmailLoginNotificationMode)
			assert.Equal(t, expected.emailVerification, m.EmailVerification)
		}
	})

	t.Run("stored modes win over the legacy toggles", func(t *testing.T) {
		data, err := msgpack.Marshal(map[string]string{
			"EmailLoginNotificationMode":    LoginNotificationAlways,
			"EmailLoginNotificationEnabled": "false",
			"EmailVerification":             EmailVerificationRequired,
			"EmailVerificationEnabled":      "false",
		})
		require.NoError(t, err)

		var m AppConfigModel
		err = msgpack.Unmarshal(data, &m)
		require.NoError(t, err)

		assert.Equal(t, AppConfigValue(LoginNotificationAlways), m.EmailLoginNotificationMode)
		assert.Equal(t, AppConfigValue(EmailVerificationRequired), m.EmailVerification)
	})
}
