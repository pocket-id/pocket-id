package onetimeaccess

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/italypaleale/francis/actor"
	"github.com/italypaleale/francis/host/local"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/appconfig"
	"github.com/pocket-id/pocket-id/backend/internal/apperror"
	"github.com/pocket-id/pocket-id/backend/internal/auditlogs"
	"github.com/pocket-id/pocket-id/backend/internal/authz"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
	testutils "github.com/pocket-id/pocket-id/backend/internal/utils/testing"
)

type fakeSessionCreator struct{}

func (fakeSessionCreator) Create(_ context.Context, _, _, _, _ string, _ time.Duration, _ bool) (string, error) {
	return "session-token", nil
}

type fakeAuditLogger struct {
	events        []auditlogs.Event
	notifications []auditlogs.SignInResult
}

func (f *fakeAuditLogger) Create(_ context.Context, event auditlogs.Event, _, _, _ string, _ auditlogs.Data, _ *gorm.DB) (auditlogs.AuditLog, bool) {
	f.events = append(f.events, event)
	return auditlogs.AuditLog{}, true
}

func (f *fakeAuditLogger) Record(_ context.Context, event auditlogs.Event, _ auditlogs.Origin, _ string, _ auditlogs.Data, _ *gorm.DB) {
	f.events = append(f.events, event)
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
			Sessions:     fakeSessionCreator{},
			AuditLog:     auditLog,
			UserProvider: fakeUserProvider{db: db},
			EmailSender:  fakeEmailSender{},
		}, h.Service())
	})
	require.NotNil(t, svc)

	return svc, host, auditLog
}

type blockingUserProvider struct {
	release chan struct{}
}

func (p blockingUserProvider) GetUser(context.Context, string) (model.User, error) {
	<-p.release
	return model.User{}, errors.New("released")
}

func TestRequestOneTimeAccessEmailDoesNotWaitForRegisteredUser(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	svc, _, _ := newServiceForTest(t, db)
	release := make(chan struct{})
	t.Cleanup(func() { close(release) })
	svc.userProvider = blockingUserProvider{release: release}
	require.NoError(t, db.Create(&model.User{Username: "registered", Email: new("registered@example.com")}).Error)

	dbConfig := appconfig.NewTestConfig(nil)
	dbConfig.EmailOneTimeAccessAsUnauthenticatedEnabled = "true"
	done := make(chan error)
	go func() {
		_, err := svc.RequestOneTimeAccessEmailAsUnauthenticatedUser(t.Context(), dbConfig, "registered@example.com", "", auditlogs.Origin{})
		done <- err
	}()

	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("the response waited for the registered user to be loaded")
	}
}

func TestGenerateTokenLength(t *testing.T) {
	for _, test := range []struct {
		name       string
		ttl        time.Duration
		wantLength int
	}{
		{name: "short-lived", ttl: 15 * time.Minute, wantLength: shortTokenLength},
		{name: "long-lived", ttl: time.Hour, wantLength: longTokenLength},
	} {
		t.Run(test.name, func(t *testing.T) {
			token, err := generateToken(test.ttl)

			require.NoError(t, err)
			require.Len(t, token, test.wantLength)
		})
	}
}

func TestExchangeTokenSuccess(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	svc, host, auditLog := newServiceForTest(t, db)

	user := model.User{
		Base:     model.Base{ID: "enabled-user"},
		Username: "enabled-user",
	}
	require.NoError(t, db.Create(&user).Error)

	token, err := StoreToken(t.Context(), svc.actorService, user.ID, time.Minute, nil, false)
	require.NoError(t, err)

	dbConfig := appconfig.NewTestConfig(nil)
	dbConfig.EmailLoginNotificationMode = appconfig.LoginNotificationBrowserRecognition
	exchangedUser, accessToken, err := svc.ExchangeToken(t.Context(), dbConfig, token, "", "1.2.3.4", "test-agent", "", true)
	require.NoError(t, err)
	require.Equal(t, user.ID, exchangedUser.ID)
	require.NotEmpty(t, accessToken.SessionToken)
	require.Len(t, auditLog.notifications, 1)
	require.True(t, auditLog.notifications[0].Notify)
	require.Equal(t, auditlogs.EventOneTimeAccessTokenSignIn, auditLog.notifications[0].AuditLog.Event)

	// The token must have been consumed
	var state TokenState
	err = host.GetState(t.Context(), TokenActorType, utils.CreateSha256Hash(token), &state)
	require.ErrorIs(t, err, actor.ErrStateNotFound)

	// A sign-in audit log must have been created
	require.Equal(t, []auditlogs.Event{auditlogs.EventOneTimeAccessTokenSignIn}, auditLog.events)
}

