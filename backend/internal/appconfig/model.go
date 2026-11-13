package appconfig

import (
	"errors"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/italypaleale/go-kit/utils"
)

const (
	LoginNotificationDisabled           = "disabled"
	LoginNotificationAlways             = "always"
	LoginNotificationIPAndUserAgent     = "ipAndUserAgent"
	LoginNotificationBrowserRecognition = "browserRecognition"
)

const (
	EmailVerificationDisabled = "disabled"
	EmailVerificationOptional = "optional"
	EmailVerificationRequired = "required"
)

const redactedValue = "XXXXXXXXXX"

type AppConfigModel struct {
	// General
	AppName             AppConfigValue `json:"appName" env:"APP_NAME" public:"true"`
	SessionDuration     AppConfigValue `json:"sessionDuration" env:"SESSION_DURATION" type:"int"` // In minutes
	HomePageURL         AppConfigValue `json:"homePageUrl" env:"HOME_PAGE_URL" public:"true"`
	EmailsVerified      AppConfigValue `json:"emailsVerified" env:"EMAILS_VERIFIED" type:"bool"`
	Appearance          AppConfigValue `json:"appearance" env:"APPEARANCE" public:"true"` // JSON-encoded object, the frontend fills in the fields it lacks
	AllowOwnAccountEdit AppConfigValue `json:"allowOwnAccountEdit" env:"ALLOW_OWN_ACCOUNT_EDIT" type:"bool" public:"true"`
	AllowUserSignups    AppConfigValue `json:"allowUserSignups" env:"ALLOW_USER_SIGNUPS" public:"true"`

	SignupDefaultUserGroupIDs AppConfigValue `json:"signupDefaultUserGroupIDs" env:"SIGNUP_DEFAULT_USER_GROUP_IDS"` // JSON-encoded array of strings
	SignupDefaultCustomClaims AppConfigValue `json:"signupDefaultCustomClaims" env:"SIGNUP_DEFAULT_CUSTOM_CLAIMS"`  // JSON-encoded array of {key:string,value:string}
	// Email
	RequireUserEmail                           AppConfigValue `json:"requireUserEmail" env:"REQUIRE_USER_EMAIL" type:"bool" public:"true"`
	SmtpHost                                   AppConfigValue `json:"smtpHost" env:"SMTP_HOST"`
	SmtpPort                                   AppConfigValue `json:"smtpPort" env:"SMTP_PORT"`
	SmtpFrom                                   AppConfigValue `json:"smtpFrom" env:"SMTP_FROM"`
	SmtpUser                                   AppConfigValue `json:"smtpUser" env:"SMTP_USER"`
	SmtpPassword                               AppConfigValue `json:"smtpPassword" env:"SMTP_PASSWORD" sensitive:"true"`
	SmtpTls                                    AppConfigValue `json:"smtpTls" env:"SMTP_TLS"`
	SmtpSkipCertVerify                         AppConfigValue `json:"smtpSkipCertVerify" env:"SMTP_SKIP_CERT_VERIFY" type:"bool"`
	EmailLoginNotificationMode                 AppConfigValue `json:"emailLoginNotificationMode" env:"EMAIL_LOGIN_NOTIFICATION_MODE"`
	EmailOneTimeAccessAsUnauthenticatedEnabled AppConfigValue `json:"emailOneTimeAccessAsUnauthenticatedEnabled" env:"EMAIL_ONE_TIME_ACCESS_AS_UNAUTHENTICATED_ENABLED" type:"bool" public:"true"`
	EmailOneTimeAccessAsAdminEnabled           AppConfigValue `json:"emailOneTimeAccessAsAdminEnabled" env:"EMAIL_ONE_TIME_ACCESS_AS_ADMIN_ENABLED" type:"bool" public:"true"`
	EmailApiKeyExpirationEnabled               AppConfigValue `json:"emailApiKeyExpirationEnabled" env:"EMAIL_API_KEY_EXPIRATION_ENABLED" type:"bool"`
	EmailVerification                          AppConfigValue `json:"emailVerification" env:"EMAIL_VERIFICATION" public:"true"`
	// LDAP
	LdapEnabled                        AppConfigValue `json:"ldapEnabled" env:"LDAP_ENABLED" type:"bool" public:"true"`
	LdapUrl                            AppConfigValue `json:"ldapUrl" env:"LDAP_URL"`
	LdapBindDn                         AppConfigValue `json:"ldapBindDn" env:"LDAP_BIND_DN"`
	LdapBindPassword                   AppConfigValue `json:"ldapBindPassword" env:"LDAP_BIND_PASSWORD" sensitive:"true"`
	LdapBase                           AppConfigValue `json:"ldapBase" env:"LDAP_BASE"`
	LdapUserSearchFilter               AppConfigValue `json:"ldapUserSearchFilter" env:"LDAP_USER_SEARCH_FILTER"`
	LdapUserGroupSearchFilter          AppConfigValue `json:"ldapUserGroupSearchFilter" env:"LDAP_USER_GROUP_SEARCH_FILTER"`
	LdapSkipCertVerify                 AppConfigValue `json:"ldapSkipCertVerify" env:"LDAP_SKIP_CERT_VERIFY" type:"bool"`
	LdapAttributeUserUniqueIdentifier  AppConfigValue `json:"ldapAttributeUserUniqueIdentifier" env:"LDAP_ATTRIBUTE_USER_UNIQUE_IDENTIFIER"`
	LdapAttributeUserUsername          AppConfigValue `json:"ldapAttributeUserUsername" env:"LDAP_ATTRIBUTE_USER_USERNAME"`
	LdapAttributeUserEmail             AppConfigValue `json:"ldapAttributeUserEmail" env:"LDAP_ATTRIBUTE_USER_EMAIL"`
	LdapAttributeUserFirstName         AppConfigValue `json:"ldapAttributeUserFirstName" env:"LDAP_ATTRIBUTE_USER_FIRST_NAME"`
	LdapAttributeUserLastName          AppConfigValue `json:"ldapAttributeUserLastName" env:"LDAP_ATTRIBUTE_USER_LAST_NAME"`
	LdapAttributeUserDisplayName       AppConfigValue `json:"ldapAttributeUserDisplayName" env:"LDAP_ATTRIBUTE_USER_DISPLAY_NAME"`
	LdapAttributeUserProfilePicture    AppConfigValue `json:"ldapAttributeUserProfilePicture" env:"LDAP_ATTRIBUTE_USER_PROFILE_PICTURE"`
	LdapAttributeGroupMember           AppConfigValue `json:"ldapAttributeGroupMember" env:"LDAP_ATTRIBUTE_GROUP_MEMBER"`
	LdapAttributeGroupUniqueIdentifier AppConfigValue `json:"ldapAttributeGroupUniqueIdentifier" env:"LDAP_ATTRIBUTE_GROUP_UNIQUE_IDENTIFIER"`
	LdapAttributeGroupName             AppConfigValue `json:"ldapAttributeGroupName" env:"LDAP_ATTRIBUTE_GROUP_NAME"`
	LdapAdminGroupName                 AppConfigValue `json:"ldapAdminGroupName" env:"LDAP_ADMIN_GROUP_NAME"`
	LdapSoftDeleteUsers                AppConfigValue `json:"ldapSoftDeleteUsers" env:"LDAP_SOFT_DELETE_USERS" type:"bool"`
	LdapAllowProfilePictureOverride    AppConfigValue `json:"ldapAllowProfilePictureOverride" env:"LDAP_ALLOW_PROFILE_PICTURE_OVERRIDE" type:"bool" public:"true"`
	// WebAuthn
	WebauthnUserVerification        AppConfigValue `json:"webauthnUserVerification" env:"WEBAUTHN_USER_VERIFICATION"`
	WebauthnAllowSyncedPasskeys     AppConfigValue `json:"webauthnAllowSyncedPasskeys" env:"WEBAUTHN_ALLOW_SYNCED_PASSKEYS" type:"bool"`
	WebauthnAuthenticatorAttachment AppConfigValue `json:"webauthnAuthenticatorAttachment" env:"WEBAUTHN_AUTHENTICATOR_ATTACHMENT"`
	WebauthnAttestation             AppConfigValue `json:"webauthnAttestation" env:"WEBAUTHN_ATTESTATION"`
	// OIDC
	CIMDURLAllowlist           AppConfigValue `json:"cimdUrlAllowlist" env:"CIMD_URL_ALLOWLIST"` // JSON-encoded array of strings
	AutoCreateOIDCClientSecret AppConfigValue `json:"autoCreateOidcClientSecret" env:"AUTO_CREATE_OIDC_CLIENT_SECRET" type:"bool"`
	// Features
	EnabledFeatures AppConfigValue `json:"enabledFeatures" env:"ENABLED_FEATURES"` // JSON-encoded array of feature ids
}

