package model

import (
	"strconv"
	"time"
)

type AppConfigVariable struct {
	Key          string `gorm:"primaryKey;not null"`
	Type         string
	IsPublic     bool
	IsInternal   bool
	Value        string
	DefaultValue string
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
	AppName             AppConfigVariable
	SessionDuration     AppConfigVariable
	EmailsVerified      AppConfigVariable
	AllowOwnAccountEdit AppConfigVariable
	// Internal
	BackgroundImageType AppConfigVariable
	LogoLightImageType  AppConfigVariable
	LogoDarkImageType   AppConfigVariable
	// Email
	SmtpHost                      AppConfigVariable
	SmtpPort                      AppConfigVariable
	SmtpFrom                      AppConfigVariable
	SmtpUser                      AppConfigVariable
	SmtpPassword                  AppConfigVariable
	SmtpTls                       AppConfigVariable
	SmtpSkipCertVerify            AppConfigVariable
	EmailLoginNotificationEnabled AppConfigVariable
	EmailOneTimeAccessEnabled     AppConfigVariable
	// LDAP
	LdapEnabled                        AppConfigVariable
	LdapUrl                            AppConfigVariable
	LdapBindDn                         AppConfigVariable
	LdapBindPassword                   AppConfigVariable
	LdapBase                           AppConfigVariable
	LdapUserSearchFilter               AppConfigVariable
	LdapUserGroupSearchFilter          AppConfigVariable
	LdapSkipCertVerify                 AppConfigVariable
	LdapAttributeUserUniqueIdentifier  AppConfigVariable
	LdapAttributeUserUsername          AppConfigVariable
	LdapAttributeUserEmail             AppConfigVariable
	LdapAttributeUserFirstName         AppConfigVariable
	LdapAttributeUserLastName          AppConfigVariable
	LdapAttributeUserProfilePicture    AppConfigVariable
	LdapAttributeGroupMember           AppConfigVariable
	LdapAttributeGroupUniqueIdentifier AppConfigVariable
	LdapAttributeGroupName             AppConfigVariable
	LdapAttributeAdminGroup            AppConfigVariable
}
