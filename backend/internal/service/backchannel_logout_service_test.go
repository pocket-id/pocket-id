package service

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lestrrat-go/jwx/v4/jws"
	"github.com/lestrrat-go/jwx/v4/jwt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/model"
	testutils "github.com/pocket-id/pocket-id/backend/internal/utils/testing"
)

func TestGenerateLogoutToken(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	envConfig := newTestEnvConfig()
	jwtService := initJwtService(t, db, newInstanceID(t, db), nil, envConfig)

	signed, err := jwtService.GenerateLogoutToken("user-id", "client-id")
	require.NoError(t, err)

	// The token must carry the "logout+jwt" typ header required by the spec
	message, err := jws.Parse([]byte(signed))
	require.NoError(t, err)
	require.Len(t, message.Signatures(), 1)
	typ, ok := message.Signatures()[0].ProtectedHeaders().Type()
	require.True(t, ok)
	assert.Equal(t, LogoutTokenJWTTyp, typ)

	alg, err := jwtService.GetKeyAlg()
	require.NoError(t, err)
	publicKey, err := jwtService.GetPublicJWK()
	require.NoError(t, err)
	token, err := jwt.ParseString(signed, jwt.WithValidate(true), jwt.WithKey(alg, publicKey))
	require.NoError(t, err)

	subject, _ := token.Subject()
	assert.Equal(t, "user-id", subject)
	audience, _ := token.Audience()
	assert.Equal(t, []string{"client-id"}, audience)
	issuer, _ := token.Issuer()
	assert.Equal(t, envConfig.AppURL, issuer)
	jti, _ := token.JwtID()
	assert.Regexp(t, uuidRegexPattern, jti)

	events, err := jwt.Get[map[string]any](token, "events")
	require.NoError(t, err)
	assert.Contains(t, events, BackchannelLogoutEvent)

	// The spec forbids a nonce claim in logout tokens
	assert.False(t, token.Has("nonce"))
}

func seedBackchannelLogoutFixtures(t *testing.T, db *gorm.DB) {
	t.Helper()

	require.NoError(t, db.Create(&model.User{Base: model.Base{ID: "user-1"}, Username: "user1"}).Error)
	require.NoError(t, db.Create(&model.User{Base: model.Base{ID: "user-2"}, Username: "user2"}).Error)

	group := model.UserGroup{Base: model.Base{ID: "group-1"}, Name: "group1", FriendlyName: "Group 1"}
	require.NoError(t, db.Create(&group).Error)
	require.NoError(t, db.Model(&group).Association("Users").Append(&model.User{Base: model.Base{ID: "user-1"}}))

	// An unrestricted client with a back-channel logout URL
	require.NoError(t, db.Create(&model.OidcClient{
		Base:                 model.Base{ID: "client-open"},
		Name:                 "Open Client",
		BackchannelLogoutURL: new("https://open.example.com/logout"),
	}).Error)

	// An unrestricted client without a back-channel logout URL
	require.NoError(t, db.Create(&model.OidcClient{
		Base: model.Base{ID: "client-silent"},
		Name: "Silent Client",
	}).Error)

	// A group-restricted client that allows group-1
	restricted := model.OidcClient{
		Base:                 model.Base{ID: "client-restricted"},
		Name:                 "Restricted Client",
		IsGroupRestricted:    true,
		BackchannelLogoutURL: new("https://restricted.example.com/logout"),
	}
	require.NoError(t, db.Create(&restricted).Error)
	require.NoError(t, db.Model(&restricted).Association("AllowedUserGroups").Append(&model.UserGroup{Base: model.Base{ID: "group-1"}}))

	for _, clientID := range []string{"client-open", "client-silent", "client-restricted"} {
		require.NoError(t, db.Create(&model.UserAuthorizedOidcClient{UserID: "user-1", ClientID: clientID}).Error)
	}
	require.NoError(t, db.Create(&model.UserAuthorizedOidcClient{UserID: "user-2", ClientID: "client-restricted"}).Error)
}

