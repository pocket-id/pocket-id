package dto

import (
	"encoding/json"
	"net/mail"

	"github.com/danielgtaylor/huma/v2"
)

type PublicAppConfigVariableDto struct {
	Key   string `json:"key"`
	Type  string `json:"type"`
	Value string `json:"value"`
}

type AppConfigVariableDto struct {
	PublicAppConfigVariableDto
	IsPublic bool `json:"isPublic"`
}

type AppConfigUpdateDto struct {
	AppName                                    string `json:"appName" required:"false" minLength:"1" maxLength:"30" unorm:"nfc"`
	SessionDuration                            string `json:"sessionDuration" required:"false"`
	HomePageURL                                string `json:"homePageUrl" required:"false"`
	EmailsVerified                             string `json:"emailsVerified" required:"false"`
	DisableAnimations                          string `json:"disableAnimations" required:"false"`
	AllowOwnAccountEdit                        string `json:"allowOwnAccountEdit" required:"false"`
	AllowUserSignups                           string `json:"allowUserSignups" required:"false" enum:"disabled,withToken,open"`
	SignupDefaultUserGroupIDs                  string `json:"signupDefaultUserGroupIDs" required:"false"`
	SignupDefaultCustomClaims                  string `json:"signupDefaultCustomClaims" required:"false"`
	AccentColor                                string `json:"accentColor" required:"false"`
	RequireUserEmail                           string `json:"requireUserEmail" required:"false"`
	SmtpHost                                   string `json:"smtpHost" required:"false"`
	SmtpPort                                   string `json:"smtpPort" required:"false"`
	SmtpFrom                                   string `json:"smtpFrom" required:"false"`
	SmtpUser                                   string `json:"smtpUser" required:"false"`
	SmtpPassword                               string `json:"smtpPassword" required:"false"`
	SmtpTls                                    string `json:"smtpTls" required:"false" enum:"none,starttls,tls"`
	SmtpSkipCertVerify                         string `json:"smtpSkipCertVerify" required:"false"`
	LdapEnabled                                string `json:"ldapEnabled" required:"false"`
	LdapUrl                                    string `json:"ldapUrl" required:"false"`
	LdapBindDn                                 string `json:"ldapBindDn" required:"false"`
	LdapBindPassword                           string `json:"ldapBindPassword" required:"false"`
	LdapBase                                   string `json:"ldapBase" required:"false"`
	LdapUserSearchFilter                       string `json:"ldapUserSearchFilter" required:"false"`
	LdapUserGroupSearchFilter                  string `json:"ldapUserGroupSearchFilter" required:"false"`
	LdapSkipCertVerify                         string `json:"ldapSkipCertVerify" required:"false"`
	LdapAttributeUserUniqueIdentifier          string `json:"ldapAttributeUserUniqueIdentifier" required:"false"`
	LdapAttributeUserUsername                  string `json:"ldapAttributeUserUsername" required:"false"`
	LdapAttributeUserEmail                     string `json:"ldapAttributeUserEmail" required:"false"`
	LdapAttributeUserFirstName                 string `json:"ldapAttributeUserFirstName" required:"false"`
	LdapAttributeUserLastName                  string `json:"ldapAttributeUserLastName" required:"false"`
	LdapAttributeUserDisplayName               string `json:"ldapAttributeUserDisplayName" required:"false"`
	LdapAttributeUserProfilePicture            string `json:"ldapAttributeUserProfilePicture" required:"false"`
	LdapAttributeGroupMember                   string `json:"ldapAttributeGroupMember" required:"false"`
	LdapAttributeGroupUniqueIdentifier         string `json:"ldapAttributeGroupUniqueIdentifier" required:"false"`
	LdapAttributeGroupName                     string `json:"ldapAttributeGroupName" required:"false"`
	LdapAdminGroupName                         string `json:"ldapAdminGroupName" required:"false"`
	LdapSoftDeleteUsers                        string `json:"ldapSoftDeleteUsers" required:"false"`
	EmailOneTimeAccessAsAdminEnabled           string `json:"emailOneTimeAccessAsAdminEnabled" required:"false"`
	EmailOneTimeAccessAsUnauthenticatedEnabled string `json:"emailOneTimeAccessAsUnauthenticatedEnabled" required:"false"`
	EmailLoginNotificationEnabled              string `json:"emailLoginNotificationEnabled" required:"false"`
	EmailApiKeyExpirationEnabled               string `json:"emailApiKeyExpirationEnabled" required:"false"`
	EmailVerificationEnabled                   string `json:"emailVerificationEnabled" required:"false"`
}

func (d *AppConfigUpdateDto) Resolve(huma.Context) []error {
	var errs []error
	if d.SmtpFrom != "" {
		address, err := mail.ParseAddress(d.SmtpFrom)
		if err != nil || address.Address != d.SmtpFrom {
			errs = append(errs, &huma.ErrorDetail{Location: "body.smtpFrom", Message: "Field validation for 'SmtpFrom' failed on the 'email' tag"})
		}
	}
	if d.SignupDefaultUserGroupIDs != "" && !json.Valid([]byte(d.SignupDefaultUserGroupIDs)) {
		errs = append(errs, &huma.ErrorDetail{Location: "body.signupDefaultUserGroupIDs", Message: "Signup default user group IDs must be valid JSON"})
	}
	if d.SignupDefaultCustomClaims != "" && !json.Valid([]byte(d.SignupDefaultCustomClaims)) {
		errs = append(errs, &huma.ErrorDetail{Location: "body.signupDefaultCustomClaims", Message: "Signup default custom claims must be valid JSON"})
	}
	return errs
}
