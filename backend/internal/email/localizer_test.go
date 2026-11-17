//go:build unit

package email

import (
	"testing"
	"testing/fstest"
	"time"

	"github.com/pocket-id/pocket-id/backend/resources"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testCatalogs holds a partial German translation next to the real English messages
var testCatalogs = fstest.MapFS{
	"en.json": {Data: mustReadEnglishCatalog()},
	"de.json": {Data: []byte(`{
		"test_title": "Test-E-Mail",
		"date_time": "{date} um {time} UTC",
		"duration_pair": "{first} und {second}",
		"duration_hours": "{count} Stunden",
		"new_sign_in_review": "Unbekannt? Prüfe die Passkeys in deinen Kontoeinstellungen."
	}`)},
	"xx.json": {Data: []byte(`{"test_title": "Unsupported"}`)},
}

func TestLocalizerTranslatesAndFallsBack(t *testing.T) {
	localizers, err := loadLocalizers(testCatalogs, ".")
	require.NoError(t, err)

	// A catalog without date formats is not used
	require.NotContains(t, localizers, "xx")
	de := localizers["de"]

	title, err := de.T("test_title")
	require.NoError(t, err)
	assert.Equal(t, "Test-E-Mail", title)

	// Untranslated keys and translations that lost their link tags use the English message
	description, err := de.T("test_description")
	require.NoError(t, err)
	assert.Equal(t, "Your email setup is working correctly!", description)
	link, err := de.Part(1, "new_sign_in_review")
	require.NoError(t, err)
	assert.Equal(t, "account settings", link)

	_, err = de.T("missing_key")
	require.Error(t, err)

	dateTime, err := de.DateTime(time.Date(2030, time.January, 2, 16, 4, 0, 0, time.FixedZone("CET", 3600)))
	require.NoError(t, err)
	assert.Equal(t, "2. Januar 2030 um 15:04 UTC", dateTime)

	duration, err := de.Duration(26 * time.Hour)
	require.NoError(t, err)
	assert.Equal(t, "1 day und 2 Stunden", duration)
}

func TestLocalizerReplacesArguments(t *testing.T) {
	localizers, err := loadLocalizers(testCatalogs, ".")
	require.NoError(t, err)
	en := localizers[baseLocale]

	// An argument value is inserted as is, even when it looks like a placeholder itself
	subject, err := en.T("signup_invitation_subject", "appName", "{appName}")
	require.NoError(t, err)
	assert.Equal(t, "You're invited to {appName}", subject)

	before, err := en.Part(0, "one_time_access_manual", "loginLink", "https://id.example.test/lc")
	require.NoError(t, err)
	assert.Equal(t, "Or open ", before)
}

func TestLocalizerDuration(t *testing.T) {
	localizers, err := loadLocalizers(testCatalogs, ".")
	require.NoError(t, err)
	en := localizers[baseLocale]

	for duration, expected := range map[time.Duration]string{
		time.Minute:                     "1 minute",
		15 * time.Minute:                "15 minutes",
		time.Hour:                       "1 hour",
		time.Hour + 30*time.Minute:      "1 hour and 30 minutes",
		3*time.Hour + 5*time.Minute:     "3 hours and 5 minutes",
		24 * time.Hour:                  "1 day",
		25 * time.Hour:                  "1 day and 1 hour",
		3*24*time.Hour + 4*time.Hour:    "3 days and 4 hours",
		7*24*time.Hour + 30*time.Minute: "7 days",
	} {
		actual, err := en.Duration(duration)
		require.NoError(t, err)
		assert.Equal(t, expected, actual, duration.String())
	}
}

func mustReadEnglishCatalog() []byte {
	data, err := resources.FS.ReadFile("i18n/email/en.json")
	if err != nil {
		panic(err)
	}
	return data
}
