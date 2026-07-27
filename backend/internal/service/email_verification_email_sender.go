package service

import (
	"context"

	"github.com/pocket-id/pocket-id/backend/internal/appconfig"
	"github.com/pocket-id/pocket-id/backend/internal/emailverification"
	"github.com/pocket-id/pocket-id/backend/internal/utils/email"
)

// EmailVerificationEmailSender adapts the email service to the email verification module
type EmailVerificationEmailSender struct {
	emailService *EmailService
}

func NewEmailVerificationEmailSender(emailService *EmailService) *EmailVerificationEmailSender {
	return &EmailVerificationEmailSender{emailService: emailService}
}

// SendEmailVerification implements emailverification.EmailSender
func (s *EmailVerificationEmailSender) SendEmailVerification(ctx context.Context, dbConfig *appconfig.AppConfigModel, to email.Address, data emailverification.EmailData) error {
	return SendEmail(ctx, s.emailService, dbConfig, to, EmailVerificationTemplate, &EmailVerificationTemplateData{
		UserFullName:     data.UserFullName,
		VerificationLink: data.VerificationLink,
	})
}
