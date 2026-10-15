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
	"github.com/pocket-id/pocket-id/backend/internal/auditlogs"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	testutils "github.com/pocket-id/pocket-id/backend/internal/utils/testing"
)

type fakeSigner struct{}

func (fakeSigner) GenerateAccessToken(_ model.User, _ string, _ time.Duration) (string, error) {
	return "access-token", nil
}

type fakeAuditLogger struct {
	events        []auditlogs.Event
	notifications []auditlogs.SignInResult
}

func (f *fakeAuditLogger) Create(_ context.Context, event auditlogs.Event, _, _, _ string, _ auditlogs.Data, _ *gorm.DB) (auditlogs.AuditLog, bool) {
	f.events = append(f.events, event)
	return auditlogs.AuditLog{}, true
}

type fakeUserProvider struct {
	db *gorm.DB
}

func (f fakeUserProvider) GetUser(ctx context.Context, userID string) (model.User, error) {
	var user model.User
	err := f.db.WithContext(ctx).Where("id = ?", userID).First(&user).Error
	return user, err
}

type sentEmail struct {
	recipient string
	code      string
}

// fakeEmailSender reports every email on the sent channel, because emails are sent from a background goroutine
type fakeEmailSender struct {
	sent chan sentEmail
}

func (f fakeEmailSender) SendOneTimeAccessEmail(_ context.Context, _ *appconfig.AppConfigModel, _, userEmail, code, _, _, _ string) error {
	f.sent <- sentEmail{recipient: userEmail, code: code}
	return nil
}

// waitForEmail returns the next login code email sent by the service
func waitForEmail(t *testing.T, svc *Service) sentEmail {
	t.Helper()

	select {
	case email := <-svc.emailSender.(fakeEmailSender).sent:
		return email
	case <-time.After(5 * time.Second):
		require.Fail(t, "expected a login code email")
		return sentEmail{}
	}
}

// requireNoEmail fails if a login code email arrives within a grace period
// Emails are sent from a background goroutine, so checking the channel immediately would pass even when the email is on its way
func requireNoEmail(t *testing.T, svc *Service) {
	t.Helper()

	select {
	case email := <-svc.emailSender.(fakeEmailSender).sent:
		require.Failf(t, "expected no login code email", "an email was sent to %s", email.recipient)
	case <-time.After(500 * time.Millisecond):
	}
}

// createUserWithEmail stores a user that can request login codes by email
func createUserWithEmail(t *testing.T, db *gorm.DB, isAdmin bool) model.User {
	t.Helper()

	email := "user@example.com"
	user := model.User{
		Base:     model.Base{ID: "email-user"},
		Username: "email-user",
		Email:    &email,
		IsAdmin:  isAdmin,
	}
	require.NoError(t, db.Create(&user).Error)

	return user
}

func configWithEmailLoginCode(mode appconfig.AppConfigValue) *appconfig.AppConfigModel {
	dbConfig := appconfig.NewTestConfig(nil)
	dbConfig.EmailOneTimeAccessAsUnauthenticated = mode
	return dbConfig
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
			EmailSender:  fakeEmailSender{sent: make(chan sentEmail, 1)},
		}, h.Service())
	})
	require.NotNil(t, svc)

	return svc, host, auditLog
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
			token, _, err := generateToken(test.ttl, false)

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

	token, _, err := StoreToken(t.Context(), svc.actorService, user.ID, time.Minute, false)
	require.NoError(t, err)

	dbConfig := appconfig.NewTestConfig(nil)
	dbConfig.EmailLoginNotificationMode = appconfig.LoginNotificationBrowserRecognition
	exchangedUser, accessToken, err := svc.ExchangeToken(t.Context(), dbConfig, token, "", "1.2.3.4", "test-agent", "")
	require.NoError(t, err)
	require.Equal(t, user.ID, exchangedUser.ID)
	require.NotEmpty(t, accessToken.AccessToken)
	require.Len(t, auditLog.notifications, 1)
	require.True(t, auditLog.notifications[0].Notify)
	require.Equal(t, auditlogs.EventOneTimeAccessTokenSignIn, auditLog.notifications[0].AuditLog.Event)

	// The token must have been consumed
	var state TokenState
	err = host.GetState(t.Context(), TokenActorType, token, &state)
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

	const token = "a10bc2"
	require.NoError(t, host.SetState(t.Context(), TokenActorType, token, TokenState{
		UserID:    user.ID,
		ExpiresAt: time.Now().Add(time.Minute),
	}, &actor.SetStateOpts{TTL: time.Minute}))

	dbConfig := appconfig.NewTestConfig(nil)
	exchangedUser, _, err := svc.ExchangeToken(t.Context(), dbConfig, "aIObc2", "", "", "", "")
	require.NoError(t, err)
	require.Equal(t, user.ID, exchangedUser.ID)
}

