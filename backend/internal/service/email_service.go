package service

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	htemplate "html/template"
	"io"
	"mime/multipart"
	"mime/quotedprintable"
	"net/textproto"
	"os"
	"strings"
	ttemplate "text/template"
	"time"

	"github.com/emersion/go-sasl"
	"github.com/emersion/go-smtp"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	"github.com/pocket-id/pocket-id/backend/internal/utils/email"
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

func (srv *EmailService) SendTestEmail(ctx context.Context, recipientUserId string) error {
	var user model.User
	err := srv.db.
		WithContext(ctx).
		First(&user, "id = ?", recipientUserId).
		Error
	if err != nil {
		return err
	}

	return SendEmail(ctx, srv,
		email.Address{
			Email: user.Email,
			Name:  user.FullName(),
		}, TestTemplate, nil)
}

func SendEmail[V any](ctx context.Context, srv *EmailService, toEmail email.Address, template email.Template[V], tData *V) error {
	dbConfig := srv.appConfigService.GetDbConfig()

	data := &email.TemplateData[V]{
		AppName: dbConfig.AppName.Value,
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
			Email: dbConfig.SmtpFrom.Value,
			Name:  dbConfig.AppName.Value,
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
	// if the address does not have an @ (which would be unusual), we use hostname

	from_address := dbConfig.SmtpFrom.Value
	domain := ""
	if strings.Contains(from_address, "@") {
		domain = strings.Split(from_address, "@")[1]
	} else {
		hostname, err := os.Hostname()
		if err != nil {
			// can that happen? we just give up
			return fmt.Errorf("failed to get own hostname: %w", err)
		} else {
			domain = hostname
		}
	}
	c.AddHeader("Message-ID", "<"+uuid.New().String()+"@"+domain+">")

	c.Body(body)

	// Check if the context is still valid before attemtping to connect
	// We need to do this because the smtp library doesn't have context support
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		// All good
	}

	// Connect to the SMTP server
	client, err := srv.getSmtpClient()
	if err != nil {
		return fmt.Errorf("failed to connect to SMTP server: %w", err)
	}
	defer client.Close()

	// Check if the context is still valid before sending the email
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		// All good
	}

	// Send the email
	if err := srv.sendEmailContent(client, toEmail, c); err != nil {
		return fmt.Errorf("send email content: %w", err)
	}

	return nil
}

func (srv *EmailService) getSmtpClient() (client *smtp.Client, err error) {
	dbConfig := srv.appConfigService.GetDbConfig()

	port := dbConfig.SmtpPort.Value
	smtpAddress := dbConfig.SmtpHost.Value + ":" + port

	tlsConfig := &tls.Config{
		InsecureSkipVerify: dbConfig.SmtpSkipCertVerify.IsTrue(), //nolint:gosec
		ServerName:         dbConfig.SmtpHost.Value,
	}

	// Connect to the SMTP server based on TLS setting
	switch dbConfig.SmtpTls.Value {
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
		return nil, fmt.Errorf("invalid SMTP TLS setting: %s", dbConfig.SmtpTls.Value)
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
	smtpUser := dbConfig.SmtpUser.Value
	smtpPassword := dbConfig.SmtpPassword.Value

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
	if err := client.Mail(srv.appConfigService.GetDbConfig().SmtpFrom.Value, nil); err != nil {
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
	_, err = io.Copy(w, strings.NewReader(c.String()))
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
