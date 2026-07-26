package usersignup

import (
	"testing"
	"time"

	"github.com/italypaleale/francis/actor"
	"github.com/italypaleale/francis/host/local"
	"github.com/stretchr/testify/require"

	testutils "github.com/pocket-id/pocket-id/backend/internal/utils/testing"
)

// newSignupTokenActorService starts a test actor host with the signup token actor registered and returns its service
func newSignupTokenActorService(t *testing.T) *actor.Service {
	t.Helper()

	var svc *actor.Service
	testutils.NewActorHostForTest(t, func(t *testing.T, h *local.Host) {
		err := h.RegisterActor(SignupTokenActorType, NewSignupTokenActor)
		require.NoError(t, err)
		svc = h.Service()
	})
	require.NotNil(t, svc)

	return svc
}

func createSignupTokenForTest(t *testing.T, svc *actor.Service, token string, state SignupTokenState) {
	t.Helper()
	_, err := svc.Invoke(t.Context(), SignupTokenActorType, token, SignupTokenMethodCreate, state)
	require.NoError(t, err)
}

func consumeSignupTokenForTest(t *testing.T, svc *actor.Service, token string) signupTokenConsumeResponse {
	t.Helper()
	res, err := svc.Invoke(t.Context(), SignupTokenActorType, token, signupTokenMethodConsume, nil)
	require.NoError(t, err)

	var out signupTokenConsumeResponse
	err = res.Decode(&out)
	require.NoError(t, err)

	return out
}

// listSignupTokenIDsForTest returns the actor IDs (that is, the token values) of every stored signup token
func listSignupTokenIDsForTest(t *testing.T, svc *actor.Service) []string {
	t.Helper()
	res, err := svc.ListStates(t.Context(), SignupTokenActorType, nil)
	require.NoError(t, err)

	ids := make([]string, len(res.States))
	for i, st := range res.States {
		ids[i] = st.ActorID
	}

	return ids
}

func TestSignupTokenActorConsume(t *testing.T) {
	svc := newSignupTokenActorService(t)

	createSignupTokenForTest(t, svc, "token-1", SignupTokenState{
		ID:           "id-1",
		ExpiresAt:    time.Now().Add(time.Hour),
		UsageLimit:   1,
		UserGroupIDs: []string{"group-a", "group-b"},
		CreatedAt:    time.Now(),
	})

	// First consume succeeds and returns the token's user groups
	res := consumeSignupTokenForTest(t, svc, "token-1")
	require.Equal(t, signupTokenConsumeOK, res.Status)
	require.Equal(t, []string{"group-a", "group-b"}, res.UserGroupIDs)

	// Second consume fails: the usage limit (1) has been reached
	res = consumeSignupTokenForTest(t, svc, "token-1")
	require.Equal(t, signupTokenConsumeLimitReached, res.Status)
}

func TestSignupTokenActorConsumeNotFound(t *testing.T) {
	svc := newSignupTokenActorService(t)

	res := consumeSignupTokenForTest(t, svc, "does-not-exist")
	require.Equal(t, signupTokenConsumeNotFound, res.Status)
}

// TestSignupTokenActorCreateExpired verifies that a token that has already expired is never stored, since its state TTL would be in the past
func TestSignupTokenActorCreateExpired(t *testing.T) {
	svc := newSignupTokenActorService(t)

	createSignupTokenForTest(t, svc, "token-expired", SignupTokenState{
		ID:         "id-expired",
		ExpiresAt:  time.Now().Add(-time.Minute),
		UsageLimit: 1,
		CreatedAt:  time.Now().Add(-time.Hour),
	})

	require.Empty(t, listSignupTokenIDsForTest(t, svc))
	require.Equal(t, signupTokenConsumeNotFound, consumeSignupTokenForTest(t, svc, "token-expired").Status)
}

func TestSignupTokenActorRelease(t *testing.T) {
	svc := newSignupTokenActorService(t)

	createSignupTokenForTest(t, svc, "token-2", SignupTokenState{
		ID:         "id-2",
		ExpiresAt:  time.Now().Add(time.Hour),
		UsageLimit: 2,
		CreatedAt:  time.Now(),
	})

	// Consume both uses
	require.Equal(t, signupTokenConsumeOK, consumeSignupTokenForTest(t, svc, "token-2").Status)
	require.Equal(t, signupTokenConsumeOK, consumeSignupTokenForTest(t, svc, "token-2").Status)
	require.Equal(t, signupTokenConsumeLimitReached, consumeSignupTokenForTest(t, svc, "token-2").Status)

	// Release one use (compensation)
	_, err := svc.Invoke(t.Context(), SignupTokenActorType, "token-2", signupTokenMethodRelease, nil)
	require.NoError(t, err)

	// Consuming succeeds again now that a use was released
	require.Equal(t, signupTokenConsumeOK, consumeSignupTokenForTest(t, svc, "token-2").Status)
}

func TestSignupTokenActorDelete(t *testing.T) {
	svc := newSignupTokenActorService(t)

	createSignupTokenForTest(t, svc, "token-3", SignupTokenState{
		ID:         "id-3",
		ExpiresAt:  time.Now().Add(time.Hour),
		UsageLimit: 1,
		CreatedAt:  time.Now(),
	})
	require.Equal(t, []string{"token-3"}, listSignupTokenIDsForTest(t, svc))

	_, err := svc.Invoke(t.Context(), SignupTokenActorType, "token-3", SignupTokenMethodDelete, nil)
	require.NoError(t, err)

	require.Empty(t, listSignupTokenIDsForTest(t, svc))

	// The token can no longer be consumed
	require.Equal(t, signupTokenConsumeNotFound, consumeSignupTokenForTest(t, svc, "token-3").Status)

	// Deleting a token that no longer exists is a no-op
	_, err = svc.Invoke(t.Context(), SignupTokenActorType, "token-3", SignupTokenMethodDelete, nil)
	require.NoError(t, err)
}

// TestSignupTokenActorMigrateDoesNotOverwrite verifies that the one-time migration never resets a token that was already migrated and used since
func TestSignupTokenActorMigrateDoesNotOverwrite(t *testing.T) {
	svc := newSignupTokenActorService(t)

	state := SignupTokenState{
		ID:         "id-4",
		ExpiresAt:  time.Now().Add(time.Hour),
		UsageLimit: 2,
		CreatedAt:  time.Now(),
	}
	createSignupTokenForTest(t, svc, "token-4", state)

	// Use the token once
	require.Equal(t, signupTokenConsumeOK, consumeSignupTokenForTest(t, svc, "token-4").Status)

	// Re-running the migration must not reset the usage count
	_, err := svc.Invoke(t.Context(), SignupTokenActorType, "token-4", signupTokenMethodMigrate, state)
	require.NoError(t, err)

	// Only one use is left, so a single consume succeeds and the next one doesn't
	require.Equal(t, signupTokenConsumeOK, consumeSignupTokenForTest(t, svc, "token-4").Status)
	require.Equal(t, signupTokenConsumeLimitReached, consumeSignupTokenForTest(t, svc, "token-4").Status)
}
