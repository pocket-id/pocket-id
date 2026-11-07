package netbird

import (
	"context"
	"encoding/base64"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protowire"
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/appconfig"
	"github.com/pocket-id/pocket-id/backend/internal/auditlogs"
	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	testutils "github.com/pocket-id/pocket-id/backend/internal/utils/testing"
)

const (
	proxySecret = "proxy-secret"
	aliceID     = "6f1e3a52-7c1b-4d2e-9a4f-0b8c5d7e2f10"
	bobID       = "0c2d4e6f-8a1b-4c3d-9e5f-7a9b1c3d5e7f"
)

type fakeSessionCreator struct {
	authenticationMethods []string
}

func (f *fakeSessionCreator) Create(_ context.Context, _, authenticationMethod, _, _ string, _ time.Duration) (string, error) {
	f.authenticationMethods = append(f.authenticationMethods, authenticationMethod)
	return "session-token", nil
}

type fakeAuditLogger struct {
	events []auditlogs.Event
}

func (f *fakeAuditLogger) CreateSignIn(_ context.Context, event auditlogs.Event, _, _, _, _ string, _ *gorm.DB, _ appconfig.AppConfigValue) auditlogs.SignInResult {
	f.events = append(f.events, event)
	return auditlogs.SignInResult{Created: true}
}

func (f *fakeAuditLogger) SendSignInNotification(_ context.Context, _ auditlogs.SignInResult) {}

func newServiceForTest(t *testing.T) (*Service, *fakeSessionCreator, *fakeAuditLogger) {
	t.Helper()

	previous := common.EnvConfig
	t.Cleanup(func() { common.EnvConfig = previous })
	common.EnvConfig.NetBirdProxySecret = proxySecret

	db := testutils.NewDatabaseForTest(t)
	for _, user := range []model.User{
		{Base: model.Base{ID: aliceID}, Username: "alice"},
		{Base: model.Base{ID: bobID}, Username: "bob", Disabled: true},
	} {
		require.NoError(t, db.Create(&user).Error)
	}

	sessions := &fakeSessionCreator{}
	auditLog := &fakeAuditLogger{}
	return &Service{db: db, sessions: sessions, auditLog: auditLog}, sessions, auditLog
}

// encodeDexUserID produces the same encoding as NetBird's EncodeDexUserID
func encodeDexUserID(userID, connectorID string) string {
	buf := protowire.AppendTag(nil, 1, protowire.BytesType)
	buf = protowire.AppendString(buf, userID)
	buf = protowire.AppendTag(buf, 2, protowire.BytesType)
	buf = protowire.AppendString(buf, connectorID)
	return base64.RawStdEncoding.EncodeToString(buf)
}

func TestIdentify(t *testing.T) {
	for _, test := range []struct {
		name          string
		secret        string
		netbirdUserID string
		wantUserID    string
	}{
		{name: "Pocket ID user ID from the proxy", secret: proxySecret, netbirdUserID: aliceID, wantUserID: aliceID},
		{name: "Dex user ID", secret: proxySecret, netbirdUserID: encodeDexUserID(aliceID, "pocket-id"), wantUserID: aliceID},
		{name: "padded Dex user ID", secret: proxySecret, netbirdUserID: encodeDexUserID(aliceID, "pocket-id") + "==", wantUserID: aliceID},
		{name: "wrong proxy secret", secret: "guess", netbirdUserID: aliceID},
		{name: "no proxy secret", netbirdUserID: aliceID},
		{name: "no header", secret: proxySecret},
		{name: "unknown user", secret: proxySecret, netbirdUserID: "4b8a2c6e-1d3f-4a5b-8c7d-9e0f1a2b3c4d"},
		{name: "disabled user", secret: proxySecret, netbirdUserID: bobID},
		{name: "Dex user ID without connector", secret: proxySecret, netbirdUserID: base64.RawStdEncoding.EncodeToString(protowire.AppendString(protowire.AppendTag(nil, 1, protowire.BytesType), aliceID))},
	} {
		t.Run(test.name, func(t *testing.T) {
			svc, _, _ := newServiceForTest(t)

			user, err := svc.Identify(t.Context(), test.secret, test.netbirdUserID)

			if test.wantUserID == "" {
				require.ErrorIs(t, err, errNoIdentity())
				return
			}
			require.NoError(t, err)
			require.Equal(t, test.wantUserID, user.ID)
		})
	}
}

func TestLogin(t *testing.T) {
	svc, sessions, auditLog := newServiceForTest(t)
	dbConfig := appconfig.NewTestConfig(nil)

	user, tokens, err := svc.Login(t.Context(), dbConfig, proxySecret, aliceID, aliceID, "1.2.3.4", "test-agent", "")

	require.NoError(t, err)
	require.Equal(t, aliceID, user.ID)
	require.Equal(t, "session-token", tokens.SessionToken)
	require.Equal(t, []string{AuthenticationMethod}, sessions.authenticationMethods)
	require.Equal(t, []auditlogs.Event{auditlogs.EventNetBirdSignIn}, auditLog.events)
}

func TestLoginRejectsAnotherUserThanConfirmed(t *testing.T) {
	svc, sessions, auditLog := newServiceForTest(t)
	dbConfig := appconfig.NewTestConfig(nil)

	_, _, err := svc.Login(t.Context(), dbConfig, proxySecret, aliceID, bobID, "1.2.3.4", "test-agent", "")

	require.ErrorIs(t, err, errNoIdentity())
	require.Empty(t, sessions.authenticationMethods)
	require.Empty(t, auditLog.events)
}
