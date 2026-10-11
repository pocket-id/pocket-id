package emailverification

import (
	"context"
	"errors"
	"net/url"
	"testing"
	"time"

	"github.com/italypaleale/francis/actor"
	"github.com/italypaleale/francis/host/local"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/appconfig"
	"github.com/pocket-id/pocket-id/backend/internal/apperror"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
	testutils "github.com/pocket-id/pocket-id/backend/internal/utils/testing"
)

type testEmailSender struct {
	err  error
	sent []sentVerificationEmail
}

type sentVerificationEmail struct {
	userFullName     string
	userEmail        string
	verificationLink string
}

func (s *testEmailSender) SendEmailVerification(_ context.Context, _ *appconfig.AppConfigModel, userFullName, userEmail, verificationLink string) error {
	if s.err != nil {
		return s.err
	}

	s.sent = append(s.sent, sentVerificationEmail{
		userFullName:     userFullName,
		userEmail:        userEmail,
		verificationLink: verificationLink,
	})
	return nil
}

func newServiceForTest(t *testing.T, emailSender *testEmailSender) (*Service, *local.Host, *gorm.DB) {
	t.Helper()

	db := testutils.NewDatabaseForTest(t)

	var service *Service
	host := testutils.NewActorHostForTest(t, func(t *testing.T, host *local.Host) {
		require.NoError(t, host.RegisterActor(ActorType, NewActor))
		service = newService(db, host.Service(), emailSender, nil, "https://id.example.test")
	})
	require.NotNil(t, service)

	return service, host, db
}

func createTestUser(t *testing.T, db *gorm.DB, userID, address string) model.User {
	t.Helper()

	user := model.User{
		Base:      model.Base{ID: userID},
		Username:  userID,
		Email:     &address,
		FirstName: "Test",
		LastName:  "User",
	}
	require.NoError(t, db.Create(&user).Error)

	return user
}

func verificationTokenFromEmail(t *testing.T, sentEmail sentVerificationEmail) string {
	t.Helper()

	verificationURL, err := url.Parse(sentEmail.verificationLink)
	require.NoError(t, err)

	token := verificationURL.Query().Get("token")
	require.NotEmpty(t, token)
	return token
}

func TestSendBindsAddressAndReplacesOutstandingToken(t *testing.T) {
	emailSender := &testEmailSender{}
	service, host, db := newServiceForTest(t, emailSender)
	user := createTestUser(t, db, "user-1", "user@example.test")

	require.NoError(t, service.Send(t.Context(), &appconfig.AppConfigModel{}, user.ID))
	firstToken := verificationTokenFromEmail(t, emailSender.sent[0])
	require.NoError(t, service.Send(t.Context(), &appconfig.AppConfigModel{}, user.ID))
	secondToken := verificationTokenFromEmail(t, emailSender.sent[1])

	var state State
	require.NoError(t, host.GetState(t.Context(), ActorType, user.ID, &state))
	require.Equal(t, "user@example.test", state.Email)
	require.Equal(t, utils.CreateSha256Hash(secondToken), state.TokenHash)
	require.NotEqual(t, firstToken, secondToken)
	require.Len(t, emailSender.sent, 2)
}

func TestVerifyConsumesTokenAndMarksBoundAddressVerified(t *testing.T) {
	emailSender := &testEmailSender{}
	service, host, db := newServiceForTest(t, emailSender)
	user := createTestUser(t, db, "user-2", "user@example.test")

	require.NoError(t, service.Send(t.Context(), &appconfig.AppConfigModel{}, user.ID))
	token := verificationTokenFromEmail(t, emailSender.sent[0])
	require.NoError(t, service.Verify(t.Context(), user.ID, token))

	var updated model.User
	require.NoError(t, db.Where("id = ?", user.ID).First(&updated).Error)
	require.True(t, updated.EmailVerified)

	var state State
	require.ErrorIs(t, host.GetState(t.Context(), ActorType, user.ID, &state), actor.ErrStateNotFound)
}

