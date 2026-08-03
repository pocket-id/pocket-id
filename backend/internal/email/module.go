package email

import (
	"context"
	"errors"
	"fmt"
	htemplate "html/template"
	"net"
	"net/url"
	"path"
	"strings"
	ttemplate "text/template"
	"time"

	"github.com/italypaleale/go-kit/emailer"
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/appconfig"
	"github.com/pocket-id/pocket-id/backend/internal/apperror"
	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	"github.com/pocket-id/pocket-id/backend/resources"
)

type Module struct {
	db            *gorm.DB
	htmlTemplates map[string]*htemplate.Template
	textTemplates map[string]*ttemplate.Template
}

type template[V any] struct {
	path  string
	title func(data *templateData[V]) string
}

type templateData[V any] struct {
	AppName string
	LogoURL string
	Data    *V
}

type address struct {
	name  string
	email string
}

func New(db *gorm.DB) (*Module, error) {
	// Preload both template variants so missing or invalid embedded templates fail during startup
	htmlTemplates, err := prepareHTMLTemplates(templatePaths)
	if err != nil {
		return nil, fmt.Errorf("prepare HTML templates: %w", err)
	}

	textTemplates, err := prepareTextTemplates(templatePaths)
	if err != nil {
		return nil, fmt.Errorf("prepare text templates: %w", err)
	}

	return &Module{
		db:            db,
		htmlTemplates: htmlTemplates,
		textTemplates: textTemplates,
	}, nil
}

func (m *Module) SendTestEmail(ctx context.Context, dbConfig *appconfig.AppConfigModel, recipientUserID string) error {
	// Resolve the recipient from the database so test emails use the same user identity as notification emails
	var user model.User
	err := m.db.
		WithContext(ctx).
		First(&user, "id = ?", recipientUserID).
		Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return apperror.UserNotFound()
	}
	if err != nil {
		return err
	}

	if user.Email == nil {
		return apperror.UserEmailNotSet()
	}

	return send(ctx, m, dbConfig, address{
		name:  user.FullName(),
		email: *user.Email,
	}, testTemplate, nil)
}

func (m *Module) SendEmailVerification(ctx context.Context, dbConfig *appconfig.AppConfigModel, userFullName, userEmail, verificationLink string) error {
	return send(ctx, m, dbConfig, address{
		name:  userFullName,
		email: userEmail,
	}, emailVerificationTemplate, &emailVerificationTemplateData{
		UserFullName:     userFullName,
		VerificationLink: verificationLink,
	})
}

func (m *Module) SendOneTimeAccessEmail(ctx context.Context, dbConfig *appconfig.AppConfigModel, userFullName, userEmail, code, loginLink, loginLinkWithCode, expirationString string) error {
	return send(ctx, m, dbConfig, address{
		name:  userFullName,
		email: userEmail,
	}, oneTimeAccessTemplate, &oneTimeAccessTemplateData{
		Code:              code,
		LoginLink:         loginLink,
		LoginLinkWithCode: loginLinkWithCode,
		ExpirationString:  expirationString,
	})
}

func (m *Module) SendNewLogin(ctx context.Context, dbConfig *appconfig.AppConfigModel, userFullName, userEmail, ipAddress, country, city, device string, dateTime time.Time) error {
	return send(ctx, m, dbConfig, address{
		name:  userFullName,
		email: userEmail,
	}, newLoginTemplate, &newLoginTemplateData{
		IPAddress: ipAddress,
		Country:   country,
		City:      city,
		Device:    device,
		DateTime:  dateTime,
	})
}

func (m *Module) SendAPIKeyExpiringSoon(ctx context.Context, dbConfig *appconfig.AppConfigModel, userFullName, userEmail, firstName, apiKeyName string, expiresAt time.Time) error {
	return send(ctx, m, dbConfig, address{
		name:  userFullName,
		email: userEmail,
	}, apiKeyExpiringSoonTemplate, &apiKeyExpiringSoonTemplateData{
		Name:       firstName,
		ApiKeyName: apiKeyName,
		ExpiresAt:  expiresAt,
	})
}

