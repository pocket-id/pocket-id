package oidc

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/json"
	"maps"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ory/fosite"
	"github.com/ory/fosite/compose"
	fositejwt "github.com/ory/fosite/token/jwt"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/model"
	testutils "github.com/pocket-id/pocket-id/backend/internal/utils/testing"
)

// TestTokenHandlerClientCredentialsGrant guards two regressions in one flow:
//   - the resource-owner re-validation must be skipped for client_credentials (which has
//     no user / empty subject), otherwise the grant is rejected with invalid_grant; and
//   - every issued JWT access token must carry an `aud` claim bound to the client
//     (RFC 9068 §2.2), which fosite only emits when the granted audience is non-empty.
//
// The provider-level tests bypass tokenHandler.token by calling NewAccessResponse directly,
// so this drives the real HTTP handler with confidential-client Basic auth.
func TestTokenHandlerClientCredentialsGrant(t *testing.T) {
	gin.SetMode(gin.TestMode)

	const (
		baseURL     = "https://issuer.example.com"
		secret      = "test-secret"
		clientID    = "cc-client"
		clientPlain = "cc-secret-value"
	)

	db := testutils.NewDatabaseForTest(t)
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	hashed, err := bcrypt.GenerateFromPassword([]byte(clientPlain), bcrypt.DefaultCost)
	require.NoError(t, err)
	require.NoError(t, db.Create(&model.OidcClient{
		Base:     model.Base{ID: clientID},
		Name:     "Client Credentials Client",
		Secret:   string(hashed),
		IsPublic: false,
	}).Error)

	provider, err := newProvider(NewStore(db, nil), nil, testTokenSigner{key: key}, Config{
		BaseURL:      baseURL,
		TokenBaseURL: baseURL,
		Secret:       []byte(secret),
	})
	require.NoError(t, err)
	handler := newTokenHandler(provider, newClaimsService(db, nil, baseURL, nil), nil)

	form := url.Values{"grant_type": {"client_credentials"}}
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/oidc/token", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(clientID, clientPlain)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = req
	handler.token(c)

	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.NotEmpty(t, body["access_token"], "client_credentials must issue a token, got error: %v", body["error"])

	claims := decodeJWTPart(t, body["access_token"].(string), 1)
	// With no resource requested, the client_credentials token is a plain token bound to the requesting client
	require.Contains(t, jwtAudience(claims), clientID, "access token must be audience-bound to the client")
}

// TestTokenHandlerClientCredentialsDropsIdentityScopes guards that a machine token never
// carries identity scopes. A client_credentials grant has a synthetic subject and no real
// user, so if it could keep the openid scope it would slip past the userinfo endpoint
// (which gates on openid) and probe for user PII. The handler must strip identity scopes.
func TestTokenHandlerClientCredentialsDropsIdentityScopes(t *testing.T) {
	gin.SetMode(gin.TestMode)

	const (
		baseURL     = "https://issuer.example.com"
		secret      = "test-secret"
		clientID    = "cc-client"
		clientPlain = "cc-secret-value"
	)

	db := testutils.NewDatabaseForTest(t)
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	hashed, err := bcrypt.GenerateFromPassword([]byte(clientPlain), bcrypt.DefaultCost)
	require.NoError(t, err)
	require.NoError(t, db.Create(&model.OidcClient{
		Base:     model.Base{ID: clientID},
		Name:     "Client Credentials Client",
		Secret:   string(hashed),
		IsPublic: false,
	}).Error)

	provider, err := newProvider(NewStore(db, nil), nil, testTokenSigner{key: key}, Config{
		BaseURL:      baseURL,
		TokenBaseURL: baseURL,
		Secret:       []byte(secret),
	})
	require.NoError(t, err)
	handler := newTokenHandler(provider, newClaimsService(db, nil, baseURL, nil), nil)

	form := url.Values{"grant_type": {"client_credentials"}, "scope": {"openid"}}
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/oidc/token", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(clientID, clientPlain)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = req
	handler.token(c)

	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.NotEmpty(t, body["access_token"], "client_credentials must issue a token, got error: %v", body["error"])

	// Introspect the issued token the same way the userinfo endpoint does: openid must be gone.
	_, accessRequest, err := provider.IntrospectToken(t.Context(), body["access_token"].(string), fosite.AccessToken, NewEmptySession())
	require.NoError(t, err)
	require.False(t, accessRequest.GetGrantedScopes().Has("openid"), "client_credentials token must not carry the openid scope")
}

