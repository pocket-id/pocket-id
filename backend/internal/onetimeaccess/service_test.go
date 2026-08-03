package onetimeaccess

import (
	"context"
	"testing"
	"time"

	"github.com/italypaleale/francis/actor"
	"github.com/italypaleale/francis/host/local"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/appconfig"
	"github.com/pocket-id/pocket-id/backend/internal/apperror"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	testutils "github.com/pocket-id/pocket-id/backend/internal/utils/testing"
)

type fakeSigner struct{}

func (fakeSigner) GenerateAccessToken(_ model.User, _ string, _ time.Duration) (string, error) {
	return "access-token", nil
}

type fakeAuditLogger struct {
	events []model.AuditLogEvent
}

func (f *fakeAuditLogger) Create(_ context.Context, event model.AuditLogEvent, _, _, _ string, _ model.AuditLogData, _ *gorm.DB) (model.AuditLog, bool) {
	f.events = append(f.events, event)
	return model.AuditLog{}, true
}

type fakeUserProvider struct {
	db *gorm.DB
}

func (f fakeUserProvider) GetUser(ctx context.Context, userID string) (model.User, error) {
	var user model.User
	err := f.db.WithContext(ctx).Where("id = ?", userID).First(&user).Error
	return user, err
}

type fakeEmailSender struct{}

func (fakeEmailSender) SendOneTimeAccessEmail(_ context.Context, _ *appconfig.AppConfigModel, _, _, _, _, _, _ string) error {
	return nil
}

// newServiceForTest sets up a Service backed by an in-memory test actor host, and returns it together with the host and the audit logger it records into
func newServiceForTest(t *testing.T, db *gorm.DB) (*Service, *local.Host, *fakeAuditLogger) {
	t.Helper()

	auditLog := &fakeAuditLogger{}

	var svc *Service
	host := testutils.NewActorHostForTest(t, func(t *testing.T, h *local.Host) {
		err := h.RegisterActor(TokenActorType, NewTokenActor)
		require.NoError(t, err)

		svc = newService(Dependencies{
			DB:           db,
			Signer:       fakeSigner{},
			AuditLog:     auditLog,
			UserProvider: fakeUserProvider{db: db},
			EmailSender:  fakeEmailSender{},
		}, h.Service())
	})
	require.NotNil(t, svc)

	return svc, host, auditLog
}

func TestExchangeTokenSuccess(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	svc, host, auditLog := newServiceForTest(t, db)

	user := model.User{
		Base:     model.Base{ID: "enabled-user"},
		Username: "enabled-user",
	}
	require.NoError(t, db.Create(&user).Error)

	token, _, err := StoreToken(t.Context(), svc.actorService, user.ID, time.Minute, false)
	require.NoError(t, err)

	dbConfig := appconfig.NewTestConfig(nil)
	exchangedUser, accessToken, err := svc.ExchangeToken(t.Context(), dbConfig, token, "", "1.2.3.4", "test-agent")
	require.NoError(t, err)
	require.Equal(t, user.ID, exchangedUser.ID)
	require.NotEmpty(t, accessToken)

	// The token must have been consumed
	var state TokenState
	err = host.GetState(t.Context(), TokenActorType, token, &state)
	require.ErrorIs(t, err, actor.ErrStateNotFound)

	// A sign-in audit log must have been created
	require.Equal(t, []model.AuditLogEvent{model.AuditLogEventOneTimeAccessTokenSignIn}, auditLog.events)
}

func TestExchangeTokenAcceptsAmbiguousAliases(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	svc, host, _ := newServiceForTest(t, db)

	user := model.User{
		Base:     model.Base{ID: "alias-user"},
		Username: "alias-user",
	}
	require.NoError(t, db.Create(&user).Error)

	const token = "a10bc2"
	require.NoError(t, host.SetState(t.Context(), TokenActorType, token, TokenState{
		UserID:    user.ID,
		ExpiresAt: time.Now().Add(time.Minute),
	}, &actor.SetStateOpts{TTL: time.Minute}))

	dbConfig := appconfig.NewTestConfig(nil)
	exchangedUser, _, err := svc.ExchangeToken(t.Context(), dbConfig, "aIObc2", "", "", "")
	require.NoError(t, err)
	require.Equal(t, user.ID, exchangedUser.ID)
}

func TestExchangeTokenInvalidToken(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	svc, _, _ := newServiceForTest(t, db)

	dbConfig := appconfig.NewTestConfig(nil)
	_, _, err := svc.ExchangeToken(t.Context(), dbConfig, "does-not-exist", "", "", "")

	require.True(t, apperror.IsCode(err, apperror.CodeTokenInvalidOrExpired))
}

func TestExchangeTokenDeviceMismatch(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	svc, host, _ := newServiceForTest(t, db)

	user := model.User{
		Base:     model.Base{ID: "device-user"},
		Username: "device-user",
	}
	require.NoError(t, db.Create(&user).Error)

	// Store a token that requires a device token
	token, deviceToken, err := StoreToken(t.Context(), svc.actorService, user.ID, time.Minute, true)
	require.NoError(t, err)
	require.NotNil(t, deviceToken)

	dbConfig := appconfig.NewTestConfig(nil)
	_, _, err = svc.ExchangeToken(t.Context(), dbConfig, token, "wrong-device-token", "", "")

	require.True(t, apperror.IsCode(err, apperror.CodeDeviceCodeInvalid))

	// The token must not have been consumed on a device-token mismatch
	var state TokenState
	err = host.GetState(t.Context(), TokenActorType, token, &state)
	require.NoError(t, err)
	require.Equal(t, user.ID, state.UserID)
}

func TestExchangeTokenRejectsDisabledUser(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	svc, host, auditLog := newServiceForTest(t, db)

	user := model.User{
		Base:     model.Base{ID: "disabled-user"},
		Username: "disabled-user",
		Disabled: true,
	}
	require.NoError(t, db.Create(&user).Error)

	// Store a one-time access token for the disabled user in the actor state store
	token, _, err := StoreToken(t.Context(), svc.actorService, user.ID, time.Minute, false)
	require.NoError(t, err)

	dbConfig := appconfig.NewTestConfig(nil)
	exchangedUser, accessToken, err := svc.ExchangeToken(t.Context(), dbConfig, token, "", "", "")

	require.True(t, apperror.IsCode(err, apperror.CodeUserDisabled))
	require.Empty(t, exchangedUser.ID)
	require.Empty(t, accessToken)

	// The token must have been restored (not consumed), since the exchange failed because the user is disabled
	var state TokenState
	err = host.GetState(t.Context(), TokenActorType, token, &state)
	require.NoError(t, err)
	require.Equal(t, user.ID, state.UserID)

	require.Empty(t, auditLog.events)
}
