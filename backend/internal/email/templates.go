package email

import (
	"fmt"
	"time"
)

// Every template path must have matching text and HTML resources and be listed in templatePaths so startup validates both variants

var newLoginTemplate = template[newLoginTemplateData]{
	path: "login-with-new-device",
	title: func(data *templateData[newLoginTemplateData]) string {
		return fmt.Sprintf("New device login with %s", data.AppName)
	},
}

var oneTimeAccessTemplate = template[oneTimeAccessTemplateData]{
	path: "one-time-access",
	title: func(_ *templateData[oneTimeAccessTemplateData]) string {
		return "Login Code"
	},
}

var testTemplate = template[struct{}]{
	path: "test",
	title: func(_ *templateData[struct{}]) string {
		return "Test email"
	},
}

var apiKeyExpiringSoonTemplate = template[apiKeyExpiringSoonTemplateData]{
	path: "api-key-expiring-soon",
	title: func(data *templateData[apiKeyExpiringSoonTemplateData]) string {
		return fmt.Sprintf("API Key \"%s\" Expiring Soon", data.Data.ApiKeyName)
	},
}

var emailVerificationTemplate = template[emailVerificationTemplateData]{
	path: "email-verification",
	title: func(data *templateData[emailVerificationTemplateData]) string {
		return "Verify your " + data.AppName + " email address"
	},
}

type newLoginTemplateData struct {
	IPAddress string
	Country   string
	City      string
	Device    string
	DateTime  time.Time
}

type oneTimeAccessTemplateData struct {
	Code              string
	LoginLink         string
	LoginLinkWithCode string
	ExpirationString  string
}

type apiKeyExpiringSoonTemplateData struct {
	Name       string
	ApiKeyName string
	ExpiresAt  time.Time
}

type emailVerificationTemplateData struct {
	UserFullName     string
	VerificationLink string
}

var templatePaths = []string{
	newLoginTemplate.path,
	oneTimeAccessTemplate.path,
	testTemplate.path,
	apiKeyExpiringSoonTemplate.path,
	emailVerificationTemplate.path,
}
