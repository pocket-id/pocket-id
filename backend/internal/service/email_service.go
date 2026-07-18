package service

import (
	"context"
	"errors"
	"fmt"
	htemplate "html/template"
	"net"
	"net/url"
	"strings"
	ttemplate "text/template"

	"github.com/italypaleale/go-kit/emailer"
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/appconfig"
	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	"github.com/pocket-id/pocket-id/backend/internal/utils/email"
)

type EmailService struct {
	appConfigService *appconfig.AppConfigService
	db               *gorm.DB
	htmlTemplates    map[string]*htemplate.Template
	textTemplates    map[string]*ttemplate.Template
}

func NewEmailService(db *gorm.DB, appConfigService *appconfig.AppConfigService) (*EmailService, error) {
	htmlTemplates, err := email.PrepareHTMLTemplates(emailTemplatesPaths)
	if err != nil {
		return nil, fmt.Errorf("prepare html templates: %w", err)
	}

	textTemplates, err := email.PrepareTextTemplates(emailTemplatesPaths)
	if err != nil {
		return nil, fmt.Errorf("prepare html templates: %w", err)
	}

	return &EmailService{
		appConfigService: appConfigService,
		db:               db,
		htmlTemplates:    htmlTemplates,
		textTemplates:    textTemplates,
	}, nil
}

func (srv *EmailService) SendTestEmail(ctx context.Context, recipientUserId string) error {
	var user model.User
	err := srv.db.
		WithContext(ctx).
		First(&user, "id = ?", recipientUserId).
		Error
	if err != nil {
		return err
	}

	if user.Email == nil {
		return &common.UserEmailNotSetError{}
	}

	return SendEmail(ctx, srv,
		email.Address{
			Email: *user.Email,
			Name:  user.FullName(),
		}, TestTemplate, nil)
}

func SendEmail[V any](ctx context.Context, srv *EmailService, toEmail email.Address, template email.Template[V], tData *V) error {
	dbConfig := srv.appConfigService.GetDbConfig()

	data := &email.TemplateData[V]{
		AppName: dbConfig.AppName.Value,
		LogoURL: common.EnvConfig.AppURL + "/api/application-images/email",
		Data:    tData,
	}

	// Render the text and HTML bodies
	text, html, err := renderBody(srv, template, data)
	if err != nil {
		return fmt.Errorf("prepare email body for '%s': %w", template.Path, err)
	}

	// Configure the emailer from the current app config
	e, err := srv.getEmailer(ctx, dbConfig)
	if err != nil {
		return fmt.Errorf("failed to configure emailer: %w", err)
	}

	// Send the email
	err = e.SendEmail(ctx, emailer.EmailAddress{
		Name:    toEmail.Name,
		Address: toEmail.Email,
	}, template.Title(data), emailer.SendEmailMessage{
		Text: text,
		HTML: html,
	})
	if err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	return nil
}

// getEmailer builds an emailer.Emailer from the current app config.
func (srv *EmailService) getEmailer(ctx context.Context, dbConfig *appconfig.AppConfigModel) (emailer.Emailer, error) {
	// We support SMTP only (for now)
	connString, err := smtpConnString(dbConfig)
	if err != nil {
		return nil, err
	}

	return emailer.NewEmailer(ctx, emailer.NewEmailerOpts{
		ConnString: connString,
	})
}

// smtpConnString builds the SMTP connection string that go-kit's emailer expects:
// smtp://<username>:<password>@<host>:<port>?fromAddress=<address>&fromName=<name>&tls=<none|starttls|tls>&insecureSkipVerify=<true|false>
func smtpConnString(dbConfig *appconfig.AppConfigModel) (string, error) {
	host := dbConfig.SmtpHost.String()
	if host == "" {
		return "", errors.New("SMTP host is not configured")
	}

	u := &url.URL{
		Scheme: "smtp",
		Host:   host,
	}
	port := dbConfig.SmtpPort.String()
	if port != "" {
		u.Host = net.JoinHostPort(host, port)
	}

	// Include credentials when set
	smtpUser := dbConfig.SmtpUser.String()
	smtpPassword := dbConfig.SmtpPassword.String()
	if smtpUser != "" || smtpPassword != "" {
		u.User = url.UserPassword(smtpUser, smtpPassword)
	}

	// TLS values from config: none, starttls, tls
	tlsMode := dbConfig.SmtpTls.String()
	if tlsMode == "" {
		tlsMode = "none"
	}

	// Build the query string args
	q := url.Values{}
	q.Set("fromAddress", dbConfig.SmtpFrom.String())
	q.Set("fromName", dbConfig.AppName.String())
	q.Set("tls", tlsMode)
	if dbConfig.SmtpSkipCertVerify.IsTrue() {
		q.Set("insecureSkipVerify", "true")
	}
	u.RawQuery = q.Encode()

	// Return the connection string
	return u.String(), nil
}

// renderBody renders the text and HTML templates for the message into strings
func renderBody[V any](srv *EmailService, template email.Template[V], data *email.TemplateData[V]) (text string, html string, err error) {
	textBuilder := &strings.Builder{}
	err = email.GetTemplate(srv.textTemplates, template).ExecuteTemplate(textBuilder, "root", data)
	if err != nil {
		return "", "", fmt.Errorf("execute text template: %w", err)
	}

	htmlBuilder := &strings.Builder{}
	err = email.GetTemplate(srv.htmlTemplates, template).ExecuteTemplate(htmlBuilder, "root", data)
	if err != nil {
		return "", "", fmt.Errorf("execute html template: %w", err)
	}

	return textBuilder.String(), htmlBuilder.String(), nil
}
