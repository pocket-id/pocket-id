package session

import (
	"context"
	"errors"
	"path/filepath"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/italypaleale/francis/actor"
	"github.com/italypaleale/francis/components/standalone"
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

// endRecorder reports each call as how the sessions ended, the user ID and the session IDs
type endRecorder chan []string

func (r endRecorder) SessionsEnded(_ context.Context, userID string, sessionIDs []string) {
	r <- append([]string{"ended", userID}, sessionIDs...)
}

func (r endRecorder) SessionsExpired(_ context.Context, userID string, sessionIDs []string) error {
	r <- append([]string{"expired", userID}, sessionIDs...)
	return nil
}

func (r endRecorder) SignedOutEverywhere(_ context.Context, userID string) error {
	r <- []string{"everywhere", userID}
	return nil
}

func (r endRecorder) next(t *testing.T) []string {
	t.Helper()
	select {
	case ended := <-r:
		return ended
	case <-time.After(10 * time.Second):
		t.Fatal("no sessions ended")
		return nil
	}
}

// failingOnceRecorder fails its first expiry call, so the expiry alarm has to retry it
type failingOnceRecorder struct {
	endRecorder
	failed atomic.Bool
}

func (r *failingOnceRecorder) SessionsExpired(ctx context.Context, userID string, sessionIDs []string) error {
	_ = r.endRecorder.SessionsExpired(ctx, userID, sessionIDs)
	if r.failed.CompareAndSwap(false, true) {
		return errors.New("transient failure")
	}
	return nil
}

func newServiceForTest(t *testing.T) (*Service, endRecorder) {
	t.Helper()

	ended := make(endRecorder, 16)
	return newServiceWithNotifier(t, ended), ended
}

func newServiceWithNotifier(t *testing.T, notifier EndNotifier, hostOpts ...local.HostOption) *Service {
	t.Helper()

	var service *Service
	testutils.NewActorHostForTest(t, func(t *testing.T, h *local.Host) {
		module, err := New(Dependencies{Actors: h, IPLocator: fakeIPLocator{}, AuditLog: &fakeAuditLogger{}, EndNotifier: notifier})
		require.NoError(t, err)
		service = module.service
	}, hostOpts...)
	return service
}

func TestValidate(t *testing.T) {
	service, _ := newServiceForTest(t)

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
	service, ended := newServiceForTest(t)
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
		require.Equal(t, []string{"ended", "revoke-one", session.ID}, ended.next(t))

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
		require.Equal(t, []string{"ended", "sign-out", session.ID}, ended.next(t))
	})

	t.Run("reports an unknown session", func(t *testing.T) {
		_, session := createSession("revoke-unknown")

		err := service.Revoke(t.Context(), "another-user", session.ID, auditlogs.Origin{})
		require.True(t, apperror.IsCode(err, apperror.CodeNotFound))
	})

	t.Run("signs out everywhere except the kept session", func(t *testing.T) {
		keptToken, kept := createSession("revoke-all")
		otherToken, other := createSession("revoke-all")

		auditLog.events = nil
		require.NoError(t, service.RevokeAll(t.Context(), "revoke-all", kept.ID, auditlogs.Origin{}))
		require.Equal(t, []recordedEvent{{event: auditlogs.EventSessionRevoked, data: auditlogs.Data{"sessionCount": "1", "exceptCurrent": "true"}}}, auditLog.events)
		require.Equal(t, []string{"ended", "revoke-all", other.ID}, ended.next(t))

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
	service, _ := newServiceForTest(t)

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

func TestSignOutEverywhere(t *testing.T) {
	service, ended := newServiceForTest(t)

	token, err := service.Create(t.Context(), "everywhere", "phr", "", "", time.Hour, true)
	require.NoError(t, err)
	_, sessionID, _, _ := parseToken(token)

	require.NoError(t, service.SignOutEverywhere(t.Context(), "everywhere", auditlogs.Origin{}))
	require.Equal(t, []string{"ended", "everywhere", sessionID}, ended.next(t))
	require.Equal(t, []string{"everywhere", "everywhere"}, ended.next(t))
}

func TestExpiry(t *testing.T) {
	t.Run("reports the expired session", func(t *testing.T) {
		service, ended := newServiceForTest(t)

		token, err := service.Create(t.Context(), "expiring", "phr", "", "", -time.Minute, true)
		require.NoError(t, err)
		_, sessionID, _, _ := parseToken(token)

		require.Equal(t, []string{"expired", "expiring", sessionID}, ended.next(t))
	})

	t.Run("retries a failed cleanup", func(t *testing.T) {
		recorder := &failingOnceRecorder{endRecorder: make(endRecorder, 16)}
		service := newServiceWithNotifier(t, recorder)

		token, err := service.Create(t.Context(), "expiring", "phr", "", "", -time.Minute, true)
		require.NoError(t, err)
		_, sessionID, _, _ := parseToken(token)

		require.Equal(t, []string{"expired", "expiring", sessionID}, recorder.next(t))
		require.Equal(t, []string{"expired", "expiring", sessionID}, recorder.next(t))
	})

	t.Run("reports sessions that expire on the SQLite provider", func(t *testing.T) {
		ended := make(endRecorder, 64)
		// The SQLite provider keeps due times in milliseconds, unlike the in-memory one the other tests use
		service := newServiceWithNotifier(t, ended, local.WithStandaloneSQLiteProvider(standalone.StandaloneSQLiteOptions{ConnectionString: "file:" + filepath.Join(t.TempDir(), "actors.db")}))

		const count = 40
		for i := range count {
			_, err := service.Create(t.Context(), "expiring-"+strconv.Itoa(i), "phr", "", "", 200*time.Millisecond+time.Duration(i)*time.Millisecond, true)
			require.NoError(t, err)
		}

		for range count {
			require.Equal(t, "expired", ended.next(t)[0])
		}
	})

	t.Run("waits for a session that is still in use", func(t *testing.T) {
		service, ended := newServiceForTest(t)

		token, err := service.Create(t.Context(), "in-use", "phr", "", "", time.Hour, false)
		require.NoError(t, err)
		_, sessionID, _, _ := parseToken(token)

		// The alarm fires early when the session was used or remembered after it was set
		require.NoError(t, service.actorService.SetAlarm(t.Context(), ActorType, "in-use", sessionID, actor.AlarmProperties{DueTime: time.Now()}))

		select {
		case event := <-ended:
			t.Fatalf("unexpected event for a session in use: %v", event)
		case <-time.After(2 * time.Second):
		}
		_, _, err = service.Validate(t.Context(), token)
		require.NoError(t, err)
	})

	t.Run("doesn't fire for a revoked session", func(t *testing.T) {
		service, ended := newServiceForTest(t)

		token, err := service.Create(t.Context(), "revoked", "phr", "", "", time.Second, true)
		require.NoError(t, err)
		_, sessionID, _, _ := parseToken(token)
		require.NoError(t, service.Revoke(t.Context(), "revoked", sessionID, auditlogs.Origin{}))
		require.Equal(t, []string{"ended", "revoked", sessionID}, ended.next(t))

		select {
		case event := <-ended:
			t.Fatalf("unexpected event after the session was revoked: %v", event)
		case <-time.After(2 * time.Second):
		}
	})
}
