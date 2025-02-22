package service

import (
	"bytes"
	"crypto/tls"
	"fmt"
	htemplate "html/template"
	"mime/multipart"
	"mime/quotedprintable"
	"net"
	"net/smtp"
	"net/textproto"
	"os"
	ttemplate "text/template"
	"time"

	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
	"github.com/pocket-id/pocket-id/backend/internal/utils/email"
	"gorm.io/gorm"
)

var netDialer = &net.Dialer{
	Timeout: 3 * time.Second,
}

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
		LogoURL: utils.GetAbsoluteURL(common.EnvConfig.AppURL, "/api/application-configuration/logo"),
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

	// Connect to the SMTP server
	if srv.appConfigService.DbConfig.SmtpTls.Value == "false" {
		client, err = srv.connectToSmtpServer(smtpAddress)
	} else if port == "465" {
		client, err = srv.connectToSmtpServerUsingImplicitTLS(
			smtpAddress,
			tlsConfig,
		)
	} else {
		client, err = srv.connectToSmtpServerUsingStartTLS(
			smtpAddress,
			tlsConfig,
		)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to connect to SMTP server: %w", err)
	}

	// Set up the authentication if user or password are set
	smtpUser := srv.appConfigService.DbConfig.SmtpUser.Value
	smtpPassword := srv.appConfigService.DbConfig.SmtpPassword.Value

	if smtpUser != "" || smtpPassword != "" {
		auth := smtp.PlainAuth("",
			srv.appConfigService.DbConfig.SmtpUser.Value,
			srv.appConfigService.DbConfig.SmtpPassword.Value,
			srv.appConfigService.DbConfig.SmtpHost.Value,
		)
		if err := client.Auth(auth); err != nil {
			return nil, fmt.Errorf("failed to authenticate SMTP client: %w", err)
		}
	}

	return client, err
}

func (srv *EmailService) connectToSmtpServer(serverAddr string) (*smtp.Client, error) {
	conn, err := netDialer.Dial("tcp", serverAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to SMTP server: %w", err)
	}
	client, err := smtp.NewClient(conn, srv.appConfigService.DbConfig.SmtpHost.Value)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to create SMTP client: %w", err)
	}

	if err := srv.sendHelloCommand(client); err != nil {
		return nil, fmt.Errorf("failed to say hello to SMTP server: %w", err)
	}

	return client, err
}

func (srv *EmailService) connectToSmtpServerUsingImplicitTLS(serverAddr string, tlsConfig *tls.Config) (*smtp.Client, error) {
	tlsDialer := &tls.Dialer{
		NetDialer: netDialer,
		Config:    tlsConfig,
	}
	conn, err := tlsDialer.Dial("tcp", serverAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to SMTP server: %w", err)
	}

	client, err := smtp.NewClient(conn, srv.appConfigService.DbConfig.SmtpHost.Value)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to create SMTP client: %w", err)
	}

	if err := srv.sendHelloCommand(client); err != nil {
		return nil, fmt.Errorf("failed to say hello to SMTP server: %w", err)
	}

	return client, nil
}

func (srv *EmailService) connectToSmtpServerUsingStartTLS(serverAddr string, tlsConfig *tls.Config) (*smtp.Client, error) {
	conn, err := netDialer.Dial("tcp", serverAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to SMTP server: %w", err)
	}

	client, err := smtp.NewClient(conn, srv.appConfigService.DbConfig.SmtpHost.Value)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to create SMTP client: %w", err)
	}

	if err := srv.sendHelloCommand(client); err != nil {
		return nil, fmt.Errorf("failed to say hello to SMTP server: %w", err)
	}

	if err := client.StartTLS(tlsConfig); err != nil {
		return nil, fmt.Errorf("failed to start TLS: %w", err)
	}
	return client, nil
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
	if err := client.Mail(srv.appConfigService.DbConfig.SmtpFrom.Value); err != nil {
		return fmt.Errorf("failed to set sender: %w", err)
	}
	if err := client.Rcpt(toEmail.Email); err != nil {
		return fmt.Errorf("failed to set recipient: %w", err)
	}
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("failed to start data: %w", err)
	}
	_, err = w.Write([]byte(c.String()))
	if err != nil {
		return fmt.Errorf("failed to write email data: %w", err)
	}
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
