package email

import (
	"time"
)

// Every template path must have matching text and HTML resources and be listed in templatePaths so startup validates both variants

var newLoginTemplate = template[newLoginTemplateData]{
	path: "login-with-new-device",
	title: func(data *templateData[newLoginTemplateData]) (string, error) {
		return data.T("new_sign_in_subject", "appName", data.AppName)
	},
}

var oneTimeAccessTemplate = template[oneTimeAccessTemplateData]{
	path: "one-time-access",
	title: func(data *templateData[oneTimeAccessTemplateData]) (string, error) {
		return data.T("one_time_access_subject", "appName", data.AppName)
	},
}

var signupInvitationTemplate = template[signupInvitationTemplateData]{
	path: "signup-invitation",
	title: func(data *templateData[signupInvitationTemplateData]) (string, error) {
		return data.T("signup_invitation_subject", "appName", data.AppName)
	},
}

var testTemplate = template[struct{}]{
	path: "test",
	title: func(data *templateData[struct{}]) (string, error) {
		return data.T("test_title")
	},
}

var apiKeyExpiringSoonTemplate = template[apiKeyExpiringSoonTemplateData]{
	path: "api-key-expiring-soon",
	title: func(data *templateData[apiKeyExpiringSoonTemplateData]) (string, error) {
		return data.T("api_key_expiring_subject", "apiKeyName", data.Data.ApiKeyName)
	},
}

var emailVerificationTemplate = template[emailVerificationTemplateData]{
	path: "email-verification",
	title: func(data *templateData[emailVerificationTemplateData]) (string, error) {
		return data.T("email_verification_subject", "appName", data.AppName)
	},
}

type newLoginTemplateData struct {
	IPAddress string
	Country   string
	City      string
	Device    string
	Method    string
	DateTime  time.Time
}

type oneTimeAccessTemplateData struct {
	Code              string
	LoginLink         string
	LoginLinkWithCode string
	Expiration        time.Duration
}

type signupInvitationTemplateData struct {
	SignupLink string
	Expiration time.Duration
}

type apiKeyExpiringSoonTemplateData struct {
	Name       string
	ApiKeyName string
	ExpiresAt  time.Time
}

type emailVerificationTemplateData struct {
	UserFullName     string
	VerificationLink string
	Expiration       time.Duration
}

var templatePaths = []string{
	newLoginTemplate.path,
	oneTimeAccessTemplate.path,
	signupInvitationTemplate.path,
	testTemplate.path,
	apiKeyExpiringSoonTemplate.path,
	emailVerificationTemplate.path,
}
