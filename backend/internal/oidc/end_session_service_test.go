package oidc

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"net/url"
	"testing"
	"time"

	"github.com/lestrrat-go/jwx/v3/jwa"
	"github.com/lestrrat-go/jwx/v3/jwt"
	"github.com/stretchr/testify/require"

	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/dto"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	testutils "github.com/pocket-id/pocket-id/backend/internal/utils/testing"
)

// TestLogoutCallbackURL locks the post-logout redirect resolution that replaced the
// deleted callback_url_util helper. An attacker-supplied post_logout_redirect_uri must
// only be honored when it exactly matches one the client registered, otherwise logout
// would be an open redirect.
func TestLogoutCallbackURL(t *testing.T) {
	noURLs := &model.OidcClient{Base: model.Base{ID: "c"}}
	withURLs := &model.OidcClient{Base: model.Base{ID: "c"}, LogoutCallbackURLs: model.UrlList{
		"https://app.example/logout",
		"https://app.example/logout2",
		"https://*.example/logout",
	}}

	t.Run("no configured logout URLs yields no callback", func(t *testing.T) {
		got, err := logoutCallbackURL(noURLs, "")
		require.NoError(t, err)
		require.Empty(t, got)
	})

	t.Run("empty input falls back to first registered URL", func(t *testing.T) {
		got, err := logoutCallbackURL(withURLs, "")
		require.NoError(t, err)
		require.Equal(t, "https://app.example/logout", got)
	})

	t.Run("exact match is honored", func(t *testing.T) {
		got, err := logoutCallbackURL(withURLs, "https://app.example/logout2")
		require.NoError(t, err)
		require.Equal(t, "https://app.example/logout2", got)
	})

	t.Run("wildcard match is honored with the requested URL", func(t *testing.T) {
		got, err := logoutCallbackURL(withURLs, "https://tenant.example/logout")
		require.NoError(t, err)
		require.Equal(t, "https://tenant.example/logout", got)
	})

	t.Run("unregistered URL is rejected (no open redirect)", func(t *testing.T) {
		_, err := logoutCallbackURL(withURLs, "https://evil.example/steal")
		var target *common.OidcInvalidCallbackURLError
		require.ErrorAs(t, err, &target)
	})
}

func TestAppendStateToURL(t *testing.T) {
	require.Equal(t, "https://app.example/cb", appendStateToURL("https://app.example/cb", ""))
	require.Equal(t, "https://app.example/cb?state=xyz", appendStateToURL("https://app.example/cb", "xyz"))

	got := appendStateToURL("https://app.example/cb?foo=bar", "xyz")
	parsed, err := url.Parse(got)
	require.NoError(t, err)
	require.Equal(t, "bar", parsed.Query().Get("foo"))
	require.Equal(t, "xyz", parsed.Query().Get("state"))
}

