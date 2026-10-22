package appconfig

import "github.com/vmihailenco/msgpack/v5"

// DecodeMsgpack preserves settings stored in actor state before boolean toggles were replaced by modes
// Francis stores actor state as MessagePack keyed by Go field name, so the legacy keys are the old field names
func (m *AppConfigModel) DecodeMsgpack(dec *msgpack.Decoder) error {
	type config AppConfigModel
	value := struct {
		*config
		LegacyNotificationEnabled                        AppConfigValue `msgpack:"EmailLoginNotificationEnabled"`
		LegacyEmailVerificationEnabled                   AppConfigValue `msgpack:"EmailVerificationEnabled"`
		LegacyEmailOneTimeAccessAsUnauthenticatedEnabled AppConfigValue `msgpack:"EmailOneTimeAccessAsUnauthenticatedEnabled"`
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
	if m.EmailOneTimeAccessAsUnauthenticated == "" && value.LegacyEmailOneTimeAccessAsUnauthenticatedEnabled != "" {
		m.EmailOneTimeAccessAsUnauthenticated = legacyEmailOneTimeAccessAsUnauthenticated(value.LegacyEmailOneTimeAccessAsUnauthenticatedEnabled)
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

// legacyEmailOneTimeAccessAsUnauthenticated maps the old toggle to the mode that keeps its behavior, which allowed admins as well
func legacyEmailOneTimeAccessAsUnauthenticated(enabled AppConfigValue) AppConfigValue {
	if enabled.IsTrue() {
		return EmailOneTimeAccessAsUnauthenticatedAll
	}
	return EmailOneTimeAccessAsUnauthenticatedDisabled
}
