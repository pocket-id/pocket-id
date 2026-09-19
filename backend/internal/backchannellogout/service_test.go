package backchannellogout

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/italypaleale/francis/host/local"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/model"
	testutils "github.com/pocket-id/pocket-id/backend/internal/utils/testing"
)

// stubSigner returns a recognizable token so delivery tests can assert what was posted without a real key
// Logout token contents are covered by the JWT service's own tests
type stubSigner struct{}

func (stubSigner) GenerateLogoutToken(userID string, clientID string) (string, error) {
	return "logout-token-" + userID + "-" + clientID, nil
}

func seedFixtures(t *testing.T, db *gorm.DB) {
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
		BackchannelLogoutURL: "https://open.example.com/logout",
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
		BackchannelLogoutURL: "https://restricted.example.com/logout",
	}
	require.NoError(t, db.Create(&restricted).Error)
	require.NoError(t, db.Model(&restricted).Association("AllowedUserGroups").Append(&model.UserGroup{Base: model.Base{ID: "group-1"}}))

	for _, clientID := range []string{"client-open", "client-silent", "client-restricted"} {
		require.NoError(t, db.Create(&model.UserAuthorizedOidcClient{UserID: "user-1", ClientID: clientID}).Error)
	}
	require.NoError(t, db.Create(&model.UserAuthorizedOidcClient{UserID: "user-2", ClientID: "client-restricted"}).Error)
}

func TestService_TargetsForUsers(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	seedFixtures(t, db)
	s := &Service{db: db}

	targets, err := s.targetsForUsers(t.Context(), db, []string{"user-1"})
	require.NoError(t, err)

	// The client without a back-channel logout URL must not be returned
	require.Len(t, targets, 2)
	byClient := map[string]target{}
	for _, tgt := range targets {
		byClient[tgt.ClientID] = tgt
	}
	assert.Equal(t, "https://open.example.com/logout", byClient["client-open"].LogoutURL)
	assert.Equal(t, "https://restricted.example.com/logout", byClient["client-restricted"].LogoutURL)
	assert.Equal(t, "user-1", byClient["client-open"].UserID)
}

func TestService_TargetsForAuthorization(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	seedFixtures(t, db)
	s := &Service{db: db}

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

func TestService_TargetsForClient(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	seedFixtures(t, db)
	s := &Service{db: db}

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

func TestService_TargetsForLostGroupAccess(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	seedFixtures(t, db)
	s := &Service{db: db}

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

func TestService_PrepareUserNotifications(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	seedFixtures(t, db)
	s := &Service{db: db}

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

		// There is nothing to send, so calling the function must be a no-op
		assert.NotPanics(t, notify)
	})
}

func TestService_sendLogoutToken_refusesRedirects(t *testing.T) {
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

	s := &Service{tokenSigner: stubSigner{}, httpClient: newHTTPClient(redirecting.Client())}
	err := s.sendLogoutToken(t.Context(), target{UserID: "user-1", ClientID: "client-1", LogoutURL: redirecting.URL})
	require.ErrorContains(t, err, "302")
	assert.False(t, redirectTargetHit)
}

func TestService_notifyClients(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)

	received := make(chan string, 2)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "application/x-www-form-urlencoded", r.Header.Get("Content-Type"))
		assert.NoError(t, r.ParseForm())
		received <- r.PostForm.Get("logout_token")
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)

	// A failing target must not prevent delivery to the remaining targets
	failingServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	t.Cleanup(failingServer.Close)

	var s *Service
	testutils.NewActorHostForTest(t, func(t *testing.T, h *local.Host) {
		var err error
		s, err = NewService(db, stubSigner{}, server.Client(), h)
		require.NoError(t, err)
	})

	s.notifyClients(t.Context(), []target{
		{UserID: "user-1", ClientID: "client-fail", LogoutURL: failingServer.URL},
		{UserID: "user-1", ClientID: "client-ok", LogoutURL: server.URL},
	})

	// The jobs are executed asynchronously by the actor runtime
	select {
	case logoutToken := <-received:
		assert.Equal(t, "logout-token-user-1-client-ok", logoutToken)
	case <-time.After(10 * time.Second):
		t.Fatal("timed out waiting for the logout token to be delivered")
	}
}
