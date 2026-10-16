package appconfig

import (
	"encoding/json"

	msgpack "github.com/vmihailenco/msgpack/v5"
)

// legacyToggles holds the boolean toggles that previous versions stored before they were replaced by modes
// Actor state is msgpack-encoded and keyed by Go field name, while JSON uses the json tags, so each toggle carries both names
type legacyToggles struct {
	NotificationEnabled                        AppConfigValue `json:"emailLoginNotificationEnabled" msgpack:"EmailLoginNotificationEnabled"`
	EmailVerificationEnabled                   AppConfigValue `json:"emailVerificationEnabled" msgpack:"EmailVerificationEnabled"`
	EmailOneTimeAccessAsUnauthenticatedEnabled AppConfigValue `json:"emailOneTimeAccessAsUnauthenticatedEnabled" msgpack:"EmailOneTimeAccessAsUnauthenticatedEnabled"`
}

// apply maps each stored toggle to the mode that replaced it, unless the config already has that mode
func (l legacyToggles) apply(m *AppConfigModel) {
	if m.EmailLoginNotificationMode == "" && l.NotificationEnabled != "" {
		m.EmailLoginNotificationMode = legacyLoginNotificationMode(l.NotificationEnabled)
	}
	if m.EmailVerification == "" && l.EmailVerificationEnabled != "" {
		m.EmailVerification = legacyEmailVerification(l.EmailVerificationEnabled)
	}
	if m.EmailOneTimeAccessAsUnauthenticated == "" && l.EmailOneTimeAccessAsUnauthenticatedEnabled != "" {
		m.EmailOneTimeAccessAsUnauthenticated = legacyEmailOneTimeAccessAsUnauthenticated(l.EmailOneTimeAccessAsUnauthenticatedEnabled)
	}
}

// UnmarshalJSON preserves settings stored before boolean toggles were replaced by modes
func (m *AppConfigModel) UnmarshalJSON(data []byte) error {
	type config AppConfigModel
	value := struct {
		*config
		legacyToggles
	}{config: (*config)(m)}
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	value.apply(m)
	return nil
}

// DecodeMsgpack preserves settings stored in actor state before boolean toggles were replaced by modes
// Actor state is decoded with msgpack, which never calls UnmarshalJSON
func (m *AppConfigModel) DecodeMsgpack(dec *msgpack.Decoder) error {
	// msgpack cannot inline an embedded pointer, so decode into a value and copy it over
	type config AppConfigModel
	var value struct {
		config        `msgpack:",inline"`
		legacyToggles `msgpack:",inline"`
	}
	if err := dec.Decode(&value); err != nil {
		return err
	}
	*m = AppConfigModel(value.config)
	value.apply(m)
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

// legacyEmailOneTimeAccessAsUnauthenticated maps the old toggle to the mode that keeps its behavior, which allowed admins as well
func legacyEmailOneTimeAccessAsUnauthenticated(enabled AppConfigValue) AppConfigValue {
	if enabled.IsTrue() {
		return EmailOneTimeAccessAsUnauthenticatedAll
	}
	return EmailOneTimeAccessAsUnauthenticatedDisabled
}
