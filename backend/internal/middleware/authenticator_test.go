package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/italypaleale/francis/host/local"
	"github.com/pocket-id/fosite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/apikey"
	"github.com/pocket-id/pocket-id/backend/internal/appconfig"
	"github.com/pocket-id/pocket-id/backend/internal/auditlogs"
	"github.com/pocket-id/pocket-id/backend/internal/authz"
	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
	"github.com/pocket-id/pocket-id/backend/internal/oidc"
	"github.com/pocket-id/pocket-id/backend/internal/session"
	"github.com/pocket-id/pocket-id/backend/internal/users"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
	"github.com/pocket-id/pocket-id/backend/internal/utils/cookie"
	testutils "github.com/pocket-id/pocket-id/backend/internal/utils/testing"
)

func TestAuthorizationWithRealCredentials(t *testing.T) {
	gin.SetMode(gin.TestMode)

	originalEnvConfig := common.EnvConfig
	defer func() {
		common.EnvConfig = originalEnvConfig
	}()
	common.EnvConfig.AppURL = "https://test.example.com"
	common.EnvConfig.EncryptionKey = []byte("0123456789abcdef0123456789abcdef")

	db := testutils.NewDatabaseForTest(t)

	usersModule := users.New(users.Dependencies{DB: db})
	apiKeyModule, err := apikey.New(t.Context(), apikey.Dependencies{DB: db, CleanupDisabled: true})
	require.NoError(t, err)

	// Create one credential of each kind for a regular user, an admin and a disabled user
	// The regular users and the admin have a verified email, the others still have to verify theirs
	user := createUserForAuthorizationTest(t, db, "auth-user", false, false, true)
	admin := createUserForAuthorizationTest(t, db, "auth-admin", true, false, true)
	disabled := createUserForAuthorizationTest(t, db, "auth-disabled", true, true, true)
	unverified := createUserForAuthorizationTest(t, db, "auth-unverified", false, false, false)
	unverifiedAdmin := createUserForAuthorizationTest(t, db, "auth-unverified-admin", true, false, false)

	var sessions *session.Module
	testutils.NewActorHostForTest(t, func(t *testing.T, h *local.Host) {
		sessions, err = session.New(session.Dependencies{Actors: h, IPLocator: noIPLocation{}, AuditLog: nopAuditLogger{}})
		require.NoError(t, err)
	})
	userSession, err := sessions.Create(t.Context(), user.ID, "", "", "", time.Hour, true)
	require.NoError(t, err)
	unverifiedSession, err := sessions.Create(t.Context(), unverified.ID, "", "", "", time.Hour, true)
	require.NoError(t, err)
	revokedSession, err := sessions.Create(t.Context(), user.ID, "", "", "", time.Hour, true)
	require.NoError(t, err)
	_, revoked, err := sessions.Validate(t.Context(), revokedSession)
	require.NoError(t, err)
	require.NoError(t, sessions.Revoke(t.Context(), user.ID, revoked.ID, auditlogs.Origin{}))
	userAPIKey := createAPIKeyForAuthorizationTest(t, db, user, "user-raw-api-key", apikey.AllScopes())
	adminAPIKey := createAPIKeyForAuthorizationTest(t, db, admin, "admin-raw-api-key", apikey.AllScopes())
	adminAccountOnlyAPIKey := createAPIKeyForAuthorizationTest(t, db, admin, "admin-account-only-raw-api-key", datatype.StringList{string(authz.AccountRead)})
	disabledAPIKey := createAPIKeyForAuthorizationTest(t, db, disabled, "disabled-raw-api-key", apikey.AllScopes())
	unverifiedAPIKey := createAPIKeyForAuthorizationTest(t, db, unverified, "unverified-raw-api-key", apikey.AllScopes())
	unverifiedAdminAPIKey := createAPIKeyForAuthorizationTest(t, db, unverifiedAdmin, "unverified-admin-raw-api-key", apikey.AllScopes())

	// Mount one route per kind of requirement
	// Email verification is required, so unverified regular users only keep the scopes needed to verify their email
	cfg := appconfig.NewTestConfig(nil)
	cfg.EmailVerification = appconfig.EmailVerificationRequired
	router := gin.New()
	router.Use(NewErrorHandlerMiddleware().Add())
	resource := authz.ResourceURI(common.EnvConfig.AppURL)
	accessTokens := fakeAccessTokenVerifier{
		"admin-token":          oauthAccessRequest(admin.ID, fosite.Arguments{resource}, string(authz.UsersRead)),
		"admin-account-token":  oauthAccessRequest(admin.ID, fosite.Arguments{resource}, string(authz.AccountRead)),
		"user-admin-token":     oauthAccessRequest(user.ID, fosite.Arguments{resource}, string(authz.UsersRead)),
		"disabled-token":       oauthAccessRequest(disabled.ID, fosite.Arguments{resource}, string(authz.UsersRead)),
		"session-scope-token":  oauthAccessRequest(admin.ID, fosite.Arguments{resource}, string(authz.AccountSession)),
		"custom-api-token":     oauthAccessRequest(admin.ID, fosite.Arguments{"https://api.example.com", common.EnvConfig.AppURL}, "openid", string(authz.UsersRead)),
		"unverified-api-token": oauthAccessRequest(unverified.ID, fosite.Arguments{resource}, string(authz.AccountRead), string(authz.AccountApps)),
		"machine-token":        oauthAccessRequest(oidc.ClientSubject(oauthTestClientID), fosite.Arguments{resource}, string(authz.UsersRead), string(authz.AccountRead)),
	}
	apiRouter := NewAuthorization(apiKeyModule, accessTokens, usersModule, sessions, appconfig.StaticResolver{Config: cfg}).Router(router.Group("/api"))
	ok := func(c *gin.Context) {
		c.String(http.StatusOK, authz.PrincipalFrom(c).UserID)
	}
	apiRouter.GET("/session-only", authz.AccountSession, ok)
	apiRouter.GET("/account", authz.AccountRead, ok)
	apiRouter.GET("/apps", authz.AccountApps, ok)
	apiRouter.GET("/authorize", authz.AccountAuthorize, ok)
	apiRouter.GET("/admin", authz.UsersRead, ok)

	tests := []struct {
		name          string
		path          string
		authorization string
		sessionCookie string
		apiKey        string
		status        int
		userID        string
		errorCode     string
	}{
		{name: "session on session-only route", path: "/api/session-only", sessionCookie: userSession, status: http.StatusOK, userID: user.ID},
		{name: "API key on session-only route", path: "/api/session-only", apiKey: adminAPIKey, status: http.StatusForbidden, errorCode: "api_key_auth_not_allowed"},
		{name: "API key on account route", path: "/api/account", apiKey: userAPIKey, status: http.StatusOK, userID: user.ID},
		{name: "regular user session on admin route", path: "/api/admin", sessionCookie: userSession, status: http.StatusForbidden, errorCode: "forbidden"},
		{name: "regular user API key that stores admin scopes on admin route", path: "/api/admin", apiKey: userAPIKey, status: http.StatusForbidden, errorCode: "forbidden"},
		{name: "admin API key on admin route", path: "/api/admin", apiKey: adminAPIKey, status: http.StatusOK, userID: admin.ID},
		{name: "admin API key without the route's scope", path: "/api/admin", apiKey: adminAccountOnlyAPIKey, status: http.StatusForbidden, errorCode: "forbidden"},
		{name: "admin API key with the route's scope", path: "/api/account", apiKey: adminAccountOnlyAPIKey, status: http.StatusOK, userID: admin.ID},
		{name: "revoked session", path: "/api/account", sessionCookie: revokedSession, status: http.StatusUnauthorized, errorCode: "not_signed_in"},
		{name: "session token in the Authorization header", path: "/api/account", authorization: "Bearer " + userSession, status: http.StatusUnauthorized, errorCode: "not_signed_in"},
		{name: "invalid session cookie falls back to API key", path: "/api/admin", sessionCookie: "not-a-jwt", apiKey: adminAPIKey, status: http.StatusOK, userID: admin.ID},
		{name: "disabled user's API key", path: "/api/account", apiKey: disabledAPIKey, status: http.StatusForbidden, errorCode: "user_disabled"},
		{name: "verified user session on apps route", path: "/api/apps", sessionCookie: userSession, status: http.StatusOK, userID: user.ID},
		{name: "unverified user session on session-only route", path: "/api/session-only", sessionCookie: unverifiedSession, status: http.StatusOK, userID: unverified.ID},
		{name: "unverified user session on account route", path: "/api/account", sessionCookie: unverifiedSession, status: http.StatusOK, userID: unverified.ID},
		{name: "unverified user session on apps route", path: "/api/apps", sessionCookie: unverifiedSession, status: http.StatusForbidden, errorCode: "email_verification_required"},
		{name: "unverified user session on authorize route", path: "/api/authorize", sessionCookie: unverifiedSession, status: http.StatusForbidden, errorCode: "email_verification_required"},
		{name: "verified user session on authorize route", path: "/api/authorize", sessionCookie: userSession, status: http.StatusOK, userID: user.ID},
		{name: "unverified user API key on account route", path: "/api/account", apiKey: unverifiedAPIKey, status: http.StatusOK, userID: unverified.ID},
		{name: "unverified user API key on apps route", path: "/api/apps", apiKey: unverifiedAPIKey, status: http.StatusForbidden, errorCode: "email_verification_required"},
		{name: "unverified admin API key on admin route", path: "/api/admin", apiKey: unverifiedAdminAPIKey, status: http.StatusOK, userID: unverifiedAdmin.ID},
		{name: "unknown API key", path: "/api/account", apiKey: "unknown", status: http.StatusUnauthorized, errorCode: "not_signed_in"},
		{name: "admin OAuth token on admin route", path: "/api/admin", authorization: "Bearer admin-token", status: http.StatusOK, userID: admin.ID},
		{name: "admin OAuth token without the route's scope", path: "/api/admin", authorization: "Bearer admin-account-token", status: http.StatusForbidden, errorCode: "forbidden"},
		{name: "regular user OAuth token that carries an admin scope", path: "/api/admin", authorization: "Bearer user-admin-token", status: http.StatusForbidden, errorCode: "forbidden"},
		{name: "disabled user's OAuth token", path: "/api/admin", authorization: "Bearer disabled-token", status: http.StatusForbidden, errorCode: "user_disabled"},
		{name: "OAuth token on session-only route", path: "/api/session-only", authorization: "Bearer session-scope-token", status: http.StatusForbidden, errorCode: "oauth_token_auth_not_allowed"},
		{name: "custom API token with a colliding scope key", path: "/api/admin", authorization: "Bearer custom-api-token", status: http.StatusUnauthorized, errorCode: "not_signed_in"},
		{name: "unverified user OAuth token on apps route", path: "/api/apps", authorization: "Bearer unverified-api-token", status: http.StatusForbidden, errorCode: "email_verification_required"},
		{name: "unknown OAuth token", path: "/api/account", authorization: "Bearer unknown", status: http.StatusUnauthorized, errorCode: "not_signed_in"},
		{name: "client acting as itself on admin route", path: "/api/admin", authorization: "Bearer machine-token", status: http.StatusOK, userID: ""},
		{name: "client acting as itself on account route", path: "/api/account", authorization: "Bearer machine-token", status: http.StatusForbidden, errorCode: "client_token_auth_not_allowed"},
		{name: "client acting as itself on session-only route", path: "/api/session-only", authorization: "Bearer machine-token", status: http.StatusForbidden, errorCode: "oauth_token_auth_not_allowed"},
		{name: "no credentials", path: "/api/account", status: http.StatusUnauthorized, errorCode: "not_signed_in"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, test.path, nil)
			if test.authorization != "" {
				req.Header.Set("Authorization", test.authorization)
			}
			if test.sessionCookie != "" {
				req.AddCookie(&http.Cookie{Name: cookie.SessionTokenCookieName, Value: test.sessionCookie, Secure: true, HttpOnly: true, SameSite: http.SameSiteLaxMode})
			}
			if test.apiKey != "" {
				req.Header.Set("X-API-Key", test.apiKey)
			}
			recorder := httptest.NewRecorder()

			router.ServeHTTP(recorder, req)

			require.Equal(t, test.status, recorder.Code, recorder.Body.String())
			if test.status == http.StatusOK {
				require.Equal(t, test.userID, recorder.Body.String())
				return
			}

			var body map[string]any
			require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
			require.Equal(t, test.errorCode, body["code"])
		})
	}
}

