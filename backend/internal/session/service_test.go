package session

import (
	"context"
	"testing"
	"time"

	"github.com/italypaleale/francis/host/local"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/apperror"
	"github.com/pocket-id/pocket-id/backend/internal/auditlogs"
	"github.com/pocket-id/pocket-id/backend/internal/authz"
	testutils "github.com/pocket-id/pocket-id/backend/internal/utils/testing"
)

type fakeIPLocator struct{}

func (fakeIPLocator) GetLocationByIP(context.Context, string) (string, string, error) {
	return "Switzerland", "Zurich", nil
}

type recordedEvent struct {
	event auditlogs.Event
	data  auditlogs.Data
}

type fakeAuditLogger struct {
	events []recordedEvent
}

func (f *fakeAuditLogger) Record(_ context.Context, event auditlogs.Event, _ auditlogs.Origin, _ string, data auditlogs.Data, _ *gorm.DB) {
	f.events = append(f.events, recordedEvent{event: event, data: data})
}

func (f *fakeAuditLogger) DeviceStringFromUserAgent(userAgent string) string {
	return "device of " + userAgent
}

func newServiceForTest(t *testing.T) *Service {
	t.Helper()

	var service *Service
	testutils.NewActorHostForTest(t, func(t *testing.T, h *local.Host) {
		module, err := New(Dependencies{Actors: h, IPLocator: fakeIPLocator{}, AuditLog: &fakeAuditLogger{}})
		require.NoError(t, err)
		service = module.service
	})
	return service
}

func TestValidate(t *testing.T) {
	service := newServiceForTest(t)

	token, err := service.Create(t.Context(), "user-1", "phr", "198.51.100.1", "test-agent", time.Hour, true)
	require.NoError(t, err)
	expiredToken, err := service.Create(t.Context(), "user-1", "phr", "198.51.100.1", "test-agent", -time.Minute, true)
	require.NoError(t, err)

	t.Run("accepts a token of an active session", func(t *testing.T) {
		userID, session, err := service.Validate(t.Context(), token)
		require.NoError(t, err)
		require.Equal(t, "user-1", userID)
		require.Equal(t, "phr", session.AuthenticationMethod)
		require.Equal(t, "Zurich", session.City)
	})

	for name, invalid := range map[string]string{
		"rejects a token with a wrong secret":   token + "x",
		"rejects a token of another user":       "user-2" + token[len("user-1"):],
		"rejects a token of an expired session": expiredToken,
		"rejects a malformed token":             "not-a-session-token",
	} {
		t.Run(name, func(t *testing.T) {
			_, _, err := service.Validate(t.Context(), invalid)
			require.True(t, apperror.IsCode(err, apperror.CodeNotSignedIn), err)
		})
	}
}

func TestSessionEndsAt(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name    string
		session Session
		want    time.Time
	}{
		{
			name:    "remembered session lasts until it expires",
			session: Session{LastActiveAt: now.Add(-24 * time.Hour), ExpiresAt: now.Add(time.Hour)},
			want:    now.Add(time.Hour),
		},
		{
			name:    "unused session ends after its idle timeout",
			session: Session{LastActiveAt: now.Add(-20 * time.Minute), ExpiresAt: now.Add(time.Hour), IdleTimeout: 30 * time.Minute},
			want:    now.Add(10 * time.Minute),
		},
		{
			name:    "active session still ends when it expires",
			session: Session{LastActiveAt: now, ExpiresAt: now.Add(10 * time.Minute), IdleTimeout: 30 * time.Minute},
			want:    now.Add(10 * time.Minute),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, tt.session.endsAt())
		})
	}
}

func TestRevoke(t *testing.T) {
	service := newServiceForTest(t)
	auditLog := service.auditLog.(*fakeAuditLogger)

	createSession := func(userID string) (string, Session) {
		token, err := service.Create(t.Context(), userID, "phr", "", "browser", time.Hour, true)
		require.NoError(t, err)
		_, session, err := service.Validate(t.Context(), token)
		require.NoError(t, err)
		return token, session
	}

	t.Run("signs out of one session", func(t *testing.T) {
		token, session := createSession("revoke-one")
		otherToken, _ := createSession("revoke-one")

		auditLog.events = nil
		require.NoError(t, service.Revoke(t.Context(), "revoke-one", session.ID, auditlogs.Origin{Principal: authz.Principal{UserID: "admin", SessionID: session.ID}}))

		_, _, err := service.Validate(t.Context(), token)
		require.True(t, apperror.IsCode(err, apperror.CodeNotSignedIn))
		_, _, err = service.Validate(t.Context(), otherToken)
		require.NoError(t, err)
		require.Equal(t, []recordedEvent{{event: auditlogs.EventSessionRevoked, data: auditlogs.Data{"device": "device of browser"}}}, auditLog.events)
	})

	t.Run("audits revoking the current session as signing out", func(t *testing.T) {
		_, session := createSession("sign-out")

		auditLog.events = nil
		require.NoError(t, service.Revoke(t.Context(), "sign-out", session.ID, auditlogs.Origin{Principal: authz.Principal{UserID: "sign-out", SessionID: session.ID}}))
		require.Equal(t, []recordedEvent{{event: auditlogs.EventSignedOut}}, auditLog.events)
	})

	t.Run("reports an unknown session", func(t *testing.T) {
		_, session := createSession("revoke-unknown")

		err := service.Revoke(t.Context(), "another-user", session.ID, auditlogs.Origin{})
		require.True(t, apperror.IsCode(err, apperror.CodeNotFound))
	})

	t.Run("signs out everywhere except the kept session", func(t *testing.T) {
		keptToken, kept := createSession("revoke-all")
		otherToken, _ := createSession("revoke-all")

		auditLog.events = nil
		require.NoError(t, service.RevokeAll(t.Context(), "revoke-all", kept.ID, auditlogs.Origin{}))
		require.Equal(t, []recordedEvent{{event: auditlogs.EventSessionRevoked, data: auditlogs.Data{"sessionCount": "1", "exceptCurrent": "true"}}}, auditLog.events)

		_, _, err := service.Validate(t.Context(), keptToken)
		require.NoError(t, err)
		_, _, err = service.Validate(t.Context(), otherToken)
		require.True(t, apperror.IsCode(err, apperror.CodeNotSignedIn))

		sessions, err := service.List(t.Context(), "revoke-all")
		require.NoError(t, err)
		require.Len(t, sessions, 1)
		require.Equal(t, kept.ID, sessions[0].ID)
	})
}

func TestRemember(t *testing.T) {
	service := newServiceForTest(t)

	t.Run("drops the idle timeout of the session", func(t *testing.T) {
		token, err := service.Create(t.Context(), "remember", "phr", "", "", time.Hour, false)
		require.NoError(t, err)
		_, session, err := service.Validate(t.Context(), token)
		require.NoError(t, err)
		require.Equal(t, idleTimeout, session.IdleTimeout)

		remembered, err := service.Remember(t.Context(), "remember", session.ID)
		require.NoError(t, err)
		require.Zero(t, remembered.IdleTimeout)

		_, session, err = service.Validate(t.Context(), token)
		require.NoError(t, err)
		require.Zero(t, session.IdleTimeout)
	})

	t.Run("reports an unknown session", func(t *testing.T) {
		_, err := service.Remember(t.Context(), "remember", "unknown")
		require.True(t, apperror.IsCode(err, apperror.CodeNotFound))
	})
}
