package service

import (
	"bytes"
	"crypto/tls"
	"errors"
	"fmt"
	"github.com/emersion/go-sasl"
	"github.com/emersion/go-smtp"
	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	"github.com/pocket-id/pocket-id/backend/internal/utils/email"
	"gorm.io/gorm"
	htemplate "html/template"
	"mime/multipart"
	"mime/quotedprintable"
	"net/textproto"
	"os"
	ttemplate "text/template"
	"time"
	"github.com/google/uuid"
	"strings"
)

type EmailService struct {
	appConfigService *AppConfigService
	db               *gorm.DB
	htmlTemplates    map[string]*htemplate.Template
	textTemplates    map[string]*ttemplate.Template
}

func NewEmailService(appConfigService *AppConfigService, db *gorm.DB) (*EmailService, error) {
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

func (srv *EmailService) SendTestEmail(recipientUserId string) error {
	var user model.User
	if err := srv.db.First(&user, "id = ?", recipientUserId).Error; err != nil {
		return err
	}

	return SendEmail(srv,
		email.Address{
			Email: user.Email,
			Name:  user.FullName(),
		}, TestTemplate, nil)
}

func SendEmail[V any](srv *EmailService, toEmail email.Address, template email.Template[V], tData *V) error {
	data := &email.TemplateData[V]{
		AppName: srv.appConfigService.DbConfig.AppName.Value,
		LogoURL: common.EnvConfig.AppURL + "/api/application-configuration/logo",
		Data:    tData,
	}

	body, boundary, err := prepareBody(srv, template, data)
	if err != nil {
		return fmt.Errorf("prepare email body for '%s': %w", template.Path, err)
	}

	// Construct the email message
	c := email.NewComposer()
	c.AddHeader("Subject", template.Title(data))
	c.AddAddressHeader("From", []email.Address{
		{
			Email: srv.appConfigService.DbConfig.SmtpFrom.Value,
			Name:  srv.appConfigService.DbConfig.AppName.Value,
		},
	})
	c.AddAddressHeader("To", []email.Address{toEmail})
	c.AddHeaderRaw("Content-Type",
		fmt.Sprintf("multipart/alternative;\n boundary=%s;\n charset=UTF-8", boundary),
	)

	c.AddHeader("MIME-Version", "1.0")
	c.AddHeader("Date", time.Now().Format(time.RFC1123Z))

	// to create a message-id, we need the FQDN of the sending server, but that may be a docker hostname or localhost
	// so we use the domain of the from address instead (the same as Thunderbird does)

	from_address := srv.appConfigService.DbConfig.SmtpFrom.Value
	domain := ""
	if strings.Contains(from_address, "@") {
		domain = strings.Split(from_address, "@")[1]
	} else {
		// this is completely wrong, but what else could we do?
		domain = from_address
	}
	c.AddHeader("Message-ID", "<" + uuid.New().String() + "@" + domain + ">")

	c.Body(body)

	// Connect to the SMTP server
	client, err := srv.getSmtpClient()
	if err != nil {
		return fmt.Errorf("failed to connect to SMTP server: %w", err)
	}
	defer client.Close()

	// Send the email
	if err := srv.sendEmailContent(client, toEmail, c); err != nil {
		return fmt.Errorf("send email content: %w", err)
	}

	return nil
}

func (srv *EmailService) getSmtpClient() (client *smtp.Client, err error) {
	port := srv.appConfigService.DbConfig.SmtpPort.Value
	smtpAddress := srv.appConfigService.DbConfig.SmtpHost.Value + ":" + port

	tlsConfig := &tls.Config{
		InsecureSkipVerify: srv.appConfigService.DbConfig.SmtpSkipCertVerify.Value == "true",
		ServerName:         srv.appConfigService.DbConfig.SmtpHost.Value,
	}

	// Connect to the SMTP server based on TLS setting
	switch srv.appConfigService.DbConfig.SmtpTls.Value {
	case "none":
		client, err = smtp.Dial(smtpAddress)
	case "tls":
		client, err = smtp.DialTLS(smtpAddress, tlsConfig)
	case "starttls":
		client, err = smtp.DialStartTLS(
			smtpAddress,
			tlsConfig,
		)
	default:
		return nil, fmt.Errorf("invalid SMTP TLS setting: %s", srv.appConfigService.DbConfig.SmtpTls.Value)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to connect to SMTP server: %w", err)
	}

	client.CommandTimeout = 10 * time.Second

	// Send the HELO command
	if err := srv.sendHelloCommand(client); err != nil {
		return nil, fmt.Errorf("failed to send HELO command: %w", err)
	}

	// Set up the authentication if user or password are set
	smtpUser := srv.appConfigService.DbConfig.SmtpUser.Value
	smtpPassword := srv.appConfigService.DbConfig.SmtpPassword.Value

	if smtpUser != "" || smtpPassword != "" {
		// Authenticate with plain auth
		auth := sasl.NewPlainClient("", smtpUser, smtpPassword)
		if err := client.Auth(auth); err != nil {
			// If the server does not support plain auth, try login auth
			var smtpErr *smtp.SMTPError
			ok := errors.As(err, &smtpErr)
			if ok && smtpErr.Code == smtp.ErrAuthUnknownMechanism.Code {
				auth = sasl.NewLoginClient(smtpUser, smtpPassword)
				err = client.Auth(auth)
			}
			// Both plain and login auth failed
			if err != nil {
				return nil, fmt.Errorf("failed to authenticate: %w", err)
			}

		}
	}

	return client, err
}

func (srv *EmailService) sendHelloCommand(client *smtp.Client) error {
	hostname, err := os.Hostname()
	if err == nil {
		if err := client.Hello(hostname); err != nil {
			return err
		}
	}
	return nil
}

func (srv *EmailService) sendEmailContent(client *smtp.Client, toEmail email.Address, c *email.Composer) error {
	// Set the sender
	if err := client.Mail(srv.appConfigService.DbConfig.SmtpFrom.Value, nil); err != nil {
		return fmt.Errorf("failed to set sender: %w", err)
	}

	// Set the recipient
	if err := client.Rcpt(toEmail.Email, nil); err != nil {
		return fmt.Errorf("failed to set recipient: %w", err)
	}

	// Get a writer to write the email data
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("failed to start data: %w", err)
	}

	// Write the email content
	_, err = w.Write([]byte(c.String()))
	if err != nil {
		return fmt.Errorf("failed to write email data: %w", err)
	}

	// Close the writer
	if err := w.Close(); err != nil {
		return fmt.Errorf("failed to close data writer: %w", err)
	}

	return nil
}

