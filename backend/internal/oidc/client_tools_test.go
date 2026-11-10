package oidc

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/italypaleale/francis/host/local"
	"github.com/stretchr/testify/require"

	"github.com/pocket-id/pocket-id/backend/internal/backchannellogout"
	"github.com/pocket-id/pocket-id/backend/internal/dto"
	"github.com/pocket-id/pocket-id/backend/internal/mcp"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	testutils "github.com/pocket-id/pocket-id/backend/internal/utils/testing"
)

func TestUpdateClientToolKeepsFieldsThatWereNotGiven(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	s := &clientService{db: db}
	h := clientHandler{service: s}

	launchURL := "https://photos.example.com"
	client, _, err := s.CreateClient(t.Context(), dto.OidcClientCreateDto{OidcClientUpdateDto: dto.OidcClientUpdateDto{
		Name:                       "Photos",
		CallbackURLs:               []string{"https://photos.example.com/callback"},
		LogoutCallbackURLs:         []string{"https://photos.example.com/logout"},
		PkceEnabled:                true,
		LaunchURL:                  &launchURL,
		AccessTokenDurationMinutes: 30,
		Credentials: dto.OidcClientCredentialsDto{FederatedIdentities: []dto.OidcClientFederatedIdentityDto{
			{Issuer: "https://issuer.example.com", Subject: "photos", JWKS: "https://issuer.example.com/jwks"},
		}},
	}}, "", false)
	require.NoError(t, err)

	group := model.UserGroup{FriendlyName: "Family", Name: "family"}
	require.NoError(t, db.Create(&group).Error)

	output, err := h.updateClientTool(t.Context(), mcp.Call{}, updateClientInput{
		ID:                  client.ID,
		Name:                new("Family photos"),
		AllowedUserGroupIDs: &[]string{group.ID},
	})
	require.NoError(t, err)
	updated := output.(clientResult)

	require.Equal(t, "Family photos", updated.Name)
	require.Equal(t, []string{"https://photos.example.com/callback"}, updated.CallbackURLs)
	require.Equal(t, []string{"https://photos.example.com/logout"}, updated.LogoutCallbackURLs)
	require.True(t, updated.PkceEnabled)
	require.Equal(t, &launchURL, updated.LaunchURL)
	require.EqualValues(t, 30, updated.AccessTokenDurationMinutes)
	require.Len(t, updated.Credentials.FederatedIdentities, 1)
	require.Equal(t, "https://issuer.example.com/jwks", updated.Credentials.FederatedIdentities[0].JWKS)
	require.True(t, updated.IsGroupRestricted)
	require.Len(t, updated.AllowedUserGroups, 1)

	// An empty group list lifts the restriction
	output, err = h.updateClientTool(t.Context(), mcp.Call{}, updateClientInput{ID: client.ID, AllowedUserGroupIDs: &[]string{}})
	require.NoError(t, err)
	require.False(t, output.(clientResult).IsGroupRestricted)
	require.Empty(t, output.(clientResult).AllowedUserGroups)
}

// userIDLogoutTokenSigner sends the user ID as the logout token, so the test can tell who was logged out
type userIDLogoutTokenSigner struct{}

func (userIDLogoutTokenSigner) GenerateLogoutToken(userID string, _ string) (string, error) {
	return userID, nil
}

func TestUpdateClientToolKeepsMembersOfNewlyAllowedGroupsSignedIn(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)

	loggedOutUserIDs := make(chan string, 4)
	logoutServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		loggedOutUserIDs <- r.PostFormValue("logout_token")
	}))
	t.Cleanup(logoutServer.Close)

	var backchannelLogout *backchannellogout.Service
	testutils.NewActorHostForTest(t, func(t *testing.T, h *local.Host) {
		var err error
		backchannelLogout, err = backchannellogout.NewService(db, userIDLogoutTokenSigner{}, logoutServer.Client(), h)
		require.NoError(t, err)
	})
	h := clientHandler{service: &clientService{db: db, backchannelLogout: backchannelLogout}}

	client := model.OidcClient{Name: "Photos", BackchannelLogoutURL: logoutServer.URL}
	require.NoError(t, db.Create(&client).Error)
	member := model.User{Username: "member"}
	require.NoError(t, db.Create(&member).Error)
	outsider := model.User{Username: "outsider"}
	require.NoError(t, db.Create(&outsider).Error)
	group := model.UserGroup{FriendlyName: "Family", Name: "family", Users: []model.User{member}}
	require.NoError(t, db.Create(&group).Error)
	for _, user := range []model.User{member, outsider} {
		require.NoError(t, db.Create(&model.UserAuthorizedOidcClient{UserID: user.ID, ClientID: client.ID}).Error)
	}

	_, err := h.updateClientTool(t.Context(), mcp.Call{}, updateClientInput{ID: client.ID, AllowedUserGroupIDs: &[]string{group.ID}})
	require.NoError(t, err)

	select {
	case userID := <-loggedOutUserIDs:
		require.Equal(t, outsider.ID, userID)
	case <-time.After(10 * time.Second):
		t.Fatal("the user outside the allowed group was not logged out")
	}

	// A logout of the member would be delivered right alongside the outsider's
	select {
	case userID := <-loggedOutUserIDs:
		t.Fatalf("unexpected logout of user %s", userID)
	case <-time.After(500 * time.Millisecond):
	}
}