// TestEndSessionService exercises the RP-initiated logout validation and the session
// revocation it triggers. The ID token hint must be a valid, first-party token whose
// subject identifies the logged-out user and whose audience matches the client; only then
// is the client's registered post-logout URL returned and the user/client sessions revoked.
func TestEndSessionService(t *testing.T) {
	const (
		baseURL  = "https://issuer.example.com"
		userID   = "user-1"
		clientID = "client-1"
		jti      = "id-token-jti"
	)

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	signer := testTokenSigner{key: key}

	type tokenOptions struct {
		issuer      string
		subject     string
		audience    string
		jti         string
		omitSubject bool
		omitJTI     bool
		omitAud     bool
		omitType    bool
	}
	signToken := func(t *testing.T, opts tokenOptions) string {
		t.Helper()
		builder := jwt.NewBuilder().IssuedAt(time.Now())
		if opts.issuer != "" {
			builder = builder.Issuer(opts.issuer)
		}
		if !opts.omitSubject {
			builder = builder.Subject(opts.subject)
		}
		if !opts.omitAud {
			builder = builder.Audience([]string{opts.audience})
		}
		if !opts.omitJTI {
			builder = builder.JwtID(opts.jti)
		}
		if !opts.omitType {
			builder = builder.Claim(common.TokenTypeClaim, idTokenType)
		}
		token, err := builder.Build()
		require.NoError(t, err)
		signed, err := jwt.Sign(token, jwt.WithKey(jwa.ES256(), key))
		require.NoError(t, err)
		return string(signed)
	}

	newService := func(t *testing.T) (*endSessionService, *Store) {
		t.Helper()
		db := testutils.NewDatabaseForTest(t)
		require.NoError(t, db.Create(&model.OidcClient{
			Base:               model.Base{ID: clientID},
			Name:               "Test Client",
			LogoutCallbackURLs: model.UrlList{"https://app.example/logout"},
		}).Error)
		require.NoError(t, db.Create(&model.User{Base: model.Base{ID: userID}, Username: "tim"}).Error)
		require.NoError(t, db.Create(&model.UserAuthorizedOidcClient{UserID: userID, ClientID: clientID}).Error)
		store := NewStore(db, nil)
		return newEndSessionService(db, store, signer, baseURL), store
	}

	validToken := tokenOptions{issuer: baseURL, subject: userID, audience: clientID, jti: jti}

	t.Run("missing id_token_hint is rejected", func(t *testing.T) {
		service, _ := newService(t)
		_, err := service.endSession(t.Context(), dto.OidcLogoutDto{}, userID)
		var target *common.TokenInvalidError
		require.ErrorAs(t, err, &target)
	})

	t.Run("malformed id_token_hint is rejected", func(t *testing.T) {
		service, _ := newService(t)
		_, err := service.endSession(t.Context(), dto.OidcLogoutDto{IdTokenHint: "not-a-jwt"}, userID)
		var target *common.TokenInvalidError
		require.ErrorAs(t, err, &target)
	})

	t.Run("token signed by a foreign key is rejected", func(t *testing.T) {
		service, _ := newService(t)
		otherKey, err := rsa.GenerateKey(rand.Reader, 2048)
		require.NoError(t, err)
		token, err := jwt.NewBuilder().
			Issuer(baseURL).Subject(userID).Audience([]string{clientID}).JwtID(jti).IssuedAt(time.Now()).Build()
		require.NoError(t, err)
		signed, err := jwt.Sign(token, jwt.WithKey(jwa.RS256(), otherKey))
		require.NoError(t, err)

		_, err = service.endSession(t.Context(), dto.OidcLogoutDto{IdTokenHint: string(signed)}, userID)
		var target *common.TokenInvalidError
		require.ErrorAs(t, err, &target)
	})

	t.Run("non-ID token (missing type claim) is rejected as id_token_hint", func(t *testing.T) {
		// A first-party JWT access token of the same user/client (signed with the same key) must
		// not be accepted as an id_token_hint; only genuine ID tokens carry the type claim.
		service, _ := newService(t)
		token := signToken(t, tokenOptions{issuer: baseURL, subject: userID, audience: clientID, jti: jti, omitType: true})
		_, err := service.endSession(t.Context(), dto.OidcLogoutDto{IdTokenHint: token}, userID)
		var target *common.TokenInvalidError
		require.ErrorAs(t, err, &target)
	})

	t.Run("subject not matching the logged-in user is rejected", func(t *testing.T) {
		service, _ := newService(t)
		token := signToken(t, tokenOptions{issuer: baseURL, subject: "someone-else", audience: clientID, jti: jti})
		_, err := service.endSession(t.Context(), dto.OidcLogoutDto{IdTokenHint: token}, userID)
		var target *common.TokenInvalidError
		require.ErrorAs(t, err, &target)
	})

	t.Run("client_id parameter not matching the token audience is rejected", func(t *testing.T) {
		service, _ := newService(t)
		token := signToken(t, validToken)
		_, err := service.endSession(t.Context(), dto.OidcLogoutDto{IdTokenHint: token, ClientId: "different-client"}, userID)
		var target *common.OidcClientIdNotMatchingError
		require.ErrorAs(t, err, &target)
	})

	t.Run("user that never authorized the client is rejected", func(t *testing.T) {
		service, _ := newService(t)
		// A valid token for a user that has no authorization record for the client.
		token := signToken(t, tokenOptions{issuer: baseURL, subject: "ghost-user", audience: clientID, jti: jti})
		_, err := service.endSession(t.Context(), dto.OidcLogoutDto{IdTokenHint: token}, "ghost-user")
		var target *common.OidcMissingAuthorizationError
		require.ErrorAs(t, err, &target)
	})

	t.Run("unregistered post_logout_redirect_uri is rejected", func(t *testing.T) {
		service, _ := newService(t)
		token := signToken(t, validToken)
		_, err := service.endSession(t.Context(), dto.OidcLogoutDto{
			IdTokenHint:           token,
			PostLogoutRedirectUri: "https://evil.example/steal",
		}, userID)
		var target *common.OidcInvalidCallbackURLError
		require.ErrorAs(t, err, &target)
	})

	t.Run("valid logout returns the callback URL and revokes the sessions", func(t *testing.T) {
		service, store := newService(t)

		// Seed an active refresh/access token pair tied to this user, client and ID token.
		require.NoError(t, store.CreateRefreshTokenSession(t.Context(), "rt-sig", "at-sig", newTestRequester("logout-req", clientID, userID, jti)))
		require.NoError(t, store.CreateAccessTokenSession(t.Context(), "at-sig", newTestRequester("logout-req", clientID, userID, jti)))

		token := signToken(t, validToken)
		callback, err := service.endSession(t.Context(), dto.OidcLogoutDto{IdTokenHint: token}, userID)
		require.NoError(t, err)
		require.Equal(t, "https://app.example/logout", callback)

		var refresh OAuth2Session
		require.NoError(t, service.db.First(&refresh, "kind = ? AND key = ?", sessionKindRefreshToken, "rt-sig").Error)
		require.False(t, refresh.Active, "refresh token must be revoked on logout")

		var accessCount int64
		require.NoError(t, service.db.Model(&OAuth2Session{}).
			Where("kind = ? AND key = ?", sessionKindAccessToken, "at-sig").
			Count(&accessCount).Error)
		require.Zero(t, accessCount, "access token must be deleted on logout")
	})

	t.Run("valid logout revokes only the session matching the id_token_hint", func(t *testing.T) {
		service, store := newService(t)

		require.NoError(t, store.CreateRefreshTokenSession(t.Context(), "rt-matching", "at-matching", newTestRequester("matching-req", clientID, userID, jti)))
		require.NoError(t, store.CreateAccessTokenSession(t.Context(), "at-matching", newTestRequester("matching-req", clientID, userID, jti)))
		require.NoError(t, store.CreateRefreshTokenSession(t.Context(), "rt-other", "at-other", newTestRequester("other-req", clientID, userID, "other-id-token-jti")))
		require.NoError(t, store.CreateAccessTokenSession(t.Context(), "at-other", newTestRequester("other-req", clientID, userID, "other-id-token-jti")))

		token := signToken(t, validToken)
		callback, err := service.endSession(t.Context(), dto.OidcLogoutDto{IdTokenHint: token}, userID)
		require.NoError(t, err)
		require.Equal(t, "https://app.example/logout", callback)

		var matchingRefresh OAuth2Session
		require.NoError(t, service.db.First(&matchingRefresh, "kind = ? AND key = ?", sessionKindRefreshToken, "rt-matching").Error)
		require.False(t, matchingRefresh.Active, "refresh token matching id_token_hint must be revoked on logout")

		var otherRefresh OAuth2Session
		require.NoError(t, service.db.First(&otherRefresh, "kind = ? AND key = ?", sessionKindRefreshToken, "rt-other").Error)
		require.True(t, otherRefresh.Active, "unrelated refresh token for the same user/client must remain active")

		var matchingAccessCount int64
		require.NoError(t, service.db.Model(&OAuth2Session{}).
			Where("kind = ? AND key = ?", sessionKindAccessToken, "at-matching").
			Count(&matchingAccessCount).Error)
		require.Zero(t, matchingAccessCount, "access token matching id_token_hint must be deleted on logout")

		var otherAccessCount int64
		require.NoError(t, service.db.Model(&OAuth2Session{}).
			Where("kind = ? AND key = ?", sessionKindAccessToken, "at-other").
			Count(&otherAccessCount).Error)
		require.Equal(t, int64(1), otherAccessCount, "unrelated access token for the same user/client must remain active")
	})

	t.Run("valid logout without UI session derives the user from id_token_hint", func(t *testing.T) {
		service, store := newService(t)

		// The OP browser session is absent, but the ID token hint still identifies the
		// End-User and RP session that should be logged out.
		require.NoError(t, store.CreateRefreshTokenSession(t.Context(), "rt-no-session", "at-no-session", newTestRequester("logout-req", clientID, userID, jti)))
		require.NoError(t, store.CreateAccessTokenSession(t.Context(), "at-no-session", newTestRequester("logout-req", clientID, userID, jti)))

		token := signToken(t, validToken)
		callback, err := service.endSession(t.Context(), dto.OidcLogoutDto{IdTokenHint: token}, "")
		require.NoError(t, err)
		require.Equal(t, "https://app.example/logout", callback)

		var refresh OAuth2Session
		require.NoError(t, service.db.First(&refresh, "kind = ? AND key = ?", sessionKindRefreshToken, "rt-no-session").Error)
		require.False(t, refresh.Active, "refresh token must be revoked on logout")

		var accessCount int64
		require.NoError(t, service.db.Model(&OAuth2Session{}).
			Where("kind = ? AND key = ?", sessionKindAccessToken, "at-no-session").
			Count(&accessCount).Error)
		require.Zero(t, accessCount, "access token must be deleted on logout")
	})

	t.Run("valid logout honors a registered post_logout_redirect_uri", func(t *testing.T) {
		service, _ := newService(t)
		token := signToken(t, validToken)
		callback, err := service.endSession(t.Context(), dto.OidcLogoutDto{
			IdTokenHint:           token,
			PostLogoutRedirectUri: "https://app.example/logout",
		}, userID)
		require.NoError(t, err)
		require.Equal(t, "https://app.example/logout", callback)
	})
}