func TestVerifyClearsSignupDeadline(t *testing.T) {
	emailSender := &testEmailSender{}
	service, _, db := newServiceForTest(t, emailSender)
	user := createTestUser(t, db, "user-deadline", "user@example.test")
	require.NoError(t, db.Model(&user).Update("email_verification_deadline", datatype.DateTime(time.Now().Add(time.Hour))).Error)

	require.NoError(t, service.Send(t.Context(), &appconfig.AppConfigModel{}, user.ID))
	token := verificationTokenFromEmail(t, emailSender.sent[0])
	require.NoError(t, service.Verify(t.Context(), user.ID, token))

	var updated model.User
	require.NoError(t, db.Where("id = ?", user.ID).First(&updated).Error)
	require.True(t, updated.EmailVerified)
	require.Nil(t, updated.EmailVerificationDeadline)
}

func TestVerifyRejectsTokenAfterAddressChanges(t *testing.T) {
	emailSender := &testEmailSender{}
	service, host, db := newServiceForTest(t, emailSender)
	user := createTestUser(t, db, "user-3", "attacker-controlled@example.test")

	require.NoError(t, service.Send(t.Context(), &appconfig.AppConfigModel{}, user.ID))
	token := verificationTokenFromEmail(t, emailSender.sent[0])
	require.NoError(t, db.Model(&model.User{}).Where("id = ?", user.ID).Updates(map[string]any{
		"email":          "victim@example.test",
		"email_verified": false,
	}).Error)

	err := service.Verify(t.Context(), user.ID, token)
	require.True(t, apperror.IsCode(err, apperror.CodeEmailVerificationTokenInvalid))

	var updated model.User
	require.NoError(t, db.Where("id = ?", user.ID).First(&updated).Error)
	require.Equal(t, "victim@example.test", *updated.Email)
	require.False(t, updated.EmailVerified)

	var state State
	require.ErrorIs(t, host.GetState(t.Context(), ActorType, user.ID, &state), actor.ErrStateNotFound)
}

func TestVerifyDoesNotConsumeStateForWrongToken(t *testing.T) {
	emailSender := &testEmailSender{}
	service, host, db := newServiceForTest(t, emailSender)
	user := createTestUser(t, db, "user-4", "user@example.test")

	require.NoError(t, service.Send(t.Context(), &appconfig.AppConfigModel{}, user.ID))

	err := service.Verify(t.Context(), user.ID, "wrong-verification-code")
	require.True(t, apperror.IsCode(err, apperror.CodeEmailVerificationTokenInvalid))

	var state State
	require.NoError(t, host.GetState(t.Context(), ActorType, user.ID, &state))
	require.NotEmpty(t, state.TokenHash)
}

func TestVerifyRejectsExpiredToken(t *testing.T) {
	emailSender := &testEmailSender{}
	service, host, db := newServiceForTest(t, emailSender)
	user := createTestUser(t, db, "user-expired", "user@example.test")
	token := "expired-verification-token"

	require.NoError(t, host.SetState(t.Context(), ActorType, user.ID, State{
		TokenHash: utils.CreateSha256Hash(token),
		Email:     *user.Email,
		ExpiresAt: time.Now().Add(time.Hour),
	}, &actor.SetStateOpts{TTL: time.Millisecond}))
	require.Eventually(t, func() bool {
		var state State
		return errors.Is(host.GetState(t.Context(), ActorType, user.ID, &state), actor.ErrStateNotFound)
	}, time.Second, time.Millisecond)

	err := service.Verify(t.Context(), user.ID, token)
	require.True(t, apperror.IsCode(err, apperror.CodeEmailVerificationTokenInvalid))

	var updated model.User
	require.NoError(t, db.Where("id = ?", user.ID).First(&updated).Error)
	require.False(t, updated.EmailVerified)
}