// appConfigEnvName returns the explicit environment variable name for a JSON configuration field
func appConfigEnvName(jsonName string) (string, bool) {
	modelType := reflect.TypeFor[AppConfigModel]()
	for i := range modelType.NumField() {
		field := modelType.Field(i)
		fieldJSONName, _, _ := strings.Cut(field.Tag.Get("json"), ",")
		if fieldJSONName == jsonName {
			envName := field.Tag.Get("env")
			return envName, envName != ""
		}
	}

	return "", false
}

// Clone returns a deep copy of the AppConfigModel.
func (m *AppConfigModel) Clone() *AppConfigModel {
	if m == nil {
		return nil
	}

	// All fields are value types (AppConfigValue is a string), so copying the struct is sufficient for a deep copy.
	clone := *m
	return &clone
}

// AppConfigValue holds a value
type AppConfigValue string

// IsTrue returns true if the value is a truthy string, such as "true", "t", "yes", "1", etc.
func (a AppConfigValue) IsTrue() bool {
	return utils.IsTruthy(string(a))
}

// AsDurationMinutes returns the value as a time.Duration, interpreting the string as a whole number of minutes.
func (a AppConfigValue) AsDurationMinutes() time.Duration {
	val, err := strconv.Atoi(string(a))
	if err != nil {
		return 0
	}
	return time.Duration(val) * time.Minute
}