func TestBackchannelLogoutService_TargetsForUsers(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	seedBackchannelLogoutFixtures(t, db)
	s := NewBackchannelLogoutService(db, nil, nil)

	targets, err := s.targetsForUsers(t.Context(), db, []string{"user-1"})
	require.NoError(t, err)

	// The client without a back-channel logout URL must not be returned
	require.Len(t, targets, 2)
	byClient := map[string]backchannelLogoutTarget{}
	for _, target := range targets {
		byClient[target.ClientID] = target
	}
	assert.Equal(t, "https://open.example.com/logout", byClient["client-open"].LogoutURL)
	assert.Equal(t, "https://restricted.example.com/logout", byClient["client-restricted"].LogoutURL)
	assert.Equal(t, "user-1", byClient["client-open"].UserID)
}

func TestBackchannelLogoutService_TargetsForAuthorization(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	seedBackchannelLogoutFixtures(t, db)
	s := NewBackchannelLogoutService(db, nil, nil)

	t.Run("returns the authorized client", func(t *testing.T) {
		targets, err := s.targetsForAuthorization(t.Context(), db, "user-1", "client-open")
		require.NoError(t, err)
		require.Len(t, targets, 1)
		assert.Equal(t, "user-1", targets[0].UserID)
		assert.Equal(t, "https://open.example.com/logout", targets[0].LogoutURL)
	})

	t.Run("returns nothing for a client without a back-channel logout URL", func(t *testing.T) {
		targets, err := s.targetsForAuthorization(t.Context(), db, "user-1", "client-silent")
		require.NoError(t, err)
		assert.Empty(t, targets)
	})

	t.Run("returns nothing for a client the user has not authorized", func(t *testing.T) {
		targets, err := s.targetsForAuthorization(t.Context(), db, "user-2", "client-open")
		require.NoError(t, err)
		assert.Empty(t, targets)
	})
}

func TestBackchannelLogoutService_TargetsForClient(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	seedBackchannelLogoutFixtures(t, db)
	s := NewBackchannelLogoutService(db, nil, nil)

	t.Run("returns every user who authorized the client", func(t *testing.T) {
		targets, err := s.targetsForClient(t.Context(), db, "client-restricted")
		require.NoError(t, err)
		require.Len(t, targets, 2)
		userIDs := []string{targets[0].UserID, targets[1].UserID}
		assert.ElementsMatch(t, []string{"user-1", "user-2"}, userIDs)
		assert.Equal(t, "https://restricted.example.com/logout", targets[0].LogoutURL)
	})

	t.Run("returns nothing for a client without a back-channel logout URL", func(t *testing.T) {
		targets, err := s.targetsForClient(t.Context(), db, "client-silent")
		require.NoError(t, err)
		assert.Empty(t, targets)
	})
}

func TestBackchannelLogoutService_TargetsForLostGroupAccess(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	seedBackchannelLogoutFixtures(t, db)
	s := NewBackchannelLogoutService(db, nil, nil)

	t.Run("member of an allowed group is not returned", func(t *testing.T) {
		targets, err := s.targetsForLostGroupAccess(t.Context(), db, []string{"user-1"}, "")
		require.NoError(t, err)
		assert.Empty(t, targets)
	})

	t.Run("user outside every allowed group is returned for restricted clients only", func(t *testing.T) {
		targets, err := s.targetsForLostGroupAccess(t.Context(), db, []string{"user-2"}, "")
		require.NoError(t, err)
		require.Len(t, targets, 1)
		assert.Equal(t, "client-restricted", targets[0].ClientID)
		assert.Equal(t, "user-2", targets[0].UserID)
		assert.Equal(t, "https://restricted.example.com/logout", targets[0].LogoutURL)
	})

	t.Run("client filter matches all users that lost access", func(t *testing.T) {
		targets, err := s.targetsForLostGroupAccess(t.Context(), db, nil, "client-restricted")
		require.NoError(t, err)
		require.Len(t, targets, 1)
		assert.Equal(t, "user-2", targets[0].UserID)
	})

	t.Run("no filters matches nothing instead of every user", func(t *testing.T) {
		targets, err := s.targetsForLostGroupAccess(t.Context(), db, nil, "")
		require.NoError(t, err)
		assert.Empty(t, targets)
	})

	t.Run("removing the user from the group makes their restricted client a target", func(t *testing.T) {
		require.NoError(t, db.Model(&model.UserGroup{Base: model.Base{ID: "group-1"}}).Association("Users").Clear())
		targets, err := s.targetsForLostGroupAccess(t.Context(), db, []string{"user-1"}, "")
		require.NoError(t, err)
		require.Len(t, targets, 1)
		assert.Equal(t, "client-restricted", targets[0].ClientID)
	})
}

