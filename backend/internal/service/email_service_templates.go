package service

import (
	"fmt"
	"time"

	"github.com/pocket-id/pocket-id/backend/internal/utils/email"
)

/**
How to add new template:
- pick unique and descriptive template ${name} (for example "login-with-new-device")
- in backend/resources/email-templates/ create "${name}_html.tmpl" and "${name}_text.tmpl"
- create xxxxTemplate and xxxxTemplateData (for example NewLoginTemplate and NewLoginTemplateData)
  - Path *must* be ${name}
- add xxxTemplate.Path to "emailTemplatePaths" at the end

Notes:
- backend app must be restarted to reread all the template files
- root "." object in templates is `email.TemplateData`
- xxxxTemplateData structure is visible under .Data in templates
*/

var NewLoginTemplate = email.Template[NewLoginTemplateData]{
	Path: "login-with-new-device",
	Title: func(data *email.TemplateData[NewLoginTemplateData]) string {
		return fmt.Sprintf("New device login with %s", data.AppName)
	},
}

var OneTimeAccessTemplate = email.Template[OneTimeAccessTemplateData]{
	Path: "one-time-access",
	Title: func(data *email.TemplateData[OneTimeAccessTemplateData]) string {
		return "Login Code"
	},
}

var TestTemplate = email.Template[struct{}]{
	Path: "test",
	Title: func(data *email.TemplateData[struct{}]) string {
		return "Test email"
	},
}

var ApiKeyExpiringSoonTemplate = email.Template[ApiKeyExpiringSoonTemplateData]{
	Path: "api-key-expiring-soon",
	Title: func(data *email.TemplateData[ApiKeyExpiringSoonTemplateData]) string {
		return fmt.Sprintf("API Key \"%s\" Expiring Soon", data.Data.ApiKeyName)
	},
}

type NewLoginTemplateData struct {
	IPAddress string
	Country   string
	City      string
	Device    string
	DateTime  time.Time
}

type OneTimeAccessTemplateData = struct {
	Code              string
	LoginLink         string
	LoginLinkWithCode string
}

type ApiKeyExpiringSoonTemplateData struct {
	Name       string
	ApiKeyName string
	ExpiresAt  time.Time
}

// this is list of all template paths used for preloading templates
var emailTemplatesPaths = []string{NewLoginTemplate.Path, OneTimeAccessTemplate.Path, TestTemplate.Path, ApiKeyExpiringSoonTemplate.Path}
