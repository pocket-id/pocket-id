package backchannellogout

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"
	"time"

	"github.com/italypaleale/francis/actor"
	"github.com/italypaleale/francis/host/local"
	"github.com/pocket-id/fosite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/model"
	"github.com/pocket-id/pocket-id/backend/internal/oidc"
	"github.com/pocket-id/pocket-id/backend/internal/session"
	testutils "github.com/pocket-id/pocket-id/backend/internal/utils/testing"
)

// stubSigner returns a recognizable token so delivery tests can assert what was posted without a real key
// Logout token contents are covered by TestGenerateLogoutToken
type stubSigner struct{}

func (stubSigner) GenerateLogoutToken(userID string, clientID string, sessionID string) (string, error) {
	return "logout-token-" + userID + "-" + clientID + "-" + sessionID, nil
}

// createGrant stores a refresh token of a grant the user authorized the client in from the given Pocket ID session
func createGrant(t *testing.T, db *gorm.DB, requestID, userID, clientID, sessionID string, scopes ...string) {
	t.Helper()

	session := oidc.NewAuthenticatedSession(userID, "", time.Time{}, time.Time{})
	session.SessionID = sessionID
	request := &fosite.Request{
		ID:           requestID,
		RequestedAt:  time.Now().UTC(),
		Client:       oidc.Client{OidcClient: model.OidcClient{Base: model.Base{ID: clientID}}},
		GrantedScope: append(fosite.Arguments{"openid"}, scopes...),
		Form:         url.Values{},
		Session:      session,
	}
	require.NoError(t, oidc.NewStore(db, nil).CreateRefreshTokenSession(t.Context(), requestID+"-refresh", "", request))
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

func TestService_AuthorizationsForUsers(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	seedFixtures(t, db)
	s := &Service{db: db}

	targets, err := s.authorizationsForUsers(t.Context(), db, []string{"user-1"})
	require.NoError(t, err)

	// The client without a back-channel logout URL must not be returned
	require.Len(t, targets, 2)
	byClient := map[string]authorization{}
	for _, tgt := range targets {
		byClient[tgt.ClientID] = tgt
	}
	assert.Equal(t, "https://open.example.com/logout", byClient["client-open"].LogoutURL)
	assert.Equal(t, "https://restricted.example.com/logout", byClient["client-restricted"].LogoutURL)
	assert.Equal(t, "user-1", byClient["client-open"].UserID)
}

func TestService_AuthorizationsCarryGrantSessions(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	seedFixtures(t, db)
	s := &Service{db: db}
	createGrant(t, db, "open-1", "user-1", "client-open", "session-1")
	createGrant(t, db, "open-2", "user-1", "client-open", "session-2")
	createGrant(t, db, "open-other-user", "user-2", "client-open", "session-3")
	createGrant(t, db, "restricted-device", "user-1", "client-restricted", "")

	authorizations, err := s.authorizationsForUsers(t.Context(), db, []string{"user-1"})
	require.NoError(t, err)
	sessionIDsByClient := map[string][]string{}
	for _, a := range authorizations {
		sessionIDsByClient[a.ClientID] = a.GrantSessionIDs
	}
	assert.ElementsMatch(t, []string{"session-1", "session-2"}, sessionIDsByClient["client-open"])
	assert.Empty(t, sessionIDsByClient["client-restricted"])
}

func TestService_AuthorizationsForAuthorization(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	seedFixtures(t, db)
	s := &Service{db: db}

	t.Run("returns the authorized client", func(t *testing.T) {
		targets, err := s.authorizationsForAuthorization(t.Context(), db, "user-1", "client-open")
		require.NoError(t, err)
		require.Len(t, targets, 1)
		assert.Equal(t, "user-1", targets[0].UserID)
		assert.Equal(t, "https://open.example.com/logout", targets[0].LogoutURL)
	})

	t.Run("returns nothing for a client without a back-channel logout URL", func(t *testing.T) {
		targets, err := s.authorizationsForAuthorization(t.Context(), db, "user-1", "client-silent")
		require.NoError(t, err)
		assert.Empty(t, targets)
	})

	t.Run("returns nothing for a client the user has not authorized", func(t *testing.T) {
		targets, err := s.authorizationsForAuthorization(t.Context(), db, "user-2", "client-open")
		require.NoError(t, err)
		assert.Empty(t, targets)
	})
}

func TestService_AuthorizationsForClient(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	seedFixtures(t, db)
	s := &Service{db: db}

	t.Run("returns every user who authorized the client", func(t *testing.T) {
		targets, err := s.authorizationsForClient(t.Context(), db, "client-restricted")
		require.NoError(t, err)
		require.Len(t, targets, 2)
		userIDs := []string{targets[0].UserID, targets[1].UserID}
		assert.ElementsMatch(t, []string{"user-1", "user-2"}, userIDs)
		assert.Equal(t, "https://restricted.example.com/logout", targets[0].LogoutURL)
	})

	t.Run("returns nothing for a client without a back-channel logout URL", func(t *testing.T) {
		targets, err := s.authorizationsForClient(t.Context(), db, "client-silent")
		require.NoError(t, err)
		assert.Empty(t, targets)
	})
}

func TestService_AuthorizationsForLostGroupAccess(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	seedFixtures(t, db)
	s := &Service{db: db}

	t.Run("member of an allowed group is not returned", func(t *testing.T) {
		targets, err := s.authorizationsForLostGroupAccess(t.Context(), db, []string{"user-1"}, "")
		require.NoError(t, err)
		assert.Empty(t, targets)
	})

	t.Run("user outside every allowed group is returned for restricted clients only", func(t *testing.T) {
		targets, err := s.authorizationsForLostGroupAccess(t.Context(), db, []string{"user-2"}, "")
		require.NoError(t, err)
		require.Len(t, targets, 1)
		assert.Equal(t, "client-restricted", targets[0].ClientID)
		assert.Equal(t, "user-2", targets[0].UserID)
		assert.Equal(t, "https://restricted.example.com/logout", targets[0].LogoutURL)
	})

	t.Run("client filter matches all users that lost access", func(t *testing.T) {
		targets, err := s.authorizationsForLostGroupAccess(t.Context(), db, nil, "client-restricted")
		require.NoError(t, err)
		require.Len(t, targets, 1)
		assert.Equal(t, "user-2", targets[0].UserID)
	})

	t.Run("no filters matches nothing instead of every user", func(t *testing.T) {
		targets, err := s.authorizationsForLostGroupAccess(t.Context(), db, nil, "")
		require.NoError(t, err)
		assert.Empty(t, targets)
	})

	t.Run("removing the user from the group makes their restricted client a target", func(t *testing.T) {
		require.NoError(t, db.Model(&model.UserGroup{Base: model.Base{ID: "group-1"}}).Association("Users").Clear())
		targets, err := s.authorizationsForLostGroupAccess(t.Context(), db, []string{"user-1"}, "")
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

type noIPLocation struct{}

func (noIPLocation) GetLocationByIP(context.Context, string) (string, string, error) {
	return "", "", nil
}

// newServiceWithReceiver points the back-channel logout URLs of the seeded clients at a test server and returns the logout tokens it receives
func newServiceWithReceiver(t *testing.T, db *gorm.DB) (*Service, *session.Module, <-chan string) {
	t.Helper()

	received := make(chan string, 16)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.NoError(t, r.ParseForm())
		received <- r.PostForm.Get("logout_token")
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)
	for _, clientID := range []string{"client-open", "client-restricted"} {
		require.NoError(t, db.Model(&model.OidcClient{}).Where("id = ?", clientID).Update("backchannel_logout_url", server.URL).Error)
	}

	var s *Service
	var sessions *session.Module
	testutils.NewActorHostForTest(t, func(t *testing.T, h *local.Host) {
		var err error
		s, err = NewService(db, stubSigner{}, server.Client(), h)
		require.NoError(t, err)
		sessions, err = session.New(session.Dependencies{Actors: h, IPLocator: noIPLocation{}, EndNotifier: s})
		require.NoError(t, err)
	})
	return s, sessions, received
}

func receiveLogoutTokens(t *testing.T, received <-chan string, count int) []string {
	t.Helper()

	tokens := make([]string, 0, count)
	for range count {
		select {
		case logoutToken := <-received:
			tokens = append(tokens, logoutToken)
		case <-time.After(10 * time.Second):
			t.Fatalf("received %d of %d logout tokens", len(tokens), count)
		}
	}
	return tokens
}

func TestService_SessionsEnded(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	seedFixtures(t, db)
	s, _, received := newServiceWithReceiver(t, db)

	createGrant(t, db, "open", "user-1", "client-open", "ended-session")
	createGrant(t, db, "restricted", "user-1", "client-restricted", "ended-session", "offline_access")
	createGrant(t, db, "silent", "user-1", "client-silent", "ended-session")
	createGrant(t, db, "other-session", "user-1", "client-open", "other-session")

	s.SessionsEnded(t.Context(), "user-1", []string{"ended-session"})

	assert.ElementsMatch(t, []string{
		"logout-token-user-1-client-open-ended-session",
		"logout-token-user-1-client-restricted-ended-session",
	}, receiveLogoutTokens(t, received, 2))

	var refreshTokens []oidc.OAuth2Session
	require.NoError(t, db.Where("kind = ?", "refresh_token").Find(&refreshTokens).Error)
	activeByRequest := map[string]bool{}
	for _, refreshToken := range refreshTokens {
		activeByRequest[refreshToken.RequestID] = refreshToken.Active
	}
	assert.Equal(t, map[string]bool{"open": false, "restricted": true, "silent": false, "other-session": true}, activeByRequest)
}

func TestService_NotifyUser(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	seedFixtures(t, db)
	s, sessions, received := newServiceWithReceiver(t, db)

	token, err := sessions.Create(t.Context(), "user-1", "", "", "", time.Hour, true)
	require.NoError(t, err)
	_, live, err := sessions.Validate(t.Context(), token)
	require.NoError(t, err)
	createGrant(t, db, "open-live", "user-1", "client-open", live.ID)
	createGrant(t, db, "open-ended", "user-1", "client-open", "ended-session")
	createGrant(t, db, "restricted-device", "user-1", "client-restricted", "")

	s.NotifyUser(t.Context(), "user-1")

	assert.ElementsMatch(t, []string{
		"logout-token-user-1-client-open-",
		"logout-token-user-1-client-open-" + live.ID,
		"logout-token-user-1-client-restricted-",
	}, receiveLogoutTokens(t, received, 3), "every client gets a token for all sessions, plus one per live session for clients that require a sid")
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

	const clientCount = 8
	received := make(chan string, clientCount)
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

	// Exceed the capacity group so idle actors cannot prevent later clients from receiving logout
	targets := []target{{UserID: "user-1", ClientID: "client-fail", LogoutURL: failingServer.URL}}
	expected := make([]string, 0, clientCount)
	for i := range clientCount {
		clientID := "api://example/client-" + strconv.Itoa(i)
		targets = append(targets, target{UserID: "user-1", ClientID: clientID, LogoutURL: server.URL})
		expected = append(expected, "logout-token-user-1-"+clientID+"-")
	}
	s.notifyClients(t.Context(), targets)

	// All recipients must receive their own token before the actors' idle timeout can release capacity
	timeout := time.NewTimer(10 * time.Second)
	defer timeout.Stop()
	actual := make([]string, 0, clientCount)
	for range clientCount {
		select {
		case logoutToken := <-received:
			actual = append(actual, logoutToken)
		case <-timeout.C:
			t.Fatalf("received %d of %d logout tokens", len(actual), clientCount)
		}
	}
	assert.ElementsMatch(t, expected, actual)
}

func TestService_sendLogoutToken_responseClassification(t *testing.T) {
	for _, test := range []struct {
		status    int
		permanent bool
	}{
		{status: http.StatusOK},
		{status: http.StatusNoContent},
		{status: http.StatusFound, permanent: true},
		{status: http.StatusBadRequest, permanent: true},
		{status: http.StatusUnauthorized, permanent: true},
		{status: http.StatusForbidden, permanent: true},
		{status: http.StatusNotFound, permanent: true},
		{status: http.StatusRequestTimeout},
		{status: http.StatusTooManyRequests},
		{status: http.StatusInternalServerError},
		{status: http.StatusBadGateway},
		{status: http.StatusServiceUnavailable},
		{status: http.StatusGatewayTimeout},
	} {
		t.Run(strconv.Itoa(test.status), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(test.status)
			}))
			t.Cleanup(server.Close)
			s := &Service{tokenSigner: stubSigner{}, httpClient: newHTTPClient(server.Client())}

			err := s.sendLogoutToken(t.Context(), target{UserID: "user-1", ClientID: "client-1", LogoutURL: server.URL})
			if test.status >= 200 && test.status < 300 {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			assert.Equal(t, test.permanent, errors.Is(err, actor.ErrJobPermanentFailure))
		})
	}
}

func TestService_sendLogoutToken_networkFailureIsRetryable(t *testing.T) {
	transportErr := errors.New("connection reset")
	s := &Service{
		tokenSigner: stubSigner{},
		httpClient: newHTTPClient(&http.Client{
			Transport: &testutils.MockRoundTripper{Err: transportErr},
		}),
	}

	err := s.sendLogoutToken(t.Context(), target{UserID: "user-1", ClientID: "client-1", LogoutURL: "https://rp.example/logout"})
	require.ErrorIs(t, err, transportErr)
	assert.NotErrorIs(t, err, actor.ErrJobPermanentFailure)
}
