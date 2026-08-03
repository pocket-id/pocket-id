package emailverification

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/italypaleale/francis/actor"
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/appconfig"
	"github.com/pocket-id/pocket-id/backend/internal/apperror"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
)

const tokenLifetime = 24 * time.Hour

type UserProvider interface {
	GetUser(ctx context.Context, userID string) (model.User, error)
}

type EmailSender interface {
	SendEmailVerification(ctx context.Context, dbConfig *appconfig.AppConfigModel, userFullName, userEmail, verificationLink string) error
}

type Service struct {
	db          *gorm.DB
	actors      *actor.Service
	users       UserProvider
	emailSender EmailSender
	appURL      string
}

func newService(db *gorm.DB, actors *actor.Service, users UserProvider, emailSender EmailSender, appURL string) *Service {
	return &Service{
		db:          db,
		actors:      actors,
		users:       users,
		emailSender: emailSender,
		appURL:      appURL,
	}
}

func (s *Service) Send(ctx context.Context, dbConfig *appconfig.AppConfigModel, userID string) error {
	user, err := s.users.GetUser(ctx, userID)
	if err != nil {
		return err
	}
	if user.Email == nil {
		return apperror.UserEmailNotSet()
	}

	token, err := utils.GenerateRandomAlphanumericString(32)
	if err != nil {
		return err
	}

	// Persist the token hash in the email verification actor for this user
	state := State{
		TokenHash: utils.CreateSha256Hash(token),
		Email:     *user.Email,
		ExpiresAt: time.Now().Add(tokenLifetime),
	}
	_, err = s.actors.Invoke(ctx, ActorType, user.ID, MethodIssue, state)
	if err != nil {
		return fmt.Errorf("error issuing email verification token: %w", err)
	}

	// Send the email verification message to the user
	err = s.emailSender.SendEmailVerification(
		ctx,
		dbConfig,
		user.FullName(),
		*user.Email,
		s.appURL+"/verify-email?token="+token,
	)
	if err != nil {
		// If the email delivery fails, discard the token in the actor to avoid leaving a valid token in the system
		s.discardAfterSendFailure(ctx, user.ID, state.TokenHash)
		return err
	}

	return nil
}

func (s *Service) Verify(ctx context.Context, userID, token string) error {
	// Consume the token in the email verification actor for this user
	response, err := s.actors.Invoke(ctx, ActorType, userID, methodConsume, tokenRequest{
		TokenHash: utils.CreateSha256Hash(token),
	})
	if err != nil {
		return fmt.Errorf("error consuming email verification token: %w", err)
	}

	var result consumeResponse
	if response == nil {
		return fmt.Errorf("email verification actor returned an empty response")
	}
	err = response.Decode(&result)
	if err != nil {
		return fmt.Errorf("error decoding email verification actor response: %w", err)
	}
	if result.Status != consumeOK {
		return apperror.InvalidEmailVerificationToken()
	}

	// Update the user's email_verified field in the database
	// We are querying by both user ID and email to ensure that the email has not changed since the token was issued
	update := s.db.
		WithContext(ctx).
		Model(&model.User{}).
		Where("id = ? AND email = ?", userID, result.State.Email).
		Updates(map[string]any{
			"email_verified": true,
			"updated_at":     new(datatype.DateTime(time.Now())),
		})
	if update.Error != nil {
		// If the database update fails, restore the token in the actor to allow the user to retry verification
		s.restoreAfterDatabaseFailure(ctx, userID, result.State)
		return update.Error
	}
	if update.RowsAffected != 1 {
		return apperror.InvalidEmailVerificationToken()
	}

	return nil
}

func (s *Service) discardAfterSendFailure(parentCtx context.Context, userID, tokenHash string) {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(parentCtx), 10*time.Second)
	defer cancel()

	_, err := s.actors.Invoke(ctx, ActorType, userID, methodDiscard, tokenRequest{TokenHash: tokenHash})
	if err != nil {
		slog.ErrorContext(ctx, "Failed to discard email verification token after email delivery failed", slog.Any("error", err))
	}
}

func (s *Service) restoreAfterDatabaseFailure(parentCtx context.Context, userID string, state State) {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(parentCtx), 10*time.Second)
	defer cancel()

	_, err := s.actors.Invoke(ctx, ActorType, userID, methodRestore, state)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to restore email verification token after the database update failed", slog.Any("error", err))
	}
}
