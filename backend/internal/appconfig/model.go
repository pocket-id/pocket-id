package appconfig

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/italypaleale/go-kit/utils"

	"github.com/pocket-id/pocket-id/backend/internal/dto"
)

type AppConfigModel struct {
	// General
	AppName             string `json:"appName" public:"true"`
	SessionDuration     string `json:"sessionDuration" type:"int"` // In minutes
	HomePageURL         string `json:"homePageUrl" public:"true"`
	EmailsVerified      string `json:"emailsVerified" type:"bool"`
	AccentColor         string `json:"accentColor" public:"true"`
	DisableAnimations   string `json:"disableAnimations" type:"bool" public:"true"`
	AllowOwnAccountEdit string `json:"allowOwnAccountEdit" type:"bool" public:"true"`
	AllowUserSignups    string `json:"allowUserSignups" public:"true"`

	SignupDefaultUserGroupIDs string `json:"signupDefaultUserGroupIDs"` // JSON-encoded array of strings
	SignupDefaultCustomClaims string `json:"signupDefaultCustomClaims"` // JSON-encoded array of {key:string,value:string}
	// Email
	RequireUserEmail                           string `json:"requireUserEmail" type:"bool" public:"true"`
	SmtpHost                                   string `json:"smtpHost"`
	SmtpPort                                   string `json:"smtpPort"`
	SmtpFrom                                   string `json:"smtpFrom"`
	SmtpUser                                   string `json:"smtpUser"`
	SmtpPassword                               string `json:"smtpPassword" sensitive:"true"`
	SmtpTls                                    string `json:"smtpTls"`
	SmtpSkipCertVerify                         string `json:"smtpSkipCertVerify" type:"bool"`
	EmailLoginNotificationEnabled              string `json:"emailLoginNotificationEnabled" type:"bool"`
	EmailOneTimeAccessAsUnauthenticatedEnabled string `json:"emailOneTimeAccessAsUnauthenticatedEnabled" type:"bool" public:"true"`
	EmailOneTimeAccessAsAdminEnabled           string `json:"emailOneTimeAccessAsAdminEnabled" type:"bool" public:"true"`
	EmailApiKeyExpirationEnabled               string `json:"emailApiKeyExpirationEnabled" type:"bool"`
	EmailVerificationEnabled                   string `json:"emailVerificationEnabled" type:"bool" public:"true"`
	// LDAP
	LdapEnabled                        string `json:"ldapEnabled" type:"bool" public:"true"`
	LdapUrl                            string `json:"ldapUrl"`
	LdapBindDn                         string `json:"ldapBindDn"`
	LdapBindPassword                   string `json:"ldapBindPassword" sensitive:"true"`
	LdapBase                           string `json:"ldapBase"`
	LdapUserSearchFilter               string `json:"ldapUserSearchFilter"`
	LdapUserGroupSearchFilter          string `json:"ldapUserGroupSearchFilter"`
	LdapSkipCertVerify                 string `json:"ldapSkipCertVerify" type:"bool"`
	LdapAttributeUserUniqueIdentifier  string `json:"ldapAttributeUserUniqueIdentifier"`
	LdapAttributeUserUsername          string `json:"ldapAttributeUserUsername"`
	LdapAttributeUserEmail             string `json:"ldapAttributeUserEmail"`
	LdapAttributeUserFirstName         string `json:"ldapAttributeUserFirstName"`
	LdapAttributeUserLastName          string `json:"ldapAttributeUserLastName"`
	LdapAttributeUserDisplayName       string `json:"ldapAttributeUserDisplayName"`
	LdapAttributeUserProfilePicture    string `json:"ldapAttributeUserProfilePicture"`
	LdapAttributeGroupMember           string `json:"ldapAttributeGroupMember"`
	LdapAttributeGroupUniqueIdentifier string `json:"ldapAttributeGroupUniqueIdentifier"`
	LdapAttributeGroupName             string `json:"ldapAttributeGroupName"`
	LdapAdminGroupName                 string `json:"ldapAdminGroupName"`
	LdapSoftDeleteUsers                string `json:"ldapSoftDeleteUsers" type:"bool"`
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
func (m *AppConfigModel) Replace(input dto.AppConfigUpdateDto) error {
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

	return nil
}

// Update sets configuration properties from the provided key-value pairs
// Keys correspond to the "json" tags on the model
// An empty string value resets the property to its default value
func (m *AppConfigModel) Update(keysAndValues ...string) error {
	// Count of keysAndValues must be even
	if len(keysAndValues)%2 != 0 {
		return errors.New("invalid number of arguments received")
	}

	rv := reflect.ValueOf(m).Elem()
	rt := rv.Type()
	defaults := reflect.ValueOf(getDefaultConfig()).Elem()

	// Iterate through the key-value pairs
	// (Note the += 2, as we are iterating through key-value pairs)
	for i := 1; i < len(keysAndValues); i += 2 {
		key := keysAndValues[i-1]
		value := keysAndValues[i]

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
