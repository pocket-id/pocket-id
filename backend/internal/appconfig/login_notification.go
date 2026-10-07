package appconfig

import "encoding/json"

const (
	LoginNotificationDisabled           = "disabled"
	LoginNotificationAlways             = "always"
	LoginNotificationIPAndUserAgent     = "ipAndUserAgent"
	LoginNotificationBrowserRecognition = "browserRecognition"
)

// UnmarshalJSON preserves the notification policy when loading state written before notification modes existed
func (m *AppConfigModel) UnmarshalJSON(data []byte) error {
	type config AppConfigModel
	value := struct {
		*config
		LegacyNotificationEnabled AppConfigValue `json:"emailLoginNotificationEnabled"`
	}{config: (*config)(m)}
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	if m.EmailLoginNotificationMode == "" && value.LegacyNotificationEnabled != "" {
		m.EmailLoginNotificationMode = legacyLoginNotificationMode(value.LegacyNotificationEnabled)
	}
	return nil
}

func legacyLoginNotificationMode(enabled AppConfigValue) AppConfigValue {
	if enabled.IsTrue() {
		return LoginNotificationBrowserRecognition
	}
	return LoginNotificationDisabled
}
