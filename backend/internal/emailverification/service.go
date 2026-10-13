package emailverification

import (
	"context"
	"errors"
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

// TokenLifetime is how long a verification link stays valid
const TokenLifetime = 24 * time.Hour

type EmailSender interface {
	SendEmailVerification(ctx context.Context, dbConfig *appconfig.AppConfigModel, userFullName, userEmail, verificationLink string) error
}

// ScimSyncScheduler schedules a SCIM sync after a confirmed email change replaced a user's address
type ScimSyncScheduler interface {
	ScheduleSync(ctx context.Context)
}

type Service struct {
	db          *gorm.DB
	actors      *actor.Service
	emailSender EmailSender
	scimSync    ScimSyncScheduler
	appURL      string
}

func newService(db *gorm.DB, actors *actor.Service, emailSender EmailSender, scimSync ScimSyncScheduler, appURL string) *Service {
	return &Service{
		db:          db,
		actors:      actors,
		emailSender: emailSender,
		scimSync:    scimSync,
		appURL:      appURL,
	}
}

func (s *Service) loadUser(ctx context.Context, userID string) (model.User, error) {
	var user model.User
	err := s.db.
		WithContext(ctx).
		Where("id = ?", userID).
		First(&user).
		Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return model.User{}, apperror.UserNotFound()
	}
	return user, err
}

// Send emails a verification link for the user's current email
func (s *Service) Send(ctx context.Context, dbConfig *appconfig.AppConfigModel, userID string) error {
	user, err := s.loadUser(ctx, userID)
	if err != nil {
		return err
	}
	if user.Email == nil {
		return apperror.UserEmailNotSet()
	}

	return s.issue(ctx, dbConfig, user, State{Email: *user.Email})
}

// RequestEmailChange emails a verification link to a new address, which replaces the user's current email once the link is used
// Until then the current email stays in effect, so a mistyped address never restricts the account
func (s *Service) RequestEmailChange(ctx context.Context, dbConfig *appconfig.AppConfigModel, userID, newEmail string) error {
	user, err := s.loadUser(ctx, userID)
	if err != nil {
		return err
	}
	if user.Email == nil {
		return apperror.UserEmailNotSet()
	}

	return s.issue(ctx, dbConfig, user, State{Email: newEmail, ReplacesEmail: *user.Email})
}

// PendingEmailChange returns the email change the user still has to confirm, or nil if there is none
func (s *Service) PendingEmailChange(ctx context.Context, userID string) (*EmailChange, error) {
	response, err := s.actors.Peek(ctx, ActorType, userID, methodPendingChange, nil)
	if err != nil {
		return nil, fmt.Errorf("error reading pending email change: %w", err)
	}
	if response == nil {
		return nil, errors.New("email verification actor returned an empty response")
	}

	var pending EmailChange
	err = response.Decode(&pending)
	if err != nil {
		return nil, fmt.Errorf("error decoding pending email change: %w", err)
	}
	if pending.Email == "" {
		return nil, nil
	}

	return &pending, nil
}

// CancelEmailChange drops a pending email change, so its verification link stops working
func (s *Service) CancelEmailChange(ctx context.Context, userID string) error {
	_, err := s.actors.Invoke(ctx, ActorType, userID, methodCancelChange, nil)
	if err != nil {
		return fmt.Errorf("error canceling email change: %w", err)
	}
	return nil
}

// issue stores a new verification token for the state's email and emails the verification link to that address
// Issuing replaces any outstanding token, so only the most recent link works
func (s *Service) issue(ctx context.Context, dbConfig *appconfig.AppConfigModel, user model.User, state State) error {
	token, err := utils.GenerateRandomAlphanumericString(32)
	if err != nil {
		return err
	}

	// Persist the token hash in the email verification actor for this user
	state.TokenHash = utils.CreateSha256Hash(token)
	state.ExpiresAt = time.Now().Add(TokenLifetime)
	_, err = s.actors.Invoke(ctx, ActorType, user.ID, MethodIssue, state)
	if err != nil {
		return fmt.Errorf("error issuing email verification token: %w", err)
	}

	// Send the email verification message to the address being verified
	err = s.emailSender.SendEmailVerification(
		ctx,
		dbConfig,
		user.FullName(),
		state.Email,
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
		return errors.New("email verification actor returned an empty response")
	}
	err = response.Decode(&result)
	if err != nil {
		return fmt.Errorf("error decoding email verification actor response: %w", err)
	}
	if result.Status != consumeOK {
		return apperror.InvalidEmailVerificationToken()
	}

	// Mark the email as verified
	// A verified account is no longer deleted for missing the signup verification deadline
	updates := map[string]any{
		"email_verified":              true,
		"email_verification_deadline": nil,
		"updated_at":                  new(datatype.DateTime(time.Now())),
	}
	query := s.db.
		WithContext(ctx).
		Model(&model.User{}).
		Where("id = ?", userID)

	// A token for an email change also moves the new address into place, unless the email was changed another way in the meantime
	// A token for the current email only applies while the email is still the one the token was issued for
	isEmailChange := result.State.ReplacesEmail != ""
	if isEmailChange {
		updates["email"] = result.State.Email
		query = query.Where("email = ?", result.State.ReplacesEmail)
	} else {
		query = query.Where("email = ?", result.State.Email)
	}

	update := query.Updates(updates)
	if errors.Is(update.Error, gorm.ErrDuplicatedKey) {
		// Another account took the new address while the change was waiting for confirmation
		return apperror.AlreadyInUse("email")
	}
	if update.Error != nil {
		// If the database update fails, restore the token in the actor to allow the user to retry verification
		s.restoreAfterDatabaseFailure(ctx, userID, result.State)
		return update.Error
	}
	if update.RowsAffected != 1 {
		return apperror.InvalidEmailVerificationToken()
	}

	// Service providers have to learn about the new address
	if isEmailChange && s.scimSync != nil {
		s.scimSync.ScheduleSync(ctx)
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