type nopAuditLogger struct{}

func (nopAuditLogger) Record(context.Context, auditlogs.Event, auditlogs.Origin, string, auditlogs.Data, *gorm.DB) {
}

func (nopAuditLogger) KeepUsername(context.Context, string, string, *gorm.DB) error {
	return nil
}

func (nopAuditLogger) DeviceStringFromUserAgent(string) string {
	return ""
}

type noIPLocation struct{}

func (noIPLocation) GetLocationByIP(context.Context, string) (string, string, error) {
	return "", "", nil
}

// fakeAccessTokenVerifier resolves raw OAuth access tokens to the requests they were granted for
type fakeAccessTokenVerifier map[string]fosite.AccessRequester

func (f fakeAccessTokenVerifier) VerifyAccessToken(_ context.Context, token string) (fosite.AccessRequester, error) {
	accessRequest, ok := f[token]
	if !ok {
		return nil, fosite.ErrRequestUnauthorized
	}
	return accessRequest, nil
}

const oauthTestClientID = "machine-client"

func oauthAccessRequest(subject string, audience fosite.Arguments, scopes ...string) fosite.AccessRequester {
	accessRequest := fosite.NewAccessRequest(&fosite.DefaultSession{Subject: subject})
	accessRequest.Client = &fosite.DefaultClient{ID: oauthTestClientID}
	accessRequest.GrantedAudience = audience
	accessRequest.GrantedScope = scopes
	return accessRequest
}

func createUserForAuthorizationTest(t *testing.T, db *gorm.DB, username string, isAdmin, disabled, emailVerified bool) model.User {
	t.Helper()

	user := model.User{
		Username:      username,
		Email:         new(username + "@example.com"),
		EmailVerified: emailVerified,
		FirstName:     "Auth",
		LastName:      "User",
		DisplayName:   "Auth User",
		IsAdmin:       isAdmin,
		Disabled:      disabled,
	}

	err := db.Create(&user).Error
	require.NoError(t, err)

	return user
}

func createAPIKeyForAuthorizationTest(t *testing.T, db *gorm.DB, owner model.User, rawToken string, scopes datatype.StringList) string {
	t.Helper()

	err := db.Create(&apikey.ApiKey{
		Name:      rawToken,
		Key:       utils.CreateSha256Hash(rawToken),
		UserID:    owner.ID,
		ExpiresAt: datatype.DateTime(time.Now().Add(24 * time.Hour)),
		Scopes:    scopes,
	}).Error
	require.NoError(t, err)

	return rawToken
}