// TestTokenHandlerClientCredentialsUsesClientSubjectGrants guards the user/client grant
// separation on the machine-to-machine path: a permission granted only for user-delegated
// access must not be mintable via client_credentials, while a client-subject grant must be.
func TestTokenHandlerClientCredentialsUsesClientSubjectGrants(t *testing.T) {
	gin.SetMode(gin.TestMode)

	const (
		baseURL     = "https://issuer.example.com"
		secret      = "test-secret"
		clientID    = "cc-client"
		clientPlain = "cc-secret-value"
		apiAudience = "https://api.orders.example.com"
	)

	db := testutils.NewDatabaseForTest(t)
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	hashed, err := bcrypt.GenerateFromPassword([]byte(clientPlain), bcrypt.DefaultCost)
	require.NoError(t, err)
	require.NoError(t, db.Create(&model.OidcClient{
		Base:     model.Base{ID: clientID},
		Name:     "Client Credentials Client",
		Secret:   string(hashed),
		IsPublic: false,
	}).Error)

	apiAccess := fakeAPIAccess{allowed: map[string]map[SubjectType][]string{
		apiAudience: {
			SubjectTypeUser:   {"read:orders"},
			SubjectTypeClient: {"write:orders"},
		},
	}}

	provider, err := newProvider(NewStore(db, apiAccess), nil, testTokenSigner{key: key}, Config{
		BaseURL:      baseURL,
		TokenBaseURL: baseURL,
		Secret:       []byte(secret),
	})
	require.NoError(t, err)
	handler := newTokenHandler(provider, newClaimsService(db, nil, baseURL, nil), apiAccess)

	requestToken := func(t *testing.T, scope string) map[string]any {
		t.Helper()
		form := url.Values{
			"grant_type": {"client_credentials"},
			"scope":      {scope},
			"resource":   {apiAudience},
		}
		req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/oidc/token", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.SetBasicAuth(clientID, clientPlain)

		rec := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(rec)
		c.Request = req
		handler.token(c)

		var body map[string]any
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		return body
	}

	// The client-subject grant is issued and audienced to the API
	body := requestToken(t, "openid write:orders")
	require.NotEmpty(t, body["access_token"], "client-granted scope must be issued, got error: %v (%v)", body["error"], body["error_description"])
	claims := decodeJWTPart(t, body["access_token"].(string), 1)
	require.Equal(t, []string{apiAudience}, jwtAudience(claims), "access token must be audience-bound to the API")
	require.Equal(t, []string{"write:orders"}, jwtScopes(claims), "identity scopes must be stripped from machine tokens")

	// The permission users may delegate is not available to the client itself
	body = requestToken(t, "read:orders")
	require.Empty(t, body["access_token"], "user-delegated permission must not be mintable machine-to-machine")
	require.Equal(t, "invalid_scope", body["error"])
}

