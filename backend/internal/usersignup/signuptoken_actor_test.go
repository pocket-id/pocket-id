package usersignup

import (
	"testing"
	"time"

	"github.com/italypaleale/francis/actor"
	"github.com/italypaleale/francis/host/local"
	"github.com/stretchr/testify/require"

	testutils "github.com/pocket-id/pocket-id/backend/internal/utils/testing"
)

// newSignupTokenActorService starts a test actor host with the signup token singleton actor registered and returns its service.
// seed, if not nil, is used as the bootstrap data to migrate tokens into the actor's state.
func newSignupTokenActorService(t *testing.T, seed []storedSignupToken) *actor.Service {
	t.Helper()

	var svc *actor.Service
	testutils.NewActorHostForTest(t, func(t *testing.T, h *local.Host) {
		err := h.RegisterSingletonActor(
			SignupTokenActorType, NewSignupTokenActor,
			local.WithBootstrapData(&signupTokenBootstrap{Tokens: seed}),
			local.WithIdleTimeout(-1),
		)
		require.NoError(t, err)
		svc = h.Service()
	})
	require.NotNil(t, svc)
	return svc
}

func createSignupTokenForTest(t *testing.T, svc *actor.Service, token storedSignupToken) {
	t.Helper()
	_, err := svc.Invoke(t.Context(), SignupTokenActorType, actor.SingletonActorID, signupTokenMethodCreate, token)
	require.NoError(t, err)
}

func consumeSignupTokenForTest(t *testing.T, svc *actor.Service, token string) signupTokenConsumeResponse {
	t.Helper()
	res, err := svc.Invoke(t.Context(), SignupTokenActorType, actor.SingletonActorID, signupTokenMethodConsume, signupTokenConsumeRequest{Token: token})
	require.NoError(t, err)
	var out signupTokenConsumeResponse
	require.NoError(t, res.Decode(&out))
	return out
}

func listSignupTokensForTest(t *testing.T, svc *actor.Service) []storedSignupToken {
	t.Helper()
	res, err := svc.Peek(t.Context(), SignupTokenActorType, actor.SingletonActorID, signupTokenMethodList, nil)
	require.NoError(t, err)
	var out signupTokenListResponse
	require.NoError(t, res.Decode(&out))
	return out.Tokens
}

