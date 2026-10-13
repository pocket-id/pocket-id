package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/apikey"
	"github.com/pocket-id/pocket-id/backend/internal/appconfig"
	"github.com/pocket-id/pocket-id/backend/internal/authz"
	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/instanceid"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
	"github.com/pocket-id/pocket-id/backend/internal/service"
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

	instanceID, err := instanceid.Load(t.Context(), db)
	require.NoError(t, err)

	jwtService, err := service.NewJwtService(t.Context(), db, instanceID)
	require.NoError(t, err)

	userService := service.NewUserService(db, jwtService, nil, nil, nil, nil, nil, nil)
	apiKeyModule, err := apikey.New(t.Context(), apikey.Dependencies{DB: db, CleanupDisabled: true})
	require.NoError(t, err)

	// Create one credential of each kind for a regular user, an admin and a disabled user
	// The regular users and the admin have a verified email, the others still have to verify theirs
	user := createUserForAuthorizationTest(t, db, "auth-user", false, false, true)
	admin := createUserForAuthorizationTest(t, db, "auth-admin", true, false, true)
	disabled := createUserForAuthorizationTest(t, db, "auth-disabled", true, true, true)
	unverified := createUserForAuthorizationTest(t, db, "auth-unverified", false, false, false)
	unverifiedAdmin := createUserForAuthorizationTest(t, db, "auth-unverified-admin", true, false, false)

	userSession, err := jwtService.GenerateAccessToken(user, "", time.Hour)
	require.NoError(t, err)
	unverifiedSession, err := jwtService.GenerateAccessToken(unverified, "", time.Hour)
	require.NoError(t, err)
	userAPIKey := createAPIKeyForAuthorizationTest(t, db, user, "user-raw-api-key")
	adminAPIKey := createAPIKeyForAuthorizationTest(t, db, admin, "admin-raw-api-key")
	disabledAPIKey := createAPIKeyForAuthorizationTest(t, db, disabled, "disabled-raw-api-key")
	unverifiedAPIKey := createAPIKeyForAuthorizationTest(t, db, unverified, "unverified-raw-api-key")
	unverifiedAdminAPIKey := createAPIKeyForAuthorizationTest(t, db, unverifiedAdmin, "unverified-admin-raw-api-key")

	// Mount one route per kind of requirement
	// Email verification is required, so unverified regular users only keep the scopes needed to verify their email
	cfg := appconfig.NewTestConfig(nil)
	cfg.EmailVerification = appconfig.EmailVerificationRequired
	router := gin.New()
	router.Use(NewErrorHandlerMiddleware().Add())
	apiRouter := NewAuthorization(apiKeyModule, userService, jwtService, appconfig.StaticResolver{Config: cfg}).Router(router.Group("/api"))
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
		{name: "session on session-only route", path: "/api/session-only", authorization: "Bearer " + userSession, status: http.StatusOK, userID: user.ID},
		{name: "API key on session-only route", path: "/api/session-only", apiKey: adminAPIKey, status: http.StatusForbidden, errorCode: "api_key_auth_not_allowed"},
		{name: "API key on account route", path: "/api/account", apiKey: userAPIKey, status: http.StatusOK, userID: user.ID},
		{name: "regular user session on admin route", path: "/api/admin", authorization: "Bearer " + userSession, status: http.StatusForbidden, errorCode: "forbidden"},
		{name: "regular user API key on admin route", path: "/api/admin", apiKey: userAPIKey, status: http.StatusForbidden, errorCode: "forbidden"},
		{name: "admin API key on admin route", path: "/api/admin", apiKey: adminAPIKey, status: http.StatusOK, userID: admin.ID},
		{name: "invalid session cookie falls back to API key", path: "/api/admin", sessionCookie: "not-a-jwt", apiKey: adminAPIKey, status: http.StatusOK, userID: admin.ID},
		{name: "disabled user's API key", path: "/api/account", apiKey: disabledAPIKey, status: http.StatusForbidden, errorCode: "user_disabled"},
		{name: "verified user session on apps route", path: "/api/apps", authorization: "Bearer " + userSession, status: http.StatusOK, userID: user.ID},
		{name: "unverified user session on session-only route", path: "/api/session-only", authorization: "Bearer " + unverifiedSession, status: http.StatusOK, userID: unverified.ID},
		{name: "unverified user session on account route", path: "/api/account", authorization: "Bearer " + unverifiedSession, status: http.StatusOK, userID: unverified.ID},
		{name: "unverified user session on apps route", path: "/api/apps", authorization: "Bearer " + unverifiedSession, status: http.StatusForbidden, errorCode: "email_verification_required"},
		{name: "unverified user session on authorize route", path: "/api/authorize", authorization: "Bearer " + unverifiedSession, status: http.StatusForbidden, errorCode: "email_verification_required"},
		{name: "verified user session on authorize route", path: "/api/authorize", authorization: "Bearer " + userSession, status: http.StatusOK, userID: user.ID},
		{name: "unverified user API key on account route", path: "/api/account", apiKey: unverifiedAPIKey, status: http.StatusOK, userID: unverified.ID},
		{name: "unverified user API key on apps route", path: "/api/apps", apiKey: unverifiedAPIKey, status: http.StatusForbidden, errorCode: "email_verification_required"},
		{name: "unverified admin API key on admin route", path: "/api/admin", apiKey: unverifiedAdminAPIKey, status: http.StatusOK, userID: unverifiedAdmin.ID},
		{name: "unknown API key", path: "/api/account", apiKey: "unknown", status: http.StatusUnauthorized, errorCode: "not_signed_in"},
		{name: "no credentials", path: "/api/account", status: http.StatusUnauthorized, errorCode: "not_signed_in"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, test.path, nil)
			if test.authorization != "" {
				req.Header.Set("Authorization", test.authorization)
			}
			if test.sessionCookie != "" {
				req.AddCookie(&http.Cookie{Name: cookie.AccessTokenCookieName, Value: test.sessionCookie, Secure: true, HttpOnly: true, SameSite: http.SameSiteLaxMode})
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

func createAPIKeyForAuthorizationTest(t *testing.T, db *gorm.DB, owner model.User, rawToken string) string {
	t.Helper()

	err := db.Create(&apikey.ApiKey{
		Name:      owner.Username + " key",
		Key:       utils.CreateSha256Hash(rawToken),
		UserID:    owner.ID,
		ExpiresAt: datatype.DateTime(time.Now().Add(24 * time.Hour)),
	}).Error
	require.NoError(t, err)

	return rawToken
}