func TestTokenHandlerClientCredentialsDefaultsResourceScopes(t *testing.T) {
	gin.SetMode(gin.TestMode)

	const (
		baseURL     = "https://issuer.example.com"
		secret      = "test-secret"
		clientID    = "cc-client"
		clientPlain = "cc-secret-value"
		apiAudience = "https://api.orders.example.com"
	)

	db := testutils.NewDatabaseForTest(t)
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	hashed, err := bcrypt.GenerateFromPassword([]byte(clientPlain), bcrypt.DefaultCost)
	require.NoError(t, err)
	require.NoError(t, db.Create(&model.OidcClient{
		Base:     model.Base{ID: clientID},
		Name:     "Client Credentials Client",
		Secret:   string(hashed),
		IsPublic: false,
	}).Error)

	apiAccess := fakeAPIAccess{allowed: map[string]map[SubjectType][]string{
		apiAudience: {
			SubjectTypeUser:   {"read:profile"},
			SubjectTypeClient: {"read:orders", "write:orders"},
		},
	}}

	provider, err := newProvider(NewStore(db, apiAccess), nil, testTokenSigner{key: key}, Config{
		BaseURL:      baseURL,
		TokenBaseURL: baseURL,
		Secret:       []byte(secret),
	})
	require.NoError(t, err)
	handler := newTokenHandler(provider, newClaimsService(db, nil, baseURL, nil), apiAccess)

	requestToken := func(t *testing.T, target string, form url.Values) map[string]any {
		t.Helper()
		req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, target, strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.SetBasicAuth(clientID, clientPlain)

		rec := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(rec)
		c.Request = req
		handler.token(c)

		var body map[string]any
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		require.NotEmpty(t, body["access_token"], "client credentials request must issue a token, got error: %v (%v)", body["error"], body["error_description"])
		return body
	}

	for _, tc := range []struct {
		name   string
		target string
		form   url.Values
	}{
		{
			name:   "resource parameter",
			target: "/api/oidc/token",
			form: url.Values{
				"grant_type": {"client_credentials"},
				"resource":   {apiAudience},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			body := requestToken(t, tc.target, tc.form)
			claims := decodeJWTPart(t, body["access_token"].(string), 1)
			require.Equal(t, []string{apiAudience}, jwtAudience(claims), "access token must be audience-bound to the API")
			require.Equal(t, []string{"read:orders", "write:orders"}, jwtScopes(claims), "all client-subject API scopes must be granted by default")
		})
	}
}

// jwtAudience normalizes the `aud` claim (string or []string) into a slice.
func jwtAudience(claims map[string]any) []string {
	switch aud := claims["aud"].(type) {
	case string:
		return []string{aud}
	case []any:
		out := make([]string, 0, len(aud))
		for _, a := range aud {
			if s, ok := a.(string); ok {
				out = append(out, s)
			}
		}
		return out
	default:
		return nil
	}
}

// jwtScopes extracts the scope claim from a decoded access token JWT as a sorted slice
// It reads the RFC 9068 `scp` list claim and falls back to the space-delimited `scope` string
func jwtScopes(claims map[string]any) []string {
	var out []string
	scp, ok := claims["scp"].([]any)
	if ok {
		for _, s := range scp {
			str, ok := s.(string)
			if ok {
				out = append(out, str)
			}
		}
	}

	if out == nil {
		scope, ok := claims["scope"].(string)
		if ok && scope != "" {
			out = strings.Fields(scope)
		}
	}

	sort.Strings(out)

	return out
}

