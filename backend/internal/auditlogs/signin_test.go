package auditlogs

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/pocket-id/pocket-id/backend/internal/appconfig"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	testutils "github.com/pocket-id/pocket-id/backend/internal/utils/testing"
)

type signInLocationResolver struct{}

func (signInLocationResolver) GetLocationByIP(context.Context, string) (string, string, error) {
	return "Switzerland", "Zurich", nil
}

func TestSignInBrowserRecognition(t *testing.T) {
	for _, tt := range []struct {
		name          string
		ip            string
		agent         string
		cookie        string
		otherUser     bool
		previousEvent Event
		enabled       bool
		wantNotify    bool
	}{
		{name: "cookie recognizes changed IPv6 prefix and browser version", ip: "2001:db8:2::1", agent: "browser/2", cookie: "known", enabled: true},
		{name: "missing cookie falls back to exact IP and agent", ip: "2001:db8:1::1", agent: "browser/1", enabled: true},
		{name: "unknown cookie falls back to exact IP and agent", ip: "2001:db8:1::1", agent: "browser/1", cookie: "forged", enabled: true},
		{name: "same IPv6 subnet is not an exact match", ip: "2001:db8:1::2", agent: "browser/1", enabled: true, wantNotify: true},
		{name: "same IP with changed agent is unknown", ip: "2001:db8:1::1", agent: "browser/2", enabled: true, wantNotify: true},
		{name: "forged cookie does not suppress notification", ip: "192.0.2.1", agent: "browser/1", cookie: "forged", enabled: true, wantNotify: true},
		{name: "cookie and history are scoped to user", ip: "2001:db8:1::1", agent: "browser/1", cookie: "known", otherUser: true, enabled: true, wantNotify: true},
		{name: "unrelated audit event does not establish familiarity", ip: "2001:db8:1::1", agent: "browser/1", previousEvent: EventClientAuthorization, enabled: true, wantNotify: true},
		{name: "login code history establishes familiarity", ip: "2001:db8:1::1", agent: "browser/1", previousEvent: EventOneTimeAccessTokenSignIn, enabled: true},
		{name: "QR login history establishes familiarity", ip: "2001:db8:1::1", agent: "browser/1", previousEvent: EventRemoteSignIn, enabled: true},
		{name: "disabled notifications still remember browser", ip: "192.0.2.1", agent: "browser/2"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			db := testutils.NewDatabaseForTest(t)
			s := newService(db, nil, signInLocationResolver{}, nil, browserTokenStub{})
			user := model.User{Base: model.Base{ID: "user"}, Username: "user"}
			require.NoError(t, db.Create(&user).Error)
			historyUserID := user.ID
			if tt.otherUser {
				other := model.User{Base: model.Base{ID: "other"}, Username: "other"}
				require.NoError(t, db.Create(&other).Error)
				historyUserID = other.ID
			}
			event := tt.previousEvent
			if event == "" {
				event = EventSignIn
			}
			_, created := s.Create(t.Context(), event, "2001:db8:1::1", "browser/1", historyUserID, Data{}, db)
			require.True(t, created)
			s.browserTokens = browserTokenStub{userID: historyUserID}

			result := s.CreateSignIn(t.Context(), EventSignIn, tt.ip, tt.agent, user.ID, tt.cookie, db, tt.enabled)
			require.True(t, result.Created)
			require.Equal(t, tt.wantNotify, result.Notify)

			require.Equal(t, "renewed:"+user.ID, result.KnownBrowserToken)
			require.Equal(t, tt.ip, *result.AuditLog.IpAddress)
			require.Equal(t, tt.agent, result.AuditLog.UserAgent)
		})
	}
}

func TestSignInWithoutAddress(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	s := newService(db, nil, signInLocationResolver{}, nil, browserTokenStub{})
	user := model.User{Base: model.Base{ID: "user"}, Username: "user"}
	require.NoError(t, db.Create(&user).Error)
	first := s.CreateSignIn(t.Context(), EventSignIn, "", "browser", user.ID, "", db, true)
	require.True(t, first.Notify)
	require.Nil(t, first.AuditLog.IpAddress)
	second := s.CreateSignIn(t.Context(), EventSignIn, "", "browser", user.ID, "", db, true)
	require.False(t, second.Notify)
}

