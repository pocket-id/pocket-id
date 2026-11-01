package session

import (
	"context"
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

func newServiceForTest(t *testing.T) *Service {
	t.Helper()

	var service *Service
	testutils.NewActorHostForTest(t, func(t *testing.T, h *local.Host) {
		module, err := New(Dependencies{Actors: h, IPLocator: fakeIPLocator{}})
		require.NoError(t, err)
		service = module.service
	})
	return service
}

func TestValidate(t *testing.T) {
	service := newServiceForTest(t)

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
	service := newServiceForTest(t)

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
		otherToken, _ := createSession("revoke-all")

		require.NoError(t, service.RevokeAll(t.Context(), "revoke-all", kept.ID))

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
