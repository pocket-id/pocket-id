package appconfig

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/italypaleale/go-kit/utils"

	"github.com/pocket-id/pocket-id/backend/internal/dto"
)

type AppConfigModel struct {
	// General
	AppName             AppConfigValue `json:"appName" public:"true"`
	SessionDuration     AppConfigValue `json:"sessionDuration" type:"int"` // In minutes
	HomePageURL         AppConfigValue `json:"homePageUrl" public:"true"`
	EmailsVerified      AppConfigValue `json:"emailsVerified" type:"bool"`
	AccentColor         AppConfigValue `json:"accentColor" public:"true"`
	DisableAnimations   AppConfigValue `json:"disableAnimations" type:"bool" public:"true"`
	AllowOwnAccountEdit AppConfigValue `json:"allowOwnAccountEdit" type:"bool" public:"true"`
	AllowUserSignups    AppConfigValue `json:"allowUserSignups" public:"true"`

	SignupDefaultUserGroupIDs AppConfigValue `json:"signupDefaultUserGroupIDs"` // JSON-encoded array of strings
	SignupDefaultCustomClaims AppConfigValue `json:"signupDefaultCustomClaims"` // JSON-encoded array of {key:string,value:string}
	// Email
	RequireUserEmail                           AppConfigValue `json:"requireUserEmail" type:"bool" public:"true"`
	SmtpHost                                   AppConfigValue `json:"smtpHost"`
	SmtpPort                                   AppConfigValue `json:"smtpPort"`
	SmtpFrom                                   AppConfigValue `json:"smtpFrom"`
	SmtpUser                                   AppConfigValue `json:"smtpUser"`
	SmtpPassword                               AppConfigValue `json:"smtpPassword" sensitive:"true"`
	SmtpTls                                    AppConfigValue `json:"smtpTls"`
	SmtpSkipCertVerify                         AppConfigValue `json:"smtpSkipCertVerify" type:"bool"`
	EmailLoginNotificationEnabled              AppConfigValue `json:"emailLoginNotificationEnabled" type:"bool"`
	EmailOneTimeAccessAsUnauthenticatedEnabled AppConfigValue `json:"emailOneTimeAccessAsUnauthenticatedEnabled" type:"bool" public:"true"`
	EmailOneTimeAccessAsAdminEnabled           AppConfigValue `json:"emailOneTimeAccessAsAdminEnabled" type:"bool" public:"true"`
	EmailApiKeyExpirationEnabled               AppConfigValue `json:"emailApiKeyExpirationEnabled" type:"bool"`
	EmailVerificationEnabled                   AppConfigValue `json:"emailVerificationEnabled" type:"bool" public:"true"`
	// LDAP
	LdapEnabled                        AppConfigValue `json:"ldapEnabled" type:"bool" public:"true"`
	LdapUrl                            AppConfigValue `json:"ldapUrl"`
	LdapBindDn                         AppConfigValue `json:"ldapBindDn"`
	LdapBindPassword                   AppConfigValue `json:"ldapBindPassword" sensitive:"true"`
	LdapBase                           AppConfigValue `json:"ldapBase"`
	LdapUserSearchFilter               AppConfigValue `json:"ldapUserSearchFilter"`
	LdapUserGroupSearchFilter          AppConfigValue `json:"ldapUserGroupSearchFilter"`
	LdapSkipCertVerify                 AppConfigValue `json:"ldapSkipCertVerify" type:"bool"`
	LdapAttributeUserUniqueIdentifier  AppConfigValue `json:"ldapAttributeUserUniqueIdentifier"`
	LdapAttributeUserUsername          AppConfigValue `json:"ldapAttributeUserUsername"`
	LdapAttributeUserEmail             AppConfigValue `json:"ldapAttributeUserEmail"`
	LdapAttributeUserFirstName         AppConfigValue `json:"ldapAttributeUserFirstName"`
	LdapAttributeUserLastName          AppConfigValue `json:"ldapAttributeUserLastName"`
	LdapAttributeUserDisplayName       AppConfigValue `json:"ldapAttributeUserDisplayName"`
	LdapAttributeUserProfilePicture    AppConfigValue `json:"ldapAttributeUserProfilePicture"`
	LdapAttributeGroupMember           AppConfigValue `json:"ldapAttributeGroupMember"`
	LdapAttributeGroupUniqueIdentifier AppConfigValue `json:"ldapAttributeGroupUniqueIdentifier"`
	LdapAttributeGroupName             AppConfigValue `json:"ldapAttributeGroupName"`
	LdapAdminGroupName                 AppConfigValue `json:"ldapAdminGroupName"`
	LdapSoftDeleteUsers                AppConfigValue `json:"ldapSoftDeleteUsers" type:"bool"`
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
		SessionDuration:           "60",
		HomePageURL:               "/settings/account",
		EmailsVerified:            "false",
		DisableAnimations:         "false",
		AllowOwnAccountEdit:       "true",
		AllowUserSignups:          "disabled",
		SignupDefaultUserGroupIDs: "[]",
		SignupDefaultCustomClaims: "[]",
		AccentColor:               "default",
		// Email
		RequireUserEmail:              "true",
		SmtpHost:                      "",
		SmtpPort:                      "",
		SmtpFrom:                      "",
		SmtpUser:                      "",
		SmtpPassword:                  "",
		SmtpTls:                       "none",
		SmtpSkipCertVerify:            "false",
		EmailLoginNotificationEnabled: "false",
		EmailOneTimeAccessAsUnauthenticatedEnabled: "false",
		EmailOneTimeAccessAsAdminEnabled:           "false",
		EmailApiKeyExpirationEnabled:               "false",
		EmailVerificationEnabled:                   "false",
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
	}
}

