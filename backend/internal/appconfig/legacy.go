package appconfig

import (
	"encoding/json"

	"github.com/vmihailenco/msgpack/v5"
)

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
	m.applyLegacyToggles(value.LegacyNotificationEnabled, value.LegacyEmailVerificationEnabled)
	return nil
}

// DecodeMsgpack preserves the same settings when the actor state is read, which is stored as MessagePack keyed by field name
func (m *AppConfigModel) DecodeMsgpack(dec *msgpack.Decoder) error {
	type config AppConfigModel
	value := struct {
		*config
		LegacyNotificationEnabled      AppConfigValue `msgpack:"EmailLoginNotificationEnabled"`
		LegacyEmailVerificationEnabled AppConfigValue `msgpack:"EmailVerificationEnabled"`
	}{config: (*config)(m)}
	if err := dec.Decode(&value); err != nil {
		return err
	}
	m.applyLegacyToggles(value.LegacyNotificationEnabled, value.LegacyEmailVerificationEnabled)
	return nil
}

func (m *AppConfigModel) applyLegacyToggles(notificationEnabled, emailVerificationEnabled AppConfigValue) {
	if m.EmailLoginNotificationMode == "" && notificationEnabled != "" {
		m.EmailLoginNotificationMode = legacyLoginNotificationMode(notificationEnabled)
	}
	if m.EmailVerification == "" && emailVerificationEnabled != "" {
		m.EmailVerification = legacyEmailVerification(emailVerificationEnabled)
	}
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
