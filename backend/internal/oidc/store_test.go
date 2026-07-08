package oidc

import (
	"net/url"
	"testing"
	"time"

	"github.com/ory/fosite"
	fositejwt "github.com/ory/fosite/token/jwt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pocket-id/pocket-id/backend/internal/model"
	testutils "github.com/pocket-id/pocket-id/backend/internal/utils/testing"
)

// TestStoreGetPARSessionIsSingleUse verifies that a pushed-authorization request_uri can be
// consumed only once (RFC 9126 §4). The consumption (read + invalidate) must be atomic so
// two concurrent /authorize requests cannot both resolve the same request_uri; the store
// enforces this with a conditional UPDATE, so the second GetPARSession returns ErrNotFound.
func TestStoreGetPARSessionIsSingleUse(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	store := NewStore(db, nil)

	const (
		clientID   = "par-client"
		requestURI = "urn:ietf:params:oauth:request_uri:single-use-test"
	)
	require.NoError(t, db.Create(&model.OidcClient{Base: model.Base{ID: clientID}, Name: "PAR Client"}).Error)

	session := NewEmptySession()
	session.SetExpiresAt(fosite.PushedAuthorizeRequestContext, time.Now().UTC().Add(time.Minute))
	redirectURI, err := url.Parse("https://rp.example.com/callback")
	require.NoError(t, err)

	request := &fosite.AuthorizeRequest{
		Request: fosite.Request{
			ID:             "par-request",
			RequestedAt:    time.Now().UTC(),
			Client:         Client{OidcClient: model.OidcClient{Base: model.Base{ID: clientID}}},
			RequestedScope: fosite.Arguments{"openid"},
			Form:           url.Values{},
			Session:        session,
		},
		RedirectURI:   redirectURI,
		ResponseTypes: fosite.Arguments{"code"},
		State:         "state-value",
	}
	require.NoError(t, store.CreatePARSession(t.Context(), requestURI, request))

	// First consumption resolves the stored request.
	first, err := store.GetPARSession(t.Context(), requestURI)
	require.NoError(t, err)
	require.NotNil(t, first)
	require.Equal(t, clientID, first.GetClient().GetID())

	// A second consumption is rejected: the request_uri is single-use.
	_, err = store.GetPARSession(t.Context(), requestURI)
	require.ErrorIs(t, err, fosite.ErrNotFound)
}

// TestStoreInvalidateAuthorizeCodeSessionIsAtomic verifies that an authorization code can
// be invalidated only once (RFC 6749 §4.1.2). Invalidation is a conditional UPDATE guarded
// on active = true, so a second concurrent token request that already read the code as
// active fails closed (ErrNotFound) instead of minting a second token set from one code.
func TestStoreInvalidateAuthorizeCodeSessionIsAtomic(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	store := NewStore(db, nil)

	const (
		clientID = "code-client"
		code     = "auth-code-single-use"
	)
	require.NoError(t, db.Create(&model.OidcClient{Base: model.Base{ID: clientID}, Name: "Code Client"}).Error)

	session := NewEmptySession()
	session.SetExpiresAt(fosite.AuthorizeCode, time.Now().UTC().Add(time.Minute))
	request := &fosite.Request{
		ID:             "code-request",
		RequestedAt:    time.Now().UTC(),
		Client:         Client{OidcClient: model.OidcClient{Base: model.Base{ID: clientID}}},
		RequestedScope: fosite.Arguments{"openid"},
		Form:           url.Values{},
		Session:        session,
	}
	require.NoError(t, store.CreateAuthorizeCodeSession(t.Context(), code, request))

	// First invalidation wins.
	require.NoError(t, store.InvalidateAuthorizeCodeSession(t.Context(), code))

	// A second invalidation of the now-inactive code fails closed: a racing token request
	// cannot proceed to issue a second set of tokens from the same code.
	require.ErrorIs(t, store.InvalidateAuthorizeCodeSession(t.Context(), code), fosite.ErrNotFound)

	// Reads of the consumed code report it as invalidated so fosite triggers reuse handling.
	_, err := store.GetAuthorizeCodeSession(t.Context(), code, nil)
	require.ErrorIs(t, err, fosite.ErrInvalidatedAuthorizeCode)
}

