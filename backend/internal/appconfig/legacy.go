package appconfig

import "encoding/json"

// UnmarshalJSON preserves settings stored in actor state before boolean toggles were replaced by modes
func (m *AppConfigModel) UnmarshalJSON(data []byte) error {
	type config AppConfigModel
	value := struct {
		*config
		LegacyNotificationEnabled      AppConfigValue `json:"emailLoginNotificationEnabled"`
		LegacyEmailVerificationEnabled AppConfigValue `json:"emailVerificationEnabled"`
	}{config: (*config)(m)}
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	if m.EmailLoginNotificationMode == "" && value.LegacyNotificationEnabled != "" {
		m.EmailLoginNotificationMode = legacyLoginNotificationMode(value.LegacyNotificationEnabled)
	}
	if m.EmailVerification == "" && value.LegacyEmailVerificationEnabled != "" {
		m.EmailVerification = legacyEmailVerification(value.LegacyEmailVerificationEnabled)
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
