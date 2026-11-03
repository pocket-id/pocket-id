package session

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/italypaleale/francis/host/local"
	"github.com/stretchr/testify/require"

	"github.com/pocket-id/pocket-id/backend/internal/apperror"
	testutils "github.com/pocket-id/pocket-id/backend/internal/utils/testing"
)

type fakeIPLocator struct{}

func (fakeIPLocator) GetLocationByIP(context.Context, string) (string, string, error) {
	return "Switzerland", "Zurich", nil
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

func newServiceWithNotifier(t *testing.T, notifier EndNotifier) *Service {
	t.Helper()

	var service *Service
	testutils.NewActorHostForTest(t, func(t *testing.T, h *local.Host) {
		module, err := New(Dependencies{Actors: h, IPLocator: fakeIPLocator{}, EndNotifier: notifier})
		require.NoError(t, err)
		service = module.service
	})
	return service
}

func TestValidate(t *testing.T) {
	service, _ := newServiceForTest(t)

	token, err := service.Create(t.Context(), "user-1", "phr", "198.51.100.1", "test-agent", time.Hour)
	require.NoError(t, err)
	expiredToken, err := service.Create(t.Context(), "user-1", "phr", "198.51.100.1", "test-agent", -time.Minute)
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

func TestRevoke(t *testing.T) {
	service, ended := newServiceForTest(t)

	createSession := func(userID string) (string, Session) {
		token, err := service.Create(t.Context(), userID, "phr", "", "", time.Hour)
		require.NoError(t, err)
		_, session, err := service.Validate(t.Context(), token)
		require.NoError(t, err)
		return token, session
	}

	t.Run("signs out of one session", func(t *testing.T) {
		token, session := createSession("revoke-one")
		otherToken, _ := createSession("revoke-one")

		require.NoError(t, service.Revoke(t.Context(), "revoke-one", session.ID))
		require.Equal(t, []string{"ended", "revoke-one", session.ID}, ended.next(t))

		_, _, err := service.Validate(t.Context(), token)
		require.True(t, apperror.IsCode(err, apperror.CodeNotSignedIn))
		_, _, err = service.Validate(t.Context(), otherToken)
		require.NoError(t, err)
	})

	t.Run("reports an unknown session", func(t *testing.T) {
		_, session := createSession("revoke-unknown")

		err := service.Revoke(t.Context(), "another-user", session.ID)
		require.True(t, apperror.IsCode(err, apperror.CodeNotFound))
	})

	t.Run("signs out everywhere except the kept session", func(t *testing.T) {
		keptToken, kept := createSession("revoke-all")
		otherToken, other := createSession("revoke-all")

		require.NoError(t, service.RevokeAll(t.Context(), "revoke-all", kept.ID))
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

func TestSignOutEverywhere(t *testing.T) {
	service, ended := newServiceForTest(t)

	token, err := service.Create(t.Context(), "everywhere", "phr", "", "", time.Hour)
	require.NoError(t, err)
	_, sessionID, _, _ := parseToken(token)

	require.NoError(t, service.SignOutEverywhere(t.Context(), "everywhere"))
	require.Equal(t, []string{"ended", "everywhere", sessionID}, ended.next(t))
	require.Equal(t, []string{"everywhere", "everywhere"}, ended.next(t))
}

func TestExpiry(t *testing.T) {
	t.Run("reports the expired session", func(t *testing.T) {
		service, ended := newServiceForTest(t)

		token, err := service.Create(t.Context(), "expiring", "phr", "", "", -time.Minute)
		require.NoError(t, err)
		_, sessionID, _, _ := parseToken(token)

		require.Equal(t, []string{"expired", "expiring", sessionID}, ended.next(t))
	})

	t.Run("retries a failed cleanup", func(t *testing.T) {
		recorder := &failingOnceRecorder{endRecorder: make(endRecorder, 16)}
		service := newServiceWithNotifier(t, recorder)

		token, err := service.Create(t.Context(), "expiring", "phr", "", "", -time.Minute)
		require.NoError(t, err)
		_, sessionID, _, _ := parseToken(token)

		require.Equal(t, []string{"expired", "expiring", sessionID}, recorder.next(t))
		require.Equal(t, []string{"expired", "expiring", sessionID}, recorder.next(t))
	})

	t.Run("doesn't fire for a revoked session", func(t *testing.T) {
		service, ended := newServiceForTest(t)

		token, err := service.Create(t.Context(), "revoked", "phr", "", "", time.Second)
		require.NoError(t, err)
		_, sessionID, _, _ := parseToken(token)
		require.NoError(t, service.Revoke(t.Context(), "revoked", sessionID))
		require.Equal(t, []string{"ended", "revoked", sessionID}, ended.next(t))

		select {
		case event := <-ended:
			t.Fatalf("unexpected event after the session was revoked: %v", event)
		case <-time.After(2 * time.Second):
		}
	})
}
