package email

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/go-playground/locales"
	"github.com/go-playground/locales/ca"
	"github.com/go-playground/locales/cs"
	"github.com/go-playground/locales/da"
	"github.com/go-playground/locales/de"
	"github.com/go-playground/locales/en"
	"github.com/go-playground/locales/es"
	"github.com/go-playground/locales/et"
	"github.com/go-playground/locales/fi"
	"github.com/go-playground/locales/fr"
	"github.com/go-playground/locales/hu"
	"github.com/go-playground/locales/it"
	"github.com/go-playground/locales/ja"
	"github.com/go-playground/locales/ko"
	"github.com/go-playground/locales/lv"
	"github.com/go-playground/locales/nb"
	"github.com/go-playground/locales/nl"
	"github.com/go-playground/locales/pl"
	"github.com/go-playground/locales/pt"
	"github.com/go-playground/locales/pt_BR"
	"github.com/go-playground/locales/ru"
	"github.com/go-playground/locales/sv"
	"github.com/go-playground/locales/tr"
	"github.com/go-playground/locales/uk"
	"github.com/go-playground/locales/vi"
	"github.com/go-playground/locales/zh_Hans_CN"
	"github.com/go-playground/locales/zh_Hant_TW"
)

// baseLocale is the language the message keys are authored in and the one every other language falls back to
const baseLocale = "en"

// dateLocales lists the languages emails can be sent in, keyed by the locale codes the frontend stores for users
// A message catalog for a language that is missing here is not used
var dateLocales = map[string]locales.Translator{
	"ca":    ca.New(),
	"cs":    cs.New(),
	"da":    da.New(),
	"de":    de.New(),
	"en":    en.New(),
	"es":    es.New(),
	"et":    et.New(),
	"fi":    fi.New(),
	"fr":    fr.New(),
	"hu":    hu.New(),
	"it":    it.New(),
	"ja":    ja.New(),
	"ko":    ko.New(),
	"lv":    lv.New(),
	"nb":    nb.New(),
	"nl":    nl.New(),
	"pl":    pl.New(),
	"pt":    pt.New(),
	"pt-BR": pt_BR.New(),
	"ru":    ru.New(),
	"sv":    sv.New(),
	"tr":    tr.New(),
	"uk":    uk.New(),
	"vi":    vi.New(),
	"zh-CN": zh_Hans_CN.New(),
	"zh-TW": zh_Hant_TW.New(),
}

// linkTags mark the part of a message that the template renders as a link
var linkTags = regexp.MustCompile(`\{#link\}|\{/link\}`)

// localizer renders the messages and values of an email in one language
type localizer struct {
	Locale   string
	messages map[string]string
	dates    locales.Translator
}

// loadLocalizers reads every message catalog in dir, keyed by locale
func loadLocalizers(fsys fs.FS, dir string) (map[string]*localizer, error) {
	files, err := fs.Glob(fsys, path.Join(dir, "*.json"))
	if err != nil {
		return nil, err
	}

	catalogs := make(map[string]map[string]string, len(files))
	for _, file := range files {
		data, err := fs.ReadFile(fsys, file)
		if err != nil {
			return nil, err
		}

		var messages map[string]string
		err = json.Unmarshal(data, &messages)
		if err != nil {
			return nil, fmt.Errorf("parse email messages '%s': %w", file, err)
		}
		catalogs[strings.TrimSuffix(path.Base(file), ".json")] = messages
	}

	base, ok := catalogs[baseLocale]
	if !ok {
		return nil, fmt.Errorf("missing email messages for '%s'", baseLocale)
	}

	localizers := make(map[string]*localizer, len(catalogs))
	for locale, messages := range catalogs {
		dates, ok := dateLocales[locale]
		if !ok {
			continue
		}

		// Fill each message that is untranslated or whose link tags don't match the base message, so templates can rely on every key and link part
		for key, baseMessage := range base {
			message, ok := messages[key]
			if !ok || len(linkTags.Split(message, -1)) != len(linkTags.Split(baseMessage, -1)) {
				messages[key] = baseMessage
			}
		}

		localizers[locale] = &localizer{Locale: locale, messages: messages, dates: dates}
	}

	return localizers, nil
}

// T returns the message for key with each {name} replaced by the value that follows that name in args
func (l *localizer) T(key string, args ...string) (string, error) {
	message, ok := l.messages[key]
	if !ok {
		return "", fmt.Errorf("unknown email message '%s'", key)
	}

	replacements := make([]string, 0, len(args))
	for i := 0; i+1 < len(args); i += 2 {
		replacements = append(replacements, "{"+args[i]+"}", args[i+1])
	}
	return strings.NewReplacer(replacements...).Replace(message), nil
}

// Part returns the text before (0), inside (1) or after (2) the link of a message
func (l *localizer) Part(index int, key string, args ...string) (string, error) {
	message, err := l.T(key, args...)
	if err != nil {
		return "", err
	}
	return linkTags.Split(message, -1)[index], nil
}

// DateTime formats t in UTC, as the recipient's time zone is unknown
func (l *localizer) DateTime(t time.Time) (string, error) {
	t = t.UTC()
	return l.T("date_time", "date", l.dates.FmtDateLong(t), "time", l.dates.FmtTimeShort(t))
}

// Duration describes d with its two largest units, in minutes and hours below a day and in days and hours from a day on
func (l *localizer) Duration(d time.Duration) (string, error) {
	// Pick the two units to show
	largeUnit, smallUnit := "hour", "minute"
	large, small := int(d.Hours()), int(d.Minutes())%60
	if d >= 24*time.Hour {
		largeUnit, smallUnit = "day", "hour"
		large, small = int(d.Hours()/24), int(d.Hours())%24
	}

	// Leave out a unit that is zero
	if large == 0 {
		return l.count(smallUnit, small)
	}
	if small == 0 {
		return l.count(largeUnit, large)
	}

	first, err := l.count(largeUnit, large)
	if err != nil {
		return "", err
	}
	second, err := l.count(smallUnit, small)
	if err != nil {
		return "", err
	}
	return l.T("duration_pair", "first", first, "second", second)
}

func (l *localizer) count(unit string, n int) (string, error) {
	if n == 1 {
		return l.T("duration_" + unit)
	}
	return l.T("duration_"+unit+"s", "count", strconv.Itoa(n))
}