func prepareBody[V any](srv *EmailService, template email.Template[V], data *email.TemplateData[V]) (string, string, error) {
	body := bytes.NewBuffer(nil)
	mpart := multipart.NewWriter(body)

	// prepare text part
	var textHeader = textproto.MIMEHeader{}
	textHeader.Add("Content-Type", "text/plain;\n charset=UTF-8")
	textHeader.Add("Content-Transfer-Encoding", "quoted-printable")
	textPart, err := mpart.CreatePart(textHeader)
	if err != nil {
		return "", "", fmt.Errorf("create text part: %w", err)
	}

	textQp := quotedprintable.NewWriter(textPart)
	err = email.GetTemplate(srv.textTemplates, template).ExecuteTemplate(textQp, "root", data)
	if err != nil {
		return "", "", fmt.Errorf("execute text template: %w", err)
	}

	// prepare html part
	var htmlHeader = textproto.MIMEHeader{}
	htmlHeader.Add("Content-Type", "text/html;\n charset=UTF-8")
	htmlHeader.Add("Content-Transfer-Encoding", "quoted-printable")
	htmlPart, err := mpart.CreatePart(htmlHeader)
	if err != nil {
		return "", "", fmt.Errorf("create html part: %w", err)
	}

	htmlQp := quotedprintable.NewWriter(htmlPart)
	err = email.GetTemplate(srv.htmlTemplates, template).ExecuteTemplate(htmlQp, "root", data)
	if err != nil {
		return "", "", fmt.Errorf("execute html template: %w", err)
	}

	err = mpart.Close()
	if err != nil {
		return "", "", fmt.Errorf("close multipart: %w", err)
	}

	return body.String(), mpart.Boundary(), nil
}
