package oidc

import (
	"net/url"
	"testing"
	"time"

	"github.com/pocket-id/fosite"
	fositejwt "github.com/pocket-id/fosite/token/jwt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

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
		clientID  = "code-client"
		code      = "auth-code-single-use"
		signature = "auth-code-single-use-signature"
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
	require.NoError(t, store.CreateAuthorizeCodeSession(t.Context(), signature, request))

	// First invalidation wins.
	require.NoError(t, store.InvalidateAuthorizeCodeSession(t.Context(), code, signature))

	// A second invalidation of the now-inactive code fails closed: a racing token request
	// cannot proceed to issue a second set of tokens from the same code, and the client gets invalid_grant.
	require.ErrorIs(t, store.InvalidateAuthorizeCodeSession(t.Context(), code, signature), fosite.ErrInvalidGrant)

	// Reads of the consumed code report it as invalidated so fosite triggers reuse handling.
	_, err := store.GetAuthorizeCodeSession(t.Context(), code, signature, nil)
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

	queryCount := 0
	const queryCounterCallback = "test:count_session_revocation_queries"
	require.NoError(t, db.Callback().Query().Before("gorm:query").Register(queryCounterCallback, func(*gorm.DB) {
		queryCount++
	}))

	require.NoError(t, store.RevokeSessionsByIDTokenHint(t.Context(), userID, clientID, idTokenJTI))
	assert.Equal(t, 1, queryCount)
	require.NoError(t, db.Callback().Query().Remove(queryCounterCallback))

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

func newBrowserSessionRequester(requestID, clientID, subject, sessionID string, scopes ...string) fosite.Requester {
	request := newTestRequester(requestID, clientID, subject, "").(*fosite.Request)
	request.Session.(*Session).SessionID = sessionID
	request.GrantedScope = append(request.GrantedScope, scopes...)
	return request
}

func TestEndBrowserSession(t *testing.T) {
	testutils.ForEachDatabase(t, testEndBrowserSession)
}

func testEndBrowserSession(t *testing.T, db *gorm.DB) {
	store := NewStore(db, nil)

	const (
		userID    = "test-user"
		sessionID = "ended-session"
	)
	for _, clientID := range []string{"client-a", "client-b", "client-c", "client-d"} {
		require.NoError(t, db.Create(&model.OidcClient{Base: model.Base{ID: clientID}, Name: clientID}).Error)
	}

	require.NoError(t, store.CreateRefreshTokenSession(t.Context(), "a-refresh", "a-access", newBrowserSessionRequester("a", "client-a", userID, sessionID)))
	require.NoError(t, store.CreateAccessTokenSession(t.Context(), "a-access", newBrowserSessionRequester("a", "client-a", userID, sessionID)))
	require.NoError(t, store.CreateRefreshTokenSession(t.Context(), "b-refresh", "b-access", newBrowserSessionRequester("b", "client-b", userID, sessionID, "offline_access")))
	require.NoError(t, store.CreateAuthorizeCodeSession(t.Context(), "c-code", newBrowserSessionRequester("c", "client-c", userID, sessionID)))
	require.NoError(t, store.CreateRefreshTokenSession(t.Context(), "other-session-refresh", "", newBrowserSessionRequester("d", "client-d", userID, "other-session")))
	require.NoError(t, store.CreateRefreshTokenSession(t.Context(), "other-user-refresh", "", newBrowserSessionRequester("e", "client-d", "other-user", sessionID)))

	clientIDs, err := EndBrowserSession(t.Context(), db, userID, sessionID)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"client-a", "client-b", "client-c"}, clientIDs)

	var sessions []OAuth2Session
	require.NoError(t, db.Find(&sessions).Error)
	activeByKey := map[string]bool{}
	for _, session := range sessions {
		activeByKey[session.Key] = session.Active
	}
	assert.Equal(t, map[string]bool{
		"a-refresh":             false,
		"b-refresh":             true,
		"c-code":                false,
		"other-session-refresh": true,
		"other-user-refresh":    true,
	}, activeByKey, "the access token of the revoked grant is deleted and the offline_access grant is kept")

	kept, err := store.GetRefreshTokenSession(t.Context(), "b-refresh", nil)
	require.NoError(t, err)
	assert.Empty(t, kept.GetSession().(*Session).SessionID, "the kept grant no longer names the ended session")
}

func TestGrantSessions(t *testing.T) {
	testutils.ForEachDatabase(t, func(t *testing.T, db *gorm.DB) {
		store := NewStore(db, nil)
		for _, clientID := range []string{"client-a", "client-b"} {
			require.NoError(t, db.Create(&model.OidcClient{Base: model.Base{ID: clientID}, Name: clientID}).Error)
		}

		require.NoError(t, store.CreateRefreshTokenSession(t.Context(), "a-1", "", newBrowserSessionRequester("a-1", "client-a", "user-1", "session-1")))
		require.NoError(t, store.CreateRefreshTokenSession(t.Context(), "a-1-rotated", "", newBrowserSessionRequester("a-1", "client-a", "user-1", "session-1")))
		require.NoError(t, store.CreateRefreshTokenSession(t.Context(), "a-2", "", newBrowserSessionRequester("a-2", "client-a", "user-1", "session-2")))
		require.NoError(t, store.CreateRefreshTokenSession(t.Context(), "b-device", "", newBrowserSessionRequester("b-device", "client-b", "user-1", "")))
		require.NoError(t, store.CreateRefreshTokenSession(t.Context(), "b-other-user", "", newBrowserSessionRequester("b-other-user", "client-b", "user-2", "session-3")))

		grantSessions, err := GrantSessions(t.Context(), db, []string{"user-1"}, []string{"client-a", "client-b"})
		require.NoError(t, err)
		assert.ElementsMatch(t, []GrantSession{
			{UserID: "user-1", ClientID: "client-a", SessionID: "session-1"},
			{UserID: "user-1", ClientID: "client-a", SessionID: "session-2"},
		}, grantSessions)
	})
}

func TestRevokeUserGrants(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	store := NewStore(db, nil)
	for _, clientID := range []string{"client-a", "client-b"} {
		require.NoError(t, db.Create(&model.OidcClient{Base: model.Base{ID: clientID}, Name: clientID}).Error)
	}
	require.NoError(t, store.CreateRefreshTokenSession(t.Context(), "a-refresh", "", newBrowserSessionRequester("a", "client-a", "user-1", "session-1")))
	require.NoError(t, store.CreateRefreshTokenSession(t.Context(), "b-offline", "", newBrowserSessionRequester("b", "client-b", "user-1", "", "offline_access")))
	require.NoError(t, store.CreateRefreshTokenSession(t.Context(), "other-user", "", newBrowserSessionRequester("c", "client-a", "user-2", "session-2")))
	require.NoError(t, store.CreateAuthorizeCodeSession(t.Context(), "unexchanged-code", newBrowserSessionRequester("d", "client-b", "user-1", "", "offline_access")))

	require.NoError(t, RevokeUserGrants(t.Context(), db, "user-1"))

	var sessions []OAuth2Session
	require.NoError(t, db.Find(&sessions).Error)
	activeByKey := map[string]bool{}
	for _, session := range sessions {
		activeByKey[session.Key] = session.Active
	}
	assert.Equal(t, map[string]bool{"a-refresh": false, "b-offline": false, "other-user": true, "unexchanged-code": false}, activeByKey)
}
