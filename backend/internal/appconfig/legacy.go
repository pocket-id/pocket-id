package appconfig

import "github.com/vmihailenco/msgpack/v5"

// DecodeMsgpack preserves settings stored in actor state before boolean toggles were replaced by modes
// Francis stores actor state as MessagePack keyed by Go field name, so the legacy keys are the old field names
func (m *AppConfigModel) DecodeMsgpack(dec *msgpack.Decoder) error {
	type config AppConfigModel
	value := struct {
		*config
		LegacyNotificationEnabled      AppConfigValue `msgpack:"EmailLoginNotificationEnabled"`
		LegacyEmailVerificationEnabled AppConfigValue `msgpack:"EmailVerificationEnabled"`
		LegacyAccentColor              AppConfigValue `msgpack:"AccentColor"`
	}{config: (*config)(m)}
	if err := dec.Decode(&value); err != nil {
		return err
	}
	if m.EmailLoginNotificationMode == "" && value.LegacyNotificationEnabled != "" {
		m.EmailLoginNotificationMode = legacyLoginNotificationMode(value.LegacyNotificationEnabled)
	}
	if m.EmailVerification == "" && value.LegacyEmailVerificationEnabled != "" {
		m.EmailVerification = legacyEmailVerification(value.LegacyEmailVerificationEnabled)
	}
	if m.ColorTheme == "" && value.LegacyAccentColor != "" {
		m.ColorTheme = legacyColorTheme(value.LegacyAccentColor)
	}
	return nil
}

func legacyLoginNotificationMode(enabled AppConfigValue) AppConfigValue {
	if enabled.IsTrue() {
		return LoginNotificationBrowserRecognition
	}
	return LoginNotificationDisabled
}

func legacyEmailVerification(enabled AppConfigValue) AppConfigValue {
	if enabled.IsTrue() {
		return EmailVerificationOptional
	}
	return EmailVerificationDisabled
}

// legacyColorTheme maps an accent color from before color themes to the theme built around it, or to a custom theme with that color
func legacyColorTheme(accent AppConfigValue) AppConfigValue {
	switch accent {
	case "", "default":
		return "mono"
	case "oklch(0.6 0.2 240)":
		return "mist"
	case "oklch(0.65 0.2 150)":
		return "sage"
	case "oklch(0.6 0.15 180)":
		return "midnight"
	case "oklch(0.6 0.24 300)":
		return "lavender"
	default:
		return "custom:" + accent
	}
}