// String implements fmt.Stringer
func (a AppConfigValue) String() string {
	return string(a)
}

func getDefaultConfig() *AppConfigModel {
	// Values are the default ones
	return &AppConfigModel{
		// General
		AppName:                   "Pocket ID",
		SessionDuration:           "1440",
		HomePageURL:               "/settings/account",
		EmailsVerified:            "false",
		AllowOwnAccountEdit:       "true",
		AllowUserSignups:          "disabled",
		SignupDefaultUserGroupIDs: "[]",
		SignupDefaultCustomClaims: "[]",
		Appearance:                "{}",
		// Email
		RequireUserEmail:           "true",
		SmtpHost:                   "",
		SmtpPort:                   "",
		SmtpFrom:                   "",
		SmtpUser:                   "",
		SmtpPassword:               "",
		SmtpTls:                    "none",
		SmtpSkipCertVerify:         "false",
		EmailLoginNotificationMode: LoginNotificationDisabled,
		EmailOneTimeAccessAsUnauthenticatedEnabled: "false",
		EmailOneTimeAccessAsAdminEnabled:           "false",
		EmailApiKeyExpirationEnabled:               "false",
		EmailVerification:                          EmailVerificationDisabled,
		// LDAP
		LdapEnabled:                        "false",
		LdapUrl:                            "",
		LdapBindDn:                         "",
		LdapBindPassword:                   "",
		LdapBase:                           "",
		LdapUserSearchFilter:               "(objectClass=person)",
		LdapUserGroupSearchFilter:          "(objectClass=groupOfNames)",
		LdapSkipCertVerify:                 "false",
		LdapAttributeUserUniqueIdentifier:  "",
		LdapAttributeUserUsername:          "",
		LdapAttributeUserEmail:             "",
		LdapAttributeUserFirstName:         "",
		LdapAttributeUserLastName:          "",
		LdapAttributeUserDisplayName:       "cn",
		LdapAttributeUserProfilePicture:    "",
		LdapAttributeGroupMember:           "member",
		LdapAttributeGroupUniqueIdentifier: "",
		LdapAttributeGroupName:             "",
		LdapAdminGroupName:                 "",
		LdapSoftDeleteUsers:                "true",
		LdapAllowProfilePictureOverride:    "false",
		// WebAuthn
		WebauthnUserVerification:        "required",
		WebauthnAllowSyncedPasskeys:     "true",
		WebauthnAuthenticatorAttachment: "any",
		WebauthnAttestation:             "none",
		// OIDC
		CIMDURLAllowlist:           "[]",
		AutoCreateOIDCClientSecret: "true",
		// Features
		// Left unset so the feature module can tell a saved value from one it still has to derive from usage
		EnabledFeatures: "",
	}
}