func TestSignupTokenActorConsume(t *testing.T) {
	svc := newSignupTokenActorService(t, nil)

	createSignupTokenForTest(t, svc, storedSignupToken{
		ID:           "id-1",
		Token:        "token-1",
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
	svc := newSignupTokenActorService(t, nil)

	res := consumeSignupTokenForTest(t, svc, "does-not-exist")
	require.Equal(t, signupTokenConsumeNotFound, res.Status)
}

func TestSignupTokenActorConsumeExpired(t *testing.T) {
	svc := newSignupTokenActorService(t, nil)

	createSignupTokenForTest(t, svc, storedSignupToken{
		ID:         "id-expired",
		Token:      "token-expired",
		ExpiresAt:  time.Now().Add(-time.Minute),
		UsageLimit: 1,
		CreatedAt:  time.Now().Add(-time.Hour),
	})

	res := consumeSignupTokenForTest(t, svc, "token-expired")
	require.Equal(t, signupTokenConsumeExpired, res.Status)
}

func TestSignupTokenActorRelease(t *testing.T) {
	svc := newSignupTokenActorService(t, nil)

	createSignupTokenForTest(t, svc, storedSignupToken{
		ID:         "id-2",
		Token:      "token-2",
		ExpiresAt:  time.Now().Add(time.Hour),
		UsageLimit: 2,
		CreatedAt:  time.Now(),
	})

	// Consume both uses
	require.Equal(t, signupTokenConsumeOK, consumeSignupTokenForTest(t, svc, "token-2").Status)
	require.Equal(t, signupTokenConsumeOK, consumeSignupTokenForTest(t, svc, "token-2").Status)
	require.Equal(t, signupTokenConsumeLimitReached, consumeSignupTokenForTest(t, svc, "token-2").Status)

	// Release one use (compensation)
	_, err := svc.Invoke(t.Context(), SignupTokenActorType, actor.SingletonActorID, signupTokenMethodRelease, signupTokenReleaseRequest{Token: "token-2"})
	require.NoError(t, err)

	// Consuming succeeds again now that a use was released
	require.Equal(t, signupTokenConsumeOK, consumeSignupTokenForTest(t, svc, "token-2").Status)
}

func TestSignupTokenActorDelete(t *testing.T) {
	svc := newSignupTokenActorService(t, nil)

	createSignupTokenForTest(t, svc, storedSignupToken{
		ID:         "id-3",
		Token:      "token-3",
		ExpiresAt:  time.Now().Add(time.Hour),
		UsageLimit: 1,
		CreatedAt:  time.Now(),
	})
	require.Len(t, listSignupTokensForTest(t, svc), 1)

	_, err := svc.Invoke(t.Context(), SignupTokenActorType, actor.SingletonActorID, signupTokenMethodDelete, signupTokenDeleteRequest{ID: "id-3"})
	require.NoError(t, err)

	require.Empty(t, listSignupTokensForTest(t, svc))

	// The token can no longer be consumed
	require.Equal(t, signupTokenConsumeNotFound, consumeSignupTokenForTest(t, svc, "token-3").Status)
}

func TestSignupTokenActorBootstrapMigration(t *testing.T) {
	now := time.Now()
	seed := []storedSignupToken{
		{ID: "valid", Token: "valid-token", ExpiresAt: now.Add(time.Hour), UsageLimit: 1, CreatedAt: now},
		{ID: "expired", Token: "expired-token", ExpiresAt: now.Add(-time.Hour), UsageLimit: 1, CreatedAt: now},
	}
	svc := newSignupTokenActorService(t, seed)

	// The singleton actor bootstraps asynchronously once the host is ready. Wait until the migrated (non-expired) token is available.
	require.Eventually(t, func() bool {
		tokens := listSignupTokensForTest(t, svc)
		return len(tokens) == 1 && tokens[0].Token == "valid-token"
	}, 10*time.Second, 20*time.Millisecond, "signup token actor was not bootstrapped in time")

	// The migrated token can be consumed
	require.Equal(t, signupTokenConsumeOK, consumeSignupTokenForTest(t, svc, "valid-token").Status)
}

func TestSignupTokenStateRemoveExpired(t *testing.T) {
	now := time.Now()
	state := &signupTokenActorState{Tokens: map[string]storedSignupToken{
		"a": {Token: "a", ExpiresAt: now.Add(time.Hour)},
		"b": {Token: "b", ExpiresAt: now.Add(-time.Minute)},
		"c": {Token: "c", ExpiresAt: now.Add(-time.Hour)},
	}}

	removed := state.removeExpired(now)
	require.Equal(t, 2, removed)
	require.Len(t, state.Tokens, 1)
	_, ok := state.Tokens["a"]
	require.True(t, ok)
}

func TestSignupTokenStateEarliestExpiration(t *testing.T) {
	now := time.Now()

	empty := &signupTokenActorState{Tokens: map[string]storedSignupToken{}}
	_, found := empty.earliestExpiration()
	require.False(t, found)

	state := &signupTokenActorState{Tokens: map[string]storedSignupToken{
		"a": {Token: "a", ExpiresAt: now.Add(2 * time.Hour)},
		"b": {Token: "b", ExpiresAt: now.Add(time.Hour)},
		"c": {Token: "c", ExpiresAt: now.Add(3 * time.Hour)},
	}}
	earliest, found := state.earliestExpiration()
	require.True(t, found)
	require.Equal(t, now.Add(time.Hour), earliest)
}