func TestExchangeTokenInvalidToken(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	svc, _, _ := newServiceForTest(t, db)

	dbConfig := appconfig.NewTestConfig(nil)
	_, _, err := svc.ExchangeToken(t.Context(), dbConfig, "does-not-exist", "", "", "", "")

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
	_, _, err = svc.ExchangeToken(t.Context(), dbConfig, token, "wrong-device-token", "", "", "")

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
	exchangedUser, accessToken, err := svc.ExchangeToken(t.Context(), dbConfig, token, "", "", "", "")

	require.True(t, apperror.IsCode(err, apperror.CodeUserDisabled))
	require.Empty(t, exchangedUser.ID)
	require.Empty(t, accessToken)

	// The token must have been restored (not consumed), since the exchange failed because the user is disabled
	var state TokenState
	err = host.GetState(t.Context(), TokenActorType, token, &state)
	require.NoError(t, err)
	require.Equal(t, user.ID, state.UserID)

	require.Empty(t, auditLog.events)
	require.Empty(t, auditLog.notifications)
}

func TestRequestOneTimeAccessEmailAsUnauthenticatedUser(t *testing.T) {
	for _, test := range []struct {
		name         string
		mode         appconfig.AppConfigValue
		isAdmin      bool
		wantDisabled bool
		wantSignIn   bool
	}{
		{name: "disabled rejects requests", mode: appconfig.EmailOneTimeAccessAsUnauthenticatedDisabled, wantDisabled: true},
		{name: "unknown mode fails closed", mode: "true", wantDisabled: true},
		{name: "non-admins mode lets users sign in", mode: appconfig.EmailOneTimeAccessAsUnauthenticatedNonAdmins, wantSignIn: true},
		{name: "non-admins mode silently excludes admins", mode: appconfig.EmailOneTimeAccessAsUnauthenticatedNonAdmins, isAdmin: true},
		{name: "all mode lets admins sign in", mode: appconfig.EmailOneTimeAccessAsUnauthenticatedAll, isAdmin: true, wantSignIn: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			db := testutils.NewDatabaseForTest(t)
			svc, _, _ := newServiceForTest(t, db)
			user := createUserWithEmail(t, db, test.isAdmin)
			dbConfig := configWithEmailLoginCode(test.mode)

			deviceToken, err := svc.RequestOneTimeAccessEmailAsUnauthenticatedUser(t.Context(), dbConfig, *user.Email, "")

			if test.wantDisabled {
				require.True(t, apperror.IsCode(err, apperror.CodeOneTimeAccessDisabled))
				requireNoEmail(t, svc)
				return
			}

			// Excluded users must get a device token that looks like a real one so the response does not reveal admin privileges
			require.NoError(t, err)
			require.Len(t, deviceToken, deviceTokenLength)

			if !test.wantSignIn {
				requireNoEmail(t, svc)
				return
			}

			// The emailed code must sign the user in from the browser that requested it
			email := waitForEmail(t, svc)
			require.Equal(t, *user.Email, email.recipient)
			exchangedUser, accessToken, err := svc.ExchangeToken(t.Context(), dbConfig, email.code, deviceToken, "", "", "")
			require.NoError(t, err)
			require.Equal(t, user.ID, exchangedUser.ID)
			require.NotEmpty(t, accessToken.AccessToken)
		})
	}
}