func TestBackchannelLogoutService_PrepareUserNotifications(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	seedBackchannelLogoutFixtures(t, db)
	s := NewBackchannelLogoutService(db, nil, nil)

	t.Run("returns a notify function for a user with targets", func(t *testing.T) {
		notify, err := s.PrepareUserNotifications(t.Context(), db, []string{"user-1"})
		require.NoError(t, err)
		require.NotNil(t, notify)
	})

	t.Run("returns a notify function that delivers nothing for a user without targets", func(t *testing.T) {
		require.NoError(t, db.Create(&model.User{Base: model.Base{ID: "user-3"}, Username: "user3"}).Error)
		notify, err := s.PrepareUserNotifications(t.Context(), db, []string{"user-3"})
		require.NoError(t, err)
		require.NotNil(t, notify)

		// There is nothing to send, so the delivery must not reach the nil JWT service the test constructed the service with
		assert.NotPanics(t, notify)
	})
}

func TestBackchannelLogoutService_sendLogoutToken_refusesRedirects(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	envConfig := newTestEnvConfig()
	jwtService := initJwtService(t, db, newInstanceID(t, db), nil, envConfig)

	// Following the redirect would turn the POST into a GET without the logout token, so it must be reported as a failure
	var redirectTargetHit bool
	redirectTarget := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		redirectTargetHit = true
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(redirectTarget.Close)
	redirecting := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, redirectTarget.URL, http.StatusFound)
	}))
	t.Cleanup(redirecting.Close)

	s := NewBackchannelLogoutService(db, jwtService, redirecting.Client())
	err := s.sendLogoutToken(t.Context(), backchannelLogoutTarget{UserID: "user-1", ClientID: "client-1", LogoutURL: redirecting.URL})
	require.ErrorContains(t, err, "302")
	assert.False(t, redirectTargetHit)
}

func TestBackchannelLogoutService_notifyClientsSync(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	envConfig := newTestEnvConfig()
	jwtService := initJwtService(t, db, newInstanceID(t, db), nil, envConfig)

	received := make(chan string, 2)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "application/x-www-form-urlencoded", r.Header.Get("Content-Type"))
		assert.NoError(t, r.ParseForm())
		received <- r.PostForm.Get("logout_token")
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)

	s := NewBackchannelLogoutService(db, jwtService, server.Client())

	// A failing target must not prevent delivery to the remaining targets
	failingServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	t.Cleanup(failingServer.Close)

	s.notifyClientsSync(t.Context(), []backchannelLogoutTarget{
		{UserID: "user-1", ClientID: "client-fail", LogoutURL: failingServer.URL},
		{UserID: "user-1", ClientID: "client-ok", LogoutURL: server.URL},
	})

	require.Len(t, received, 1)
	logoutToken := <-received
	require.NotEmpty(t, logoutToken)

	alg, err := jwtService.GetKeyAlg()
	require.NoError(t, err)
	publicKey, err := jwtService.GetPublicJWK()
	require.NoError(t, err)
	token, err := jwt.ParseString(logoutToken, jwt.WithValidate(true), jwt.WithKey(alg, publicKey))
	require.NoError(t, err)

	subject, _ := token.Subject()
	assert.Equal(t, "user-1", subject)
	audience, _ := token.Audience()
	assert.Equal(t, []string{"client-ok"}, audience)
}