// fromLegacyConfig builds an appConfigModel from a legacy config map
// The map's keys correspond to the "json" tags on appConfigModel, and all values are strings that are cast to each field's type
// Keys that are missing (or have an empty value) retain the default value
func fromLegacyConfig(legacyCfg map[string]string) (*AppConfigModel, error) {
	// Start from the default configuration, then override with the values from the legacy config
	dest := getDefaultConfig()

	rt := reflect.ValueOf(dest).Elem().Type()
	rv := reflect.ValueOf(dest).Elem()
	for i := range rt.NumField() {
		field := rt.Field(i)

		// Get the value of the json tag, taking only what's before the comma
		key, _, _ := strings.Cut(field.Tag.Get("json"), ",")

		// Look up the value in the legacy config
		// If the key is missing or the value is empty, we keep the default value
		value, ok := legacyCfg[key]
		if !ok || value == "" {
			continue
		}

		// Cast the string value to the field's type
		fv := rv.Field(i)
		switch fv.Kind() { //nolint:exhaustive
		case reflect.String:
			fv.SetString(value)
		case reflect.Bool:
			fv.SetBool(utils.IsTruthy(value))
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			n, err := strconv.ParseInt(value, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("failed to parse integer value for key '%s': %w", key, err)
			}
			fv.SetInt(n)
		default:
			return nil, fmt.Errorf("unsupported field type '%s' for key '%s'", fv.Kind(), key)
		}
	}

	return dest, nil
}

// Replace updates every configuration property with the values from the input DTO
// An empty string value resets the corresponding property to its default value
func (m *AppConfigModel) Replace(input dto.AppConfigUpdateDto) {
	// Collect the values from the input DTO into a map, keyed by the "json" tag
	inRv := reflect.ValueOf(input)
	inRt := inRv.Type()
	values := make(map[string]string, inRt.NumField())
	for i := range inRt.NumField() {
		// Get the value of the json tag, taking only what's before the comma
		key, _, _ := strings.Cut(inRt.Field(i).Tag.Get("json"), ",")
		values[key] = inRv.Field(i).String()
	}

	// Iterate through all the properties, setting each one from the input
	// Properties that are missing from the input or have an empty value are reset to their default
	defaults := reflect.ValueOf(getDefaultConfig()).Elem()
	rv := reflect.ValueOf(m).Elem()
	rt := rv.Type()
	for i := range rt.NumField() {
		key, _, _ := strings.Cut(rt.Field(i).Tag.Get("json"), ",")

		value, ok := values[key]
		if !ok || value == "" {
			value = defaults.Field(i).String()
		}

		rv.Field(i).SetString(value)
	}
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
