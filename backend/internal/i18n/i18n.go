// Package i18n knows the languages Pocket ID is translated into
package i18n

import (
	"golang.org/x/text/language"
)

// locales are the languages of the UI, as the frontend stores them for users, and must match the locales in frontend/project.inlang/settings.json
var locales = []string{"ca", "cs", "da", "de", "en", "es", "et", "fi", "fr", "hu", "it", "ja", "ko", "lv", "nl", "no", "pl", "pt", "pt-BR", "ru", "sv", "tr", "uk", "vi", "zh-CN", "zh-TW"}

var matcher = newMatcher()

func newMatcher() language.Matcher {
	tags := make([]language.Tag, len(locales))
	for i, locale := range locales {
		tags[i] = language.MustParse(locale)

		// Pocket ID stores Norwegian Bokmål as "no", but the matcher sends "nb" to Danish unless Bokmål is listed under its own code
		if locale == "no" {
			tags[i] = language.MustParse("nb")
		}
	}
	return language.NewMatcher(tags)
}

// Match returns the language closest to a preference in the Accept-Language format, such as "de-CH" or "fr, en;q=0.8", or nil if none is close
func Match(preference string) *string {
	tags, _, err := language.ParseAcceptLanguage(preference)
	if err != nil || len(tags) == 0 {
		return nil
	}

	_, index, confidence := matcher.Match(tags...)
	if confidence == language.No {
		return nil
	}
	return &locales[index]
}