func TestStoreRevokeSessionsByIDTokenHintRevokesMatchingFositeSessions(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	store := NewStore(db, nil)

	const (
		userID          = "test-user-123"
		clientID        = "test-client-456"
		otherClientID   = "other-client-789"
		idTokenJTI      = "matching-id-token-jti"
		otherIDTokenJTI = "other-id-token-jti"
	)

	require.NoError(t, db.Create(&model.OidcClient{
		Base: model.Base{ID: clientID},
		Name: "Test Client",
	}).Error)
	require.NoError(t, db.Create(&model.OidcClient{
		Base: model.Base{ID: otherClientID},
		Name: "Other Client",
	}).Error)

	require.NoError(t, store.CreateRefreshTokenSession(t.Context(), "matching-refresh", "matching-access", newTestRequester("matching-request", clientID, userID, idTokenJTI)))
	require.NoError(t, store.CreateAccessTokenSession(t.Context(), "matching-access", newTestRequester("matching-request", clientID, userID, idTokenJTI)))
	require.NoError(t, store.CreateRefreshTokenSession(t.Context(), "same-client-different-session", "same-client-different-access", newTestRequester("same-client-request", clientID, userID, otherIDTokenJTI)))
	require.NoError(t, store.CreateAccessTokenSession(t.Context(), "same-client-different-access", newTestRequester("same-client-request", clientID, userID, otherIDTokenJTI)))
	require.NoError(t, store.CreateRefreshTokenSession(t.Context(), "other-client-same-jti", "other-client-access", newTestRequester("other-client-request", otherClientID, userID, idTokenJTI)))
	require.NoError(t, store.CreateAccessTokenSession(t.Context(), "other-client-access", newTestRequester("other-client-request", otherClientID, userID, idTokenJTI)))

	require.NoError(t, store.RevokeSessionsByIDTokenHint(t.Context(), userID, clientID, idTokenJTI))

	var sessions []OAuth2Session
	require.NoError(t, db.Order("key").Find(&sessions).Error)

	activeRefreshByKey := map[string]bool{}
	accessKeys := map[string]bool{}
	for _, session := range sessions {
		switch session.Kind {
		case sessionKindRefreshToken:
			activeRefreshByKey[session.Key] = session.Active
		case sessionKindAccessToken:
			accessKeys[session.Key] = true
		}
	}

	assert.False(t, activeRefreshByKey["matching-refresh"])
	assert.True(t, activeRefreshByKey["same-client-different-session"])
	assert.True(t, activeRefreshByKey["other-client-same-jti"])
	assert.False(t, accessKeys["matching-access"])
	assert.True(t, accessKeys["same-client-different-access"])
	assert.True(t, accessKeys["other-client-access"])
}

func TestStoreRevokeSessionsByIDTokenHintSkipsSessionsWithoutMatchingJTI(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	store := NewStore(db, nil)

	const (
		userID        = "test-user-123"
		clientID      = "test-client-456"
		otherClientID = "other-client-789"
	)

	require.NoError(t, db.Create(&model.OidcClient{
		Base: model.Base{ID: clientID},
		Name: "Test Client",
	}).Error)
	require.NoError(t, db.Create(&model.OidcClient{
		Base: model.Base{ID: otherClientID},
		Name: "Other Client",
	}).Error)

	require.NoError(t, store.CreateRefreshTokenSession(t.Context(), "same-client-refresh", "same-client-access", newTestRequester("same-client-request", clientID, userID, "")))
	require.NoError(t, store.CreateAccessTokenSession(t.Context(), "same-client-access", newTestRequester("same-client-request", clientID, userID, "")))
	require.NoError(t, store.CreateRefreshTokenSession(t.Context(), "other-client-refresh", "other-client-access", newTestRequester("other-client-request", otherClientID, userID, "")))
	require.NoError(t, store.CreateAccessTokenSession(t.Context(), "other-client-access", newTestRequester("other-client-request", otherClientID, userID, "")))

	require.NoError(t, store.RevokeSessionsByIDTokenHint(t.Context(), userID, clientID, "missing-from-stored-session"))

	var sessions []OAuth2Session
	require.NoError(t, db.Order("key").Find(&sessions).Error)

	activeRefreshByKey := map[string]bool{}
	accessKeys := map[string]bool{}
	for _, session := range sessions {
		switch session.Kind {
		case sessionKindRefreshToken:
			activeRefreshByKey[session.Key] = session.Active
		case sessionKindAccessToken:
			accessKeys[session.Key] = true
		}
	}

	assert.True(t, activeRefreshByKey["same-client-refresh"])
	assert.True(t, activeRefreshByKey["other-client-refresh"])
	assert.True(t, accessKeys["same-client-access"])
	assert.True(t, accessKeys["other-client-access"])
}

func newTestRequester(requestID, clientID, subject, idTokenJTI string) fosite.Requester {
	session := NewEmptySession()
	session.Subject = subject
	session.Claims = &fositejwt.IDTokenClaims{
		JTI:         idTokenJTI,
		RequestedAt: time.Now().UTC(),
		Extra:       map[string]any{},
	}

	return &fosite.Request{
		ID:          requestID,
		RequestedAt: time.Now().UTC(),
		Client: Client{
			OidcClient: model.OidcClient{
				Base: model.Base{ID: clientID},
			},
		},
		GrantedScope: fosite.Arguments{"openid"},
		Form:         map[string][]string{},
		Session:      session,
	}
}