func TestExchangeTokenAcceptsAmbiguousAliases(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	svc, host, _ := newServiceForTest(t, db)

	user := model.User{
		Base:     model.Base{ID: "alias-user"},
		Username: "alias-user",
	}
	require.NoError(t, db.Create(&user).Error)

	require.NoError(t, host.SetState(t.Context(), TokenActorType, utils.CreateSha256Hash("a10bc2"), TokenState{
		UserID:    user.ID,
		ExpiresAt: time.Now().Add(time.Minute),
	}, &actor.SetStateOpts{TTL: time.Minute}))

	dbConfig := appconfig.NewTestConfig(nil)
	exchangedUser, _, err := svc.ExchangeToken(t.Context(), dbConfig, "aIObc2", "", "", "", "", true)
	require.NoError(t, err)
	require.Equal(t, user.ID, exchangedUser.ID)
}

func TestExchangeTokenInvalidToken(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	svc, _, _ := newServiceForTest(t, db)

	dbConfig := appconfig.NewTestConfig(nil)
	_, _, err := svc.ExchangeToken(t.Context(), dbConfig, "does-not-exist", "", "", "", "", true)

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
	token, err := StoreToken(t.Context(), svc.actorService, user.ID, time.Minute, new("device-token"), false)
	require.NoError(t, err)

	dbConfig := appconfig.NewTestConfig(nil)
	_, _, err = svc.ExchangeToken(t.Context(), dbConfig, token, "wrong-device-token", "", "", "", true)

	require.True(t, apperror.IsCode(err, apperror.CodeTokenInvalidOrExpired))

	// The token must not have been consumed on a device-token mismatch
	var state TokenState
	err = host.GetState(t.Context(), TokenActorType, utils.CreateSha256Hash(token), &state)
	require.NoError(t, err)
	require.Equal(t, user.ID, state.UserID)

	// The state store must hold neither the token nor the device token, so reading it doesn't hand out a working login code
	require.Equal(t, utils.CreateSha256Hash("device-token"), *state.DeviceToken)
	err = host.GetState(t.Context(), TokenActorType, token, &TokenState{})
	require.ErrorIs(t, err, actor.ErrStateNotFound)
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
	token, err := StoreToken(t.Context(), svc.actorService, user.ID, time.Minute, nil, false)
	require.NoError(t, err)

	dbConfig := appconfig.NewTestConfig(nil)
	exchangedUser, accessToken, err := svc.ExchangeToken(t.Context(), dbConfig, token, "", "", "", "", true)

	require.True(t, apperror.IsCode(err, apperror.CodeUserDisabled))
	require.Empty(t, exchangedUser.ID)
	require.Empty(t, accessToken)

	// The token must have been restored (not consumed), since the exchange failed because the user is disabled
	var state TokenState
	err = host.GetState(t.Context(), TokenActorType, utils.CreateSha256Hash(token), &state)
	require.NoError(t, err)
	require.Equal(t, user.ID, state.UserID)

	require.Empty(t, auditLog.events)
	require.Empty(t, auditLog.notifications)
}

func (f *fakeAuditLogger) CreateSignIn(ctx context.Context, event auditlogs.Event, ipAddress, userAgent, userID, browserToken string, tx *gorm.DB, mode appconfig.AppConfigValue) auditlogs.SignInResult {
	entry, created := f.Create(ctx, event, ipAddress, userAgent, userID, auditlogs.Data{}, tx)
	entry.Event = event
	return auditlogs.SignInResult{AuditLog: entry, Created: created, Notify: mode != appconfig.LoginNotificationDisabled, KnownBrowserToken: "recognized-browser"}
}

func (f *fakeAuditLogger) SendSignInNotification(_ context.Context, result auditlogs.SignInResult) {
	f.notifications = append(f.notifications, result)
}

func TestLoginCodesForAdminsRequireAdminAccess(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	svc, _, _ := newServiceForTest(t, db)
	admin := model.User{Username: "admin", Email: new("admin@example.com"), IsAdmin: true}
	require.NoError(t, db.Create(&admin).Error)
	member := model.User{Username: "member"}
	require.NoError(t, db.Create(&member).Error)
	loginCodes := auditlogs.Origin{Principal: authz.Principal{Scopes: authz.NewScopeSet(authz.UsersLoginCodes)}}
	dbConfig := appconfig.NewTestConfig(nil)
	dbConfig.EmailOneTimeAccessAsAdminEnabled = "true"

	// Neither a link nor an emailed code may be created for an admin
	_, err := svc.CreateToken(t.Context(), admin.ID, time.Minute, loginCodes)
	require.True(t, apperror.IsCode(err, apperror.CodeForbidden), err)
	err = svc.RequestOneTimeAccessEmailAsAdmin(t.Context(), dbConfig, admin.ID, time.Minute, loginCodes)
	require.True(t, apperror.IsCode(err, apperror.CodeForbidden), err)

	// A code created for a regular user stops working once they become an admin
	token, err := svc.CreateToken(t.Context(), member.ID, time.Minute, loginCodes)
	require.NoError(t, err)
	require.NoError(t, db.Model(&member).Update("is_admin", true).Error)
	_, _, err = svc.ExchangeToken(t.Context(), dbConfig, token, "", "1.2.3.4", "test-agent", "", false)
	require.True(t, apperror.IsCode(err, apperror.CodeTokenInvalidOrExpired), err)
}
