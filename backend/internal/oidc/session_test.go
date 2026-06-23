package oidc

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNewAuthenticatedSession(t *testing.T) {
	authenticationTime := time.Date(2026, 6, 16, 10, 0, 0, 0, time.FixedZone("CEST", 2*60*60))
	requestedAt := time.Date(2026, 6, 16, 9, 59, 0, 0, time.FixedZone("CEST", 2*60*60))

	session := NewAuthenticatedSession("user-id", "passkey", authenticationTime, requestedAt)

	require.Equal(t, "user-id", session.Subject)
	require.Equal(t, "user-id", session.Claims.Subject)
	require.Equal(t, "passkey", session.AuthenticationMethod)
	require.Equal(t, authenticationTime.UTC(), session.Claims.AuthTime)
	require.Equal(t, requestedAt.UTC(), session.Claims.RequestedAt)
	require.NotNil(t, session.Claims.Extra)
}

func TestNewAuthenticatedSessionDefaultsTimes(t *testing.T) {
	before := time.Now().UTC()
	session := NewAuthenticatedSession("user-id", "passkey", time.Time{}, time.Time{})
	after := time.Now().UTC()

	require.False(t, session.Claims.AuthTime.IsZero())
	require.False(t, session.Claims.RequestedAt.IsZero())
	require.False(t, session.Claims.AuthTime.Before(before))
	require.False(t, session.Claims.AuthTime.After(after))
	require.Equal(t, session.Claims.AuthTime, session.Claims.RequestedAt)
}
