package model

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"
)

type AppConfigVariable struct {
	Key   string `gorm:"primaryKey;not null"`
	Value string
}

// IsTrue returns true if the value is a truthy string, such as "true", "t", "yes", "1", etc.
func (a *AppConfigVariable) IsTrue() bool {
	ok, _ := strconv.ParseBool(a.Value)
	return ok
}

// AsDurationMinutes returns the value as a time.Duration, interpreting the string as a whole number of minutes.
func (a *AppConfigVariable) AsDurationMinutes() time.Duration {
	val, err := strconv.Atoi(a.Value)
	if err != nil {
		return 0
	}
	return time.Duration(val) * time.Minute
}

type AppConfig struct {
	// General
	AppName             AppConfigVariable `key:"appName,public"` // Public
	SessionDuration     AppConfigVariable `key:"sessionDuration"`
	EmailsVerified      AppConfigVariable `key:"emailsVerified"`
	AllowOwnAccountEdit AppConfigVariable `key:"allowOwnAccountEdit,public"` // Public
	// Internal
	BackgroundImageType AppConfigVariable `key:"backgroundImageType,internal"` // Internal
	LogoLightImageType  AppConfigVariable `key:"logoLightImageType,internal"`  // Internal
	LogoDarkImageType   AppConfigVariable `key:"logoDarkImageType,internal"`   // Internal
	// Email
	SmtpHost                      AppConfigVariable `key:"smtpHost"`
	SmtpPort                      AppConfigVariable `key:"smtpPort"`
	SmtpFrom                      AppConfigVariable `key:"smtpFrom"`
	SmtpUser                      AppConfigVariable `key:"smtpUser"`
	SmtpPassword                  AppConfigVariable `key:"smtpPassword"`
	SmtpTls                       AppConfigVariable `key:"smtpTls"`
	SmtpSkipCertVerify            AppConfigVariable `key:"smtpSkipCertVerify"`
	EmailLoginNotificationEnabled AppConfigVariable `key:"emailLoginNotificationEnabled"`
	EmailOneTimeAccessEnabled     AppConfigVariable `key:"emailOneTimeAccessEnabled,public"` // Public
	// LDAP
	LdapEnabled                        AppConfigVariable `key:"ldapEnabled,public"` // Public
	LdapUrl                            AppConfigVariable `key:"ldapUrl"`
	LdapBindDn                         AppConfigVariable `key:"ldapBindDn"`
	LdapBindPassword                   AppConfigVariable `key:"ldapBindPassword"`
	LdapBase                           AppConfigVariable `key:"ldapBase"`
	LdapUserSearchFilter               AppConfigVariable `key:"ldapUserSearchFilter"`
	LdapUserGroupSearchFilter          AppConfigVariable `key:"ldapUserGroupSearchFilter"`
	LdapSkipCertVerify                 AppConfigVariable `key:"ldapSkipCertVerify"`
	LdapAttributeUserUniqueIdentifier  AppConfigVariable `key:"ldapAttributeUserUniqueIdentifier"`
	LdapAttributeUserUsername          AppConfigVariable `key:"ldapAttributeUserUsername"`
	LdapAttributeUserEmail             AppConfigVariable `key:"ldapAttributeUserEmail"`
	LdapAttributeUserFirstName         AppConfigVariable `key:"ldapAttributeUserFirstName"`
	LdapAttributeUserLastName          AppConfigVariable `key:"ldapAttributeUserLastName"`
	LdapAttributeUserProfilePicture    AppConfigVariable `key:"ldapAttributeUserProfilePicture"`
	LdapAttributeGroupMember           AppConfigVariable `key:"ldapAttributeGroupMember"`
	LdapAttributeGroupUniqueIdentifier AppConfigVariable `key:"ldapAttributeGroupUniqueIdentifier"`
	LdapAttributeGroupName             AppConfigVariable `key:"ldapAttributeGroupName"`
	LdapAttributeAdminGroup            AppConfigVariable `key:"ldapAttributeAdminGroup"`
	LdapSoftDeleteUsers                AppConfigVariable `key:"ldapSoftDeleteUsers"`
}

func (c *AppConfig) ToAppConfigVariableSlice(showAll bool) []AppConfigVariable {
	// Use reflection to iterate through all fields
	cfgValue := reflect.ValueOf(c).Elem()
	cfgType := cfgValue.Type()

	res := make([]AppConfigVariable, cfgType.NumField())

	for i := range cfgType.NumField() {
		field := cfgType.Field(i)

		key, attrs, _ := strings.Cut(field.Tag.Get("key"), ",")
		if key == "" {
			continue
		}

		// If we're only showing public variables and this is not public, skip it
		if !showAll && attrs != "public" {
			continue
		}

		fieldValue := cfgValue.Field(i)

		res[i] = AppConfigVariable{
			Key:   key,
			Value: fieldValue.FieldByName("Value").String(),
		}
	}

	return res
}

func (c *AppConfig) FieldByKey(key string) (string, error) {
	rv := reflect.ValueOf(c).Elem()
	rt := rv.Type()

	// Find the field in the struct whose "key" tag matches
	for i := range rt.NumField() {
		// Grab only the first part of the key, if there's a comma with additional properties
		tagValue, _, _ := strings.Cut(rt.Field(i).Tag.Get("key"), ",")
		if tagValue != key {
			continue
		}

		valueField := rv.Field(i).FieldByName("Value")
		return valueField.String(), nil
	}

	// If we are here, the config key was not found
	return "", AppConfigKeyNotFoundError{field: key}
}

func (c *AppConfig) UpdateField(key string, value string, noInternal bool) error {
	rv := reflect.ValueOf(c).Elem()
	rt := rv.Type()

	// Find the field in the struct whose "key" tag matches, then update that
	for i := range rt.NumField() {
		// Separate the key (before the comma) from any optional attributes after
		tagValue, attrs, _ := strings.Cut(rt.Field(i).Tag.Get("key"), ",")
		if tagValue != key {
			continue
		}

		// If the field is internal and noInternal is true, we skip that
		if noInternal && attrs == "internal" {
			return AppConfigInternalForbiddenError{field: key}
		}

		valueField := rv.Field(i).FieldByName("Value")
		if !valueField.CanSet() {
			return fmt.Errorf("field Value in AppConfigVariable is not settable for config key '%s'", key)
		}

		// Update the value
		valueField.SetString(value)

		// Return once updated
		return nil
	}

	// If we're here, we have not found the right field to update
	return AppConfigKeyNotFoundError{field: key}
}

type AppConfigKeyNotFoundError struct {
	field string
}

func (e AppConfigKeyNotFoundError) Error() string {
	return fmt.Sprintf("cannot find config key '%s'", e.field)
}

func (e AppConfigKeyNotFoundError) Is(target error) bool {
	// Ignore the field property when checking if an error is of the type AppConfigKeyNotFoundError
	x := AppConfigKeyNotFoundError{}
	return errors.As(target, &x)
}

type AppConfigInternalForbiddenError struct {
	field string
}

func (e AppConfigInternalForbiddenError) Error() string {
	return fmt.Sprintf("field '%s' is internal and can't be updated", e.field)
}

func (e AppConfigInternalForbiddenError) Is(target error) bool {
	// Ignore the field property when checking if an error is of the type AppConfigInternalForbiddenError
	x := AppConfigInternalForbiddenError{}
	return errors.As(target, &x)
}