func TestSignInAuditRollsBackWithLogin(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	s := newService(db, nil, signInLocationResolver{}, nil, browserTokenStub{})
	user := model.User{Base: model.Base{ID: "user"}, Username: "user"}
	require.NoError(t, db.Create(&user).Error)
	tx := db.Begin()
	require.NoError(t, tx.Error)
	result := s.CreateSignIn(t.Context(), EventSignIn, "192.0.2.1", "browser", user.ID, "", tx, true)
	require.True(t, result.Created)
	require.True(t, result.Notify)
	require.NoError(t, tx.Rollback().Error)
	var count int64
	require.NoError(t, db.Model(&AuditLog{}).Count(&count).Error)
	require.Zero(t, count)
}

type notificationConfig struct{ config *appconfig.AppConfigModel }

func (c notificationConfig) GetConfig(context.Context) (*appconfig.AppConfigModel, error) {
	return c.config, nil
}

type loginNotification struct{ recipient, ip, method string }
type loginNotificationSender struct{ sent chan loginNotification }

func (s loginNotificationSender) SendNewLogin(_ context.Context, _ *appconfig.AppConfigModel, _, email, ip, _, _, _, method string, _ time.Time) error {
	s.sent <- loginNotification{recipient: email, ip: ip, method: method}
	return nil
}

func TestSignInNotificationsForEveryMethod(t *testing.T) {
	for _, tt := range []struct {
		event  Event
		method string
	}{
		{EventSignIn, "Passkey"},
		{EventOneTimeAccessTokenSignIn, "One-time code"},
		{EventRemoteSignIn, "Another device (QR code)"},
	} {
		t.Run(tt.method, func(t *testing.T) {
			db := testutils.NewDatabaseForTest(t)
			email := "user@example.test"
			user := model.User{Base: model.Base{ID: "user"}, Username: "user", Email: &email}
			require.NoError(t, db.Create(&user).Error)
			sender := loginNotificationSender{sent: make(chan loginNotification, 1)}
			config := &appconfig.AppConfigModel{EmailLoginNotificationEnabled: "true"}
			service := newService(db, sender, signInLocationResolver{}, notificationConfig{config}, browserTokenStub{})

			// Each successful sign-in method delivers a notification for an unfamiliar browser
			result := service.CreateSignIn(t.Context(), tt.event, "192.0.2.1", "browser", user.ID, "", db, true)
			require.True(t, result.Created)
			require.True(t, result.Notify)
			service.SendSignInNotification(t.Context(), result)
			select {
			case sent := <-sender.sent:
				require.Equal(t, loginNotification{recipient: email, ip: "192.0.2.1", method: tt.method}, sent)
			case <-time.After(5 * time.Second):
				t.Fatal("sign-in notification was not delivered")
			}
		})
	}
}

// browserTokenStub exercises sign-in decisions while JWT validation is covered by the token service tests
type browserTokenStub struct {
	userID  string
	signErr error
}

func (s browserTokenStub) generate(userID string, _ time.Duration) (string, error) {
	if s.signErr != nil {
		return "", s.signErr
	}
	return "renewed:" + userID, nil
}

func (s browserTokenStub) verify(token, userID string) error {
	if token == "renewed:"+userID || (token == "known" && userID == s.userID) {
		return nil
	}
	return errors.New("invalid browser token")
}

func TestSignInBrowserSigningFailurePreservesLogin(t *testing.T) {
	for _, token := range []string{"", "invalid-token", "known"} {
		t.Run(token, func(t *testing.T) {
			db := testutils.NewDatabaseForTest(t)
			s := newService(db, nil, signInLocationResolver{}, nil, browserTokenStub{
				userID: "user", signErr: errors.New("signing failed"),
			})
			user := model.User{Base: model.Base{ID: "user"}, Username: "user"}
			require.NoError(t, db.Create(&user).Error)
			tx := db.Begin()
			require.NoError(t, tx.Error)
			result := s.CreateSignIn(t.Context(), EventSignIn, "192.0.2.1", "browser", user.ID, token, tx, true)
			require.NoError(t, tx.Commit().Error)
			require.True(t, result.Created)
			require.Equal(t, token, result.KnownBrowserToken)
			require.Equal(t, token != "known", result.Notify)
			require.NoError(t, db.First(&AuditLog{}, "id = ?", result.AuditLog.ID).Error)
		})
	}
}