// applyDefaults fills empty properties from the default configuration and reports whether the model changed
func (m *AppConfigModel) applyDefaults() bool {
	defaults := reflect.ValueOf(getDefaultConfig()).Elem()
	values := reflect.ValueOf(m).Elem()
	changed := false

	for i := range values.NumField() {
		if values.Field(i).String() != "" || defaults.Field(i).String() == "" {
			continue
		}

		values.Field(i).Set(defaults.Field(i))
		changed = true
	}

	return changed
}

// Update sets configuration properties from the provided key-value pairs
// Keys correspond to the "json" tags on the model
// An empty string value resets the property to its default value
func (m *AppConfigModel) Update(values map[string]string) error {
	rv := reflect.ValueOf(m).Elem()
	rt := rv.Type()
	defaults := reflect.ValueOf(getDefaultConfig()).Elem()

	// Iterate through the key-value pairs
	for key, value := range values {
		// Find the field in the struct whose "json" tag matches
		fieldIdx := -1
		for j := range rt.NumField() {
			// Separate the key (before the comma) from any optional attributes after
			tagValue, _, _ := strings.Cut(rt.Field(j).Tag.Get("json"), ",")
			if tagValue == key {
				fieldIdx = j
				break
			}
		}
		if fieldIdx < 0 {
			return AppConfigKeyNotFoundError{field: key}
		}

		// An empty string means we use the default value for the property
		if value == "" {
			value = defaults.Field(fieldIdx).String()
		}

		rv.Field(fieldIdx).SetString(value)
	}

	return nil
}

// AppConfigVariable is a single application configuration property, as a key/value pair
type AppConfigVariable struct {
	Key         string
	Value       string
	EnvOverride bool
}

// ToAppConfigVariableSlice returns the configuration as a slice of key/value pairs for HTTP responses
// If showAll is false, only properties marked as public are included
// Properties in envOverrides are marked as set in the environment
// Sensitive values that are set are redacted, because credentials that may read the config must never read secrets back
func (m *AppConfigModel) ToAppConfigVariableSlice(showAll bool, envOverrides map[string]string) []AppConfigVariable {
	// Iterate through all fields
	cfgValue := reflect.ValueOf(m).Elem()
	cfgType := cfgValue.Type()

	res := make([]AppConfigVariable, 0, cfgType.NumField())
	for i := range cfgType.NumField() {
		field := cfgType.Field(i)

		key, _, _ := strings.Cut(field.Tag.Get("json"), ",")
		if key == "" {
			continue
		}

		// If we're only showing public variables and this is not public, skip it
		if !showAll && field.Tag.Get("public") != "true" {
			continue
		}

		value := cfgValue.Field(i).String()
		if value != "" && field.Tag.Get("sensitive") == "true" {
			value = redactedValue
		}
		_, envOverride := envOverrides[key]

		res = append(res, AppConfigVariable{
			Key:         key,
			Value:       value,
			EnvOverride: envOverride,
		})
	}

	return res
}

type AppConfigKeyNotFoundError struct {
	field string
}

func (e AppConfigKeyNotFoundError) Error() string {
	return "cannot find config key '" + e.field + "'"
}

func (e AppConfigKeyNotFoundError) Is(target error) bool {
	// Ignore the field property when checking if an error is of the type AppConfigKeyNotFoundError
	_, ok := errors.AsType[*AppConfigKeyNotFoundError](target)
	return ok
}