func TestExchangeTokenRechecksSelfRequestedCodes(t *testing.T) {
	t.Run("user who becomes admin after requesting a code cannot use it", func(t *testing.T) {
		db := testutils.NewDatabaseForTest(t)
		svc, _, auditLog := newServiceForTest(t, db)
		user := createUserWithEmail(t, db, false)
		dbConfig := configWithEmailLoginCode(appconfig.EmailOneTimeAccessAsUnauthenticatedNonAdmins)

		deviceToken, err := svc.RequestOneTimeAccessEmailAsUnauthenticatedUser(t.Context(), dbConfig, *user.Email, "")
		require.NoError(t, err)
		email := waitForEmail(t, svc)

		// Grant admin privileges while the code is still valid
		require.NoError(t, db.Model(&user).Update("is_admin", true).Error)

		_, accessToken, err := svc.ExchangeToken(t.Context(), dbConfig, email.code, deviceToken, "", "", "")
		require.True(t, apperror.IsCode(err, apperror.CodeTokenInvalidOrExpired))
		require.Empty(t, accessToken)
		require.Empty(t, auditLog.events)
	})

	t.Run("code stops working once the setting is disabled", func(t *testing.T) {
		db := testutils.NewDatabaseForTest(t)
		svc, _, auditLog := newServiceForTest(t, db)
		user := createUserWithEmail(t, db, false)

		deviceToken, err := svc.RequestOneTimeAccessEmailAsUnauthenticatedUser(t.Context(), configWithEmailLoginCode(appconfig.EmailOneTimeAccessAsUnauthenticatedNonAdmins), *user.Email, "")
		require.NoError(t, err)
		email := waitForEmail(t, svc)

		_, accessToken, err := svc.ExchangeToken(t.Context(), configWithEmailLoginCode(appconfig.EmailOneTimeAccessAsUnauthenticatedDisabled), email.code, deviceToken, "", "", "")
		require.True(t, apperror.IsCode(err, apperror.CodeTokenInvalidOrExpired))
		require.Empty(t, accessToken)
		require.Empty(t, auditLog.events)
	})

	// Codes created by an admin are the recovery path for admins, so the setting must never block them
	for _, mode := range []appconfig.AppConfigValue{appconfig.EmailOneTimeAccessAsUnauthenticatedNonAdmins, appconfig.EmailOneTimeAccessAsUnauthenticatedDisabled} {
		t.Run("admin can use a code created by an admin when the mode is "+string(mode), func(t *testing.T) {
			db := testutils.NewDatabaseForTest(t)
			svc, _, _ := newServiceForTest(t, db)
			admin := createUserWithEmail(t, db, true)

			code, err := svc.CreateToken(t.Context(), admin.ID, time.Hour)
			require.NoError(t, err)

			exchangedUser, accessToken, err := svc.ExchangeToken(t.Context(), configWithEmailLoginCode(mode), code, "", "", "", "")
			require.NoError(t, err)
			require.Equal(t, admin.ID, exchangedUser.ID)
			require.NotEmpty(t, accessToken.AccessToken)
		})
	}
}

func (f *fakeAuditLogger) CreateSignIn(ctx context.Context, event auditlogs.Event, ipAddress, userAgent, userID, browserToken string, tx *gorm.DB, mode appconfig.AppConfigValue) auditlogs.SignInResult {
	entry, created := f.Create(ctx, event, ipAddress, userAgent, userID, auditlogs.Data{}, tx)
	entry.Event = event
	return auditlogs.SignInResult{AuditLog: entry, Created: created, Notify: mode != appconfig.LoginNotificationDisabled, KnownBrowserToken: "recognized-browser"}
}

func (f *fakeAuditLogger) SendSignInNotification(_ context.Context, result auditlogs.SignInResult) {
	f.notifications = append(f.notifications, result)
}