func TestVerifyRestoresActorStateAfterDatabaseWriteFailure(t *testing.T) {
	emailSender := &testEmailSender{}
	service, host, db := newServiceForTest(t, emailSender)
	user := createTestUser(t, db, "user-restore", "user@example.test")

	require.NoError(t, service.Send(t.Context(), &appconfig.AppConfigModel{}, user.ID))
	token := verificationTokenFromEmail(t, emailSender.sent[0])

	forcedError := errors.New("forced database write failure")
	require.NoError(t, db.Callback().Update().Before("gorm:update").Register("test:fail-email-verification-update", func(tx *gorm.DB) {
		_ = tx.AddError(forcedError)
	}))

	require.ErrorIs(t, service.Verify(t.Context(), user.ID, token), forcedError)

	var state State
	require.NoError(t, host.GetState(t.Context(), ActorType, user.ID, &state))
	require.Equal(t, utils.CreateSha256Hash(token), state.TokenHash)
	require.Equal(t, *user.Email, state.Email)
}

func TestVerifyPreservesNewActorStateAfterDatabaseWriteFailure(t *testing.T) {
	emailSender := &testEmailSender{}
	service, host, db := newServiceForTest(t, emailSender)
	user := createTestUser(t, db, "user-concurrent-issue", "user@example.test")

	require.NoError(t, service.Send(t.Context(), &appconfig.AppConfigModel{}, user.ID))
	token := verificationTokenFromEmail(t, emailSender.sent[0])
	replacement := State{
		TokenHash: "new-token-hash",
		Email:     *user.Email,
		ExpiresAt: time.Now().Add(time.Hour),
	}

	forcedError := errors.New("forced database write failure")
	require.NoError(t, db.Callback().Update().Before("gorm:update").Register("test:issue-token-before-email-verification-update-fails", func(tx *gorm.DB) {
		_, err := host.Service().Invoke(tx.Statement.Context, ActorType, user.ID, MethodIssue, replacement)
		if err != nil {
			_ = tx.AddError(err)
			return
		}
		_ = tx.AddError(forcedError)
	}))

	require.ErrorIs(t, service.Verify(t.Context(), user.ID, token), forcedError)

	var state State
	require.NoError(t, host.GetState(t.Context(), ActorType, user.ID, &state))
	require.Equal(t, replacement.TokenHash, state.TokenHash)
	require.Equal(t, replacement.Email, state.Email)
	require.True(t, replacement.ExpiresAt.Equal(state.ExpiresAt))
}

func TestSendDiscardsTokenWhenEmailDeliveryFails(t *testing.T) {
	emailSender := &testEmailSender{err: errors.New("delivery failed")}
	service, host, db := newServiceForTest(t, emailSender)
	user := createTestUser(t, db, "user-5", "user@example.test")

	require.ErrorContains(t, service.Send(t.Context(), &appconfig.AppConfigModel{}, user.ID), "delivery failed")

	var state State
	require.ErrorIs(t, host.GetState(t.Context(), ActorType, user.ID, &state), actor.ErrStateNotFound)
}

func TestRequestEmailChangeSendsLinkToNewAddressAndKeepsCurrentEmail(t *testing.T) {
	emailSender := &testEmailSender{}
	service, _, db := newServiceForTest(t, emailSender)
	user := createTestUser(t, db, "user-change", "old@example.test")
	require.NoError(t, db.Model(&user).Update("email_verified", true).Error)

	require.NoError(t, service.RequestEmailChange(t.Context(), &appconfig.AppConfigModel{}, user.ID, "new@example.test"))
	require.Len(t, emailSender.sent, 1)
	require.Equal(t, "new@example.test", emailSender.sent[0].userEmail)

	pending, err := service.PendingEmailChange(t.Context(), user.ID)
	require.NoError(t, err)
	require.NotNil(t, pending)
	require.Equal(t, "new@example.test", pending.Email)

	var unchanged model.User
	require.NoError(t, db.Where("id = ?", user.ID).First(&unchanged).Error)
	require.Equal(t, "old@example.test", *unchanged.Email)
	require.True(t, unchanged.EmailVerified)
}

