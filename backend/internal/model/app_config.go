package model

import (
	"strconv"
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
	BackgroundImageType AppConfigVariable `key:"backgroundImageType,internal"`
	LogoLightImageType  AppConfigVariable `key:"logoLightImageType,internal"`
	LogoDarkImageType   AppConfigVariable `key:"logoDarkImageType,internal"`
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
}
