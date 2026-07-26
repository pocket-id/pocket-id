package service

import (
	"context"

	"github.com/pocket-id/pocket-id/backend/internal/appconfig"
	"github.com/pocket-id/pocket-id/backend/internal/onetimeaccess"
	"github.com/pocket-id/pocket-id/backend/internal/utils/email"
)

// OneTimeAccessEmailSender sends the one-time access email.
// It adapts the email service, which owns the email templates, to the interface the onetimeaccess module depends on.
type OneTimeAccessEmailSender struct {
	emailService *EmailService
}

func NewOneTimeAccessEmailSender(emailService *EmailService) *OneTimeAccessEmailSender {
	return &OneTimeAccessEmailSender{emailService: emailService}
}

// SendOneTimeAccessEmail implements onetimeaccess.EmailSender
func (s *OneTimeAccessEmailSender) SendOneTimeAccessEmail(ctx context.Context, dbConfig *appconfig.AppConfigModel, to email.Address, data onetimeaccess.EmailData) error {
	return SendEmail(ctx, s.emailService, dbConfig, to, OneTimeAccessTemplate, &OneTimeAccessTemplateData{
		Code:              data.Code,
		LoginLink:         data.LoginLink,
		LoginLinkWithCode: data.LoginLinkWithCode,
		ExpirationString:  data.ExpirationString,
	})
}