func TestVerifyEmailChangeReplacesEmail(t *testing.T) {
	emailSender := &testEmailSender{}
	service, _, db := newServiceForTest(t, emailSender)
	user := createTestUser(t, db, "user-confirm", "old@example.test")

	require.NoError(t, service.RequestEmailChange(t.Context(), &appconfig.AppConfigModel{}, user.ID, "new@example.test"))
	token := verificationTokenFromEmail(t, emailSender.sent[0])
	require.NoError(t, service.Verify(t.Context(), user.ID, token))

	var updated model.User
	require.NoError(t, db.Where("id = ?", user.ID).First(&updated).Error)
	require.Equal(t, "new@example.test", *updated.Email)
	require.True(t, updated.EmailVerified)

	pending, err := service.PendingEmailChange(t.Context(), user.ID)
	require.NoError(t, err)
	require.Nil(t, pending)
}

func TestVerifyEmailChangeRejectsTokenAfterEmailChangedElsewhere(t *testing.T) {
	emailSender := &testEmailSender{}
	service, _, db := newServiceForTest(t, emailSender)
	user := createTestUser(t, db, "user-admin-changed", "old@example.test")

	require.NoError(t, service.RequestEmailChange(t.Context(), &appconfig.AppConfigModel{}, user.ID, "new@example.test"))
	token := verificationTokenFromEmail(t, emailSender.sent[0])

	// An admin sets a different email before the user confirms their change, and that one must win
	require.NoError(t, db.Model(&user).Update("email", "admin-set@example.test").Error)

	err := service.Verify(t.Context(), user.ID, token)
	require.True(t, apperror.IsCode(err, apperror.CodeEmailVerificationTokenInvalid))

	var updated model.User
	require.NoError(t, db.Where("id = ?", user.ID).First(&updated).Error)
	require.Equal(t, "admin-set@example.test", *updated.Email)
}

func TestVerifyEmailChangeRejectsAddressTakenInTheMeantime(t *testing.T) {
	emailSender := &testEmailSender{}
	service, _, db := newServiceForTest(t, emailSender)
	user := createTestUser(t, db, "user-race", "old@example.test")

	require.NoError(t, service.RequestEmailChange(t.Context(), &appconfig.AppConfigModel{}, user.ID, "new@example.test"))
	token := verificationTokenFromEmail(t, emailSender.sent[0])
	createTestUser(t, db, "user-taker", "new@example.test")

	err := service.Verify(t.Context(), user.ID, token)
	require.True(t, apperror.IsCode(err, apperror.CodeAlreadyInUse))

	var unchanged model.User
	require.NoError(t, db.Where("id = ?", user.ID).First(&unchanged).Error)
	require.Equal(t, "old@example.test", *unchanged.Email)
}

func TestCancelEmailChangeOnlyDropsPendingChange(t *testing.T) {
	emailSender := &testEmailSender{}
	service, host, db := newServiceForTest(t, emailSender)
	user := createTestUser(t, db, "user-cancel", "old@example.test")

	t.Run("cancels a pending change and invalidates its link", func(t *testing.T) {
		require.NoError(t, service.RequestEmailChange(t.Context(), &appconfig.AppConfigModel{}, user.ID, "new@example.test"))
		token := verificationTokenFromEmail(t, emailSender.sent[len(emailSender.sent)-1])

		require.NoError(t, service.CancelEmailChange(t.Context(), user.ID))

		pending, err := service.PendingEmailChange(t.Context(), user.ID)
		require.NoError(t, err)
		require.Nil(t, pending)
		require.True(t, apperror.IsCode(service.Verify(t.Context(), user.ID, token), apperror.CodeEmailVerificationTokenInvalid))
	})

	t.Run("keeps a verification of the current email", func(t *testing.T) {
		require.NoError(t, service.Send(t.Context(), &appconfig.AppConfigModel{}, user.ID))

		require.NoError(t, service.CancelEmailChange(t.Context(), user.ID))

		var state State
		require.NoError(t, host.GetState(t.Context(), ActorType, user.ID, &state))
		require.Equal(t, "old@example.test", state.Email)
	})
}
