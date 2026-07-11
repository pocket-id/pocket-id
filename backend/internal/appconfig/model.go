package appconfig

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/italypaleale/go-kit/utils"
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