// TestTokenHandlerRefreshGrantRevalidatesUser is the regression guard for the most
// security-sensitive part of the fosite migration: fosite's refresh-token grant replays
// the stored session without reloading the user, so the token handler must re-check the
// resource owner on every refresh. Without that re-check, a user who is disabled or
// removed from a group-restricted client after the initial login keeps minting fresh
// access/ID tokens (and rotating the refresh token) until the 30-day refresh token
// expires, defeating offboarding and incident response. This drives the real refresh grant
// end to end through the HTTP handler.
func TestTokenHandlerRefreshGrantRevalidatesUser(t *testing.T) {
	gin.SetMode(gin.TestMode)

	const (
		baseURL = "https://issuer.example.com"
		secret  = "test-secret"
	)

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	signer := testTokenSigner{key: key}

	// mintRefreshToken stores an active refresh-token session for the user/client pair and
	// returns the opaque token. It mirrors how the e2e test service seeds refresh tokens:
	// the HMAC signature is derived from the same global secret the provider uses, so the
	// real refresh grant resolves it.
	mintRefreshToken := func(t *testing.T, db *gorm.DB, clientID, userID string) string {
		t.Helper()
		globalSecret, err := DeriveGlobalSecret([]byte(secret))
		require.NoError(t, err)
		strategy := compose.NewOAuth2HMACStrategy(&fosite.Config{
			GlobalSecret:         globalSecret,
			RefreshTokenLifespan: 30 * 24 * time.Hour,
		})
		token, signature, err := strategy.GenerateRefreshToken(t.Context(), nil)
		require.NoError(t, err)

		now := time.Now().UTC()
		session := NewEmptySession()
		session.Subject = userID
		session.Claims = &fositejwt.IDTokenClaims{
			Subject:     userID,
			RequestedAt: now,
			AuthTime:    now,
			Extra:       map[string]any{},
		}
		session.SetExpiresAt(fosite.RefreshToken, now.Add(30*24*time.Hour))
		session.SetExpiresAt(fosite.AccessToken, now.Add(time.Hour))

		request := fosite.NewRequest()
		request.ID = "refresh-req-" + userID
		request.RequestedAt = now
		request.Client = Client{OidcClient: model.OidcClient{Base: model.Base{ID: clientID}, IsPublic: true}}
		request.RequestedScope = fosite.Arguments{"openid"}
		request.GrantedScope = fosite.Arguments{"openid"}
		request.RequestedAudience = fosite.Arguments{clientID}
		request.GrantedAudience = fosite.Arguments{clientID}
		request.Session = session

		require.NoError(t, NewStore(db, nil).CreateRefreshTokenSession(t.Context(), signature, "", request))
		return token
	}

	doRefresh := func(t *testing.T, db *gorm.DB, clientID, refreshToken string) map[string]any {
		t.Helper()
		provider, err := newProvider(NewStore(db, nil), nil, signer, Config{
			BaseURL:      baseURL,
			TokenBaseURL: baseURL,
			Secret:       []byte(secret),
		})
		require.NoError(t, err)
		handler := newTokenHandler(provider, newClaimsService(db, nil, baseURL, nil), nil)

		form := url.Values{
			"grant_type":    {"refresh_token"},
			"refresh_token": {refreshToken},
			"client_id":     {clientID},
		}
		req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/oidc/token", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		rec := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(rec)
		c.Request = req
		handler.token(c)

		var body map[string]any
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		return body
	}

	createClient := func(t *testing.T, db *gorm.DB, client model.OidcClient) {
		t.Helper()
		require.NoError(t, db.Create(&client).Error)
	}

	t.Run("enabled user receives rotated tokens", func(t *testing.T) {
		db := testutils.NewDatabaseForTest(t)
		const clientID, userID = "client-ok", "user-ok"
		createClient(t, db, model.OidcClient{Base: model.Base{ID: clientID}, Name: "Client", IsPublic: true})
		require.NoError(t, db.Create(&model.User{Base: model.Base{ID: userID}, Username: "tim"}).Error)

		token := mintRefreshToken(t, db, clientID, userID)
		body := doRefresh(t, db, clientID, token)

		require.NotEmpty(t, body["access_token"], "expected a new access token, got error: %v", body["error"])
		require.NotEmpty(t, body["refresh_token"])
		require.NotEqual(t, token, body["refresh_token"], "refresh token must be rotated")
	})

	t.Run("disabled user is rejected on refresh", func(t *testing.T) {
		db := testutils.NewDatabaseForTest(t)
		const clientID, userID = "client-disabled", "user-disabled"
		createClient(t, db, model.OidcClient{Base: model.Base{ID: clientID}, Name: "Client", IsPublic: true})
		require.NoError(t, db.Create(&model.User{Base: model.Base{ID: userID}, Username: "tim", Disabled: true}).Error)

		token := mintRefreshToken(t, db, clientID, userID)
		body := doRefresh(t, db, clientID, token)

		require.Empty(t, body["access_token"])
		require.Equal(t, "invalid_grant", body["error"])
	})

	t.Run("user removed from a group-restricted client is rejected on refresh", func(t *testing.T) {
		db := testutils.NewDatabaseForTest(t)
		const clientID, userID = "client-restricted", "user-outsider"
		group := model.UserGroup{Base: model.Base{ID: "allowed-group"}, Name: "allowed", FriendlyName: "Allowed"}
		require.NoError(t, db.Create(&group).Error)
		createClient(t, db, model.OidcClient{
			Base:              model.Base{ID: clientID},
			Name:              "Restricted",
			IsPublic:          true,
			IsGroupRestricted: true,
			AllowedUserGroups: []model.UserGroup{group},
		})
		// User is not a member of the allowed group.
		require.NoError(t, db.Create(&model.User{Base: model.Base{ID: userID}, Username: "outsider"}).Error)

		token := mintRefreshToken(t, db, clientID, userID)
		body := doRefresh(t, db, clientID, token)

		require.Empty(t, body["access_token"])
		require.Equal(t, "access_denied", body["error"])
	})
}