func send[V any](ctx context.Context, module *Module, dbConfig *appconfig.AppConfigModel, recipient address, tmpl template[V], data *V) error {
	// Combine application metadata with message-specific data before rendering both MIME alternatives
	templateData := &templateData[V]{
		AppName: dbConfig.AppName.String(),
		LogoURL: common.EnvConfig.AppURL + "/api/application-images/email",
		Data:    data,
	}

	// Render the complete message before opening an SMTP connection so template failures never produce partial deliveries
	text, html, err := renderBody(module, tmpl, templateData)
	if err != nil {
		return fmt.Errorf("prepare email body for '%s': %w", tmpl.path, err)
	}

	// Resolve SMTP settings for each delivery so application configuration changes take effect without restarting
	emailerService, err := module.getEmailer(ctx, dbConfig)
	if err != nil {
		return fmt.Errorf("failed to configure emailer: %w", err)
	}

	// Send text and HTML together so clients can select the format they support
	err = emailerService.SendEmail(ctx, emailer.EmailAddress{
		Name:    recipient.name,
		Address: recipient.email,
	}, tmpl.title(templateData), emailer.SendEmailMessage{
		Text: text,
		HTML: html,
	})
	if err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	return nil
}

func (m *Module) getEmailer(ctx context.Context, dbConfig *appconfig.AppConfigModel) (emailer.Emailer, error) {
	connString, err := smtpConnString(dbConfig)
	if err != nil {
		return nil, err
	}

	return emailer.NewEmailer(ctx, emailer.NewEmailerOpts{
		ConnString: connString,
	})
}

func smtpConnString(dbConfig *appconfig.AppConfigModel) (string, error) {
	// Build the SMTP authority from the configured endpoint and optional credentials
	host := dbConfig.SmtpHost.String()
	if host == "" {
		return "", errors.New("SMTP host is not configured")
	}

	smtpURL := &url.URL{
		Scheme: "smtp",
		Host:   host,
	}
	port := dbConfig.SmtpPort.String()
	if port != "" {
		smtpURL.Host = net.JoinHostPort(host, port)
	}

	smtpUser := dbConfig.SmtpUser.String()
	smtpPassword := dbConfig.SmtpPassword.String()
	if smtpUser != "" || smtpPassword != "" {
		smtpURL.User = url.UserPassword(smtpUser, smtpPassword)
	}

	// Preserve sender identity and transport security settings in the connection string consumed by the emailer
	tlsMode := dbConfig.SmtpTls.String()
	if tlsMode == "" {
		tlsMode = "none"
	}

	query := url.Values{}
	query.Set("fromAddress", dbConfig.SmtpFrom.String())
	query.Set("fromName", dbConfig.AppName.String())
	query.Set("tls", tlsMode)
	if dbConfig.SmtpSkipCertVerify.IsTrue() {
		query.Set("insecureSkipVerify", "true")
	}
	smtpURL.RawQuery = query.Encode()

	return smtpURL.String(), nil
}

func renderBody[V any](module *Module, tmpl template[V], data *templateData[V]) (text string, html string, err error) {
	// Render both variants from the same data so the plain-text and HTML messages cannot diverge
	textBuilder := &strings.Builder{}
	err = module.textTemplates[tmpl.path].ExecuteTemplate(textBuilder, "root", data)
	if err != nil {
		return "", "", fmt.Errorf("execute text template: %w", err)
	}

	htmlBuilder := &strings.Builder{}
	err = module.htmlTemplates[tmpl.path].ExecuteTemplate(htmlBuilder, "root", data)
	if err != nil {
		return "", "", fmt.Errorf("execute HTML template: %w", err)
	}

	return textBuilder.String(), htmlBuilder.String(), nil
}

func prepareTextTemplates(templates []string) (map[string]*ttemplate.Template, error) {
	textTemplates := make(map[string]*ttemplate.Template, len(templates))
	for _, tmpl := range templates {
		templatePath := path.Join("email-templates", tmpl+"_text.tmpl")

		parsedTemplate, err := ttemplate.ParseFS(resources.FS, templatePath)
		if err != nil {
			return nil, fmt.Errorf("parsing template '%s': %w", tmpl, err)
		}

		textTemplates[tmpl] = parsedTemplate
	}

	return textTemplates, nil
}

func prepareHTMLTemplates(templates []string) (map[string]*htemplate.Template, error) {
	htmlTemplates := make(map[string]*htemplate.Template, len(templates))
	for _, tmpl := range templates {
		templatePath := path.Join("email-templates", tmpl+"_html.tmpl")

		parsedTemplate, err := htemplate.ParseFS(resources.FS, templatePath)
		if err != nil {
			return nil, fmt.Errorf("parsing template '%s': %w", tmpl, err)
		}

		htmlTemplates[tmpl] = parsedTemplate
	}

	return htmlTemplates, nil
}