func TestTokenHandlerRefreshGrantPreservesAudienceAndScope(t *testing.T) {
	gin.SetMode(gin.TestMode)

	const (
		baseURL     = "https://issuer.example.com"
		secret      = "test-secret"
		apiResource = "https://api.orders.example.com"
	)

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	signer := testTokenSigner{key: key}

	// grantedAPI is a client that has been granted read:orders on the Orders API
	grantedAPI := userAccess(map[string][]string{apiResource: {"read:orders"}})
	// revokedAPI stands in for the same client after its API grant was removed: no APIs, no scopes
	revokedAPI := userAccess(map[string][]string{})

	// mintRefreshToken stores an active refresh-token session with the given granted scope and audience, standing in for a token issued by an earlier authorize, and returns the opaque token
	mintRefreshToken := func(t *testing.T, db *gorm.DB, clientID, userID string, grantedScope, grantedAudience fosite.Arguments) string {
		t.Helper()
		globalSecret, err := DeriveGlobalSecret([]byte(secret))
		require.NoError(t, err)
		strategy := compose.NewOAuth2HMACStrategy(&fosite.Config{
			GlobalSecret:         globalSecret,
			RefreshTokenLifespan: 30 * 24 * time.Hour,
		})
		token, signature, err := strategy.GenerateRefreshToken(t.Context(), nil)
		require.NoError(t, err)

		now := time.Now().UTC()
		session := NewEmptySession()
		session.Subject = userID
		session.Claims = &fositejwt.IDTokenClaims{
			Subject:     userID,
			RequestedAt: now,
			AuthTime:    now,
			Extra:       map[string]any{},
		}
		session.SetExpiresAt(fosite.RefreshToken, now.Add(30*24*time.Hour))
		session.SetExpiresAt(fosite.AccessToken, now.Add(time.Hour))

		request := fosite.NewRequest()
		request.ID = "refresh-req-" + userID
		request.RequestedAt = now
		request.Client = Client{OidcClient: model.OidcClient{Base: model.Base{ID: clientID}, IsPublic: true}}
		request.RequestedScope = grantedScope
		request.GrantedScope = grantedScope
		request.RequestedAudience = grantedAudience
		request.GrantedAudience = grantedAudience
		request.Session = session

		err = NewStore(db, nil).CreateRefreshTokenSession(t.Context(), signature, "", request)
		require.NoError(t, err)

		return token
	}

	// doRefresh runs the refresh grant through the real HTTP handler; apiAccess widens the client's
	// allowed scopes and audiences the same way the api module does in production, and extra merges
	// additional form parameters (such as a widened scope) over the base refresh request
	doRefresh := func(t *testing.T, db *gorm.DB, apiAccess APIAccessProvider, clientID, refreshToken string, extra url.Values) map[string]any {
		t.Helper()
		provider, err := newProvider(NewStore(db, apiAccess), nil, signer, Config{
			BaseURL:      baseURL,
			TokenBaseURL: baseURL,
			Secret:       []byte(secret),
		})
		require.NoError(t, err)
		handler := newTokenHandler(provider, newClaimsService(db, nil, baseURL, nil), nil)

		form := url.Values{
			"grant_type":    {"refresh_token"},
			"refresh_token": {refreshToken},
			"client_id":     {clientID},
		}
		maps.Copy(form, extra)
		req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/oidc/token", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		rec := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(rec)
		c.Request = req
		handler.token(c)

		var body map[string]any
		err = json.Unmarshal(rec.Body.Bytes(), &body)
		require.NoError(t, err)

		return body
	}

	seedUserAndClient := func(t *testing.T, db *gorm.DB, clientID, userID string) {
		t.Helper()
		rErr := db.Create(&model.OidcClient{Base: model.Base{ID: clientID}, Name: "Client", IsPublic: true}).Error
		require.NoError(t, rErr)
		rErr = db.Create(&model.User{Base: model.Base{ID: userID}, Username: "tim"}).Error
		require.NoError(t, rErr)
	}

	t.Run("refresh keeps the API audience and adds the issuer so the token still reaches userinfo", func(t *testing.T) {
		db := testutils.NewDatabaseForTest(t)
		const clientID, userID = "client-api", "user-api"
		seedUserAndClient(t, db, clientID, userID)

		token := mintRefreshToken(t, db, clientID, userID,
			fosite.Arguments{"openid", "read:orders"},
			fosite.Arguments{apiResource},
		)
		body := doRefresh(t, db, grantedAPI, clientID, token, nil)
		require.NotEmpty(t, body["access_token"], "expected a new access token, got error: %v", body["error"])
		// The grant still carries openid, so the identity side keeps issuing an ID token on refresh
		require.NotEmpty(t, body["id_token"])

		claims := decodeJWTPart(t, body["access_token"].(string), 1)
		// The refreshed access token stays bound to the original API audience and re-adds the issuer so it keeps working at userinfo, never widening to any other API
		require.ElementsMatch(t, []string{apiResource, baseURL}, jwtAudience(claims))
		// A token that requested openid alongside the API keeps the identity scope on the access token, matching what it was granted
		require.Equal(t, []string{"openid", "read:orders"}, jwtScopes(claims))
	})

	t.Run("refresh cannot upscope beyond the original grant via the scope parameter", func(t *testing.T) {
		db := testutils.NewDatabaseForTest(t)
		const clientID, userID = "client-upscope", "user-upscope"
		seedUserAndClient(t, db, clientID, userID)

		token := mintRefreshToken(t, db, clientID, userID,
			fosite.Arguments{"openid", "read:orders"},
			fosite.Arguments{apiResource},
		)
		// The refresh handler ignores the scope parameter and replays the stored grant, so asking for write:orders is a no-op rather than an escalation
		body := doRefresh(t, db, grantedAPI, clientID, token, url.Values{"scope": {"openid read:orders write:orders"}})
		require.NotEmpty(t, body["access_token"], "expected a new access token, got error: %v", body["error"])

		claims := decodeJWTPart(t, body["access_token"].(string), 1)
		require.ElementsMatch(t, []string{apiResource, baseURL}, jwtAudience(claims))
		// write:orders never appears on the refreshed token even though it was requested
		require.Equal(t, []string{"openid", "read:orders"}, jwtScopes(claims))
	})

	t.Run("revoking the client API grant makes the next refresh fail", func(t *testing.T) {
		db := testutils.NewDatabaseForTest(t)
		const clientID, userID = "client-revoked", "user-revoked"
		seedUserAndClient(t, db, clientID, userID)

		token := mintRefreshToken(t, db, clientID, userID,
			fosite.Arguments{"openid", "read:orders"},
			fosite.Arguments{apiResource},
		)
		// With the grant revoked the client no longer advertises read:orders, so the refresh handler rejects replaying that stored scope
		// Revocation is therefore enforced on the very next refresh rather than lingering until the refresh token expires
		body := doRefresh(t, db, revokedAPI, clientID, token, nil)
		require.Empty(t, body["access_token"])
		require.Equal(t, "invalid_scope", body["error"])
	})
}
