package oidc

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/auditlogs"
	"github.com/pocket-id/pocket-id/backend/internal/dto"
	"github.com/pocket-id/pocket-id/backend/internal/mcp"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	testutils "github.com/pocket-id/pocket-id/backend/internal/utils/testing"
)

func TestUpdateClientToolKeepsFieldsThatWereNotGiven(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	s := &clientService{db: db, auditLog: &fakeAuditLogger{}}
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
	}}, auditlogs.Origin{}, false)
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

// groupRecordingBackchannelLogout records the client's allowed groups whenever users who lost group access are logged out
type groupRecordingBackchannelLogout struct {
	BackchannelLogout
	db            *gorm.DB
	allowedGroups [][]string
}

func (b *groupRecordingBackchannelLogout) NotifyLostGroupAccess(ctx context.Context, _ []string, clientID string) {
	var groupIDs []string
	b.db.WithContext(ctx).Table("oidc_clients_allowed_user_groups").Where("oidc_client_id = ?", clientID).Pluck("user_group_id", &groupIDs)
	b.allowedGroups = append(b.allowedGroups, groupIDs)
}

func TestUpdateClientToolKeepsMembersOfNewlyAllowedGroupsSignedIn(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	backchannelLogout := &groupRecordingBackchannelLogout{db: db}
	h := clientHandler{service: &clientService{db: db, backchannelLogout: backchannelLogout, auditLog: &fakeAuditLogger{}}}

	client := model.OidcClient{Name: "Photos"}
	require.NoError(t, db.Create(&client).Error)
	group := model.UserGroup{FriendlyName: "Family", Name: "family"}
	require.NoError(t, db.Create(&group).Error)

	_, err := h.updateClientTool(t.Context(), mcp.Call{}, updateClientInput{ID: client.ID, AllowedUserGroupIDs: &[]string{group.ID}})
	require.NoError(t, err)

	require.NotEmpty(t, backchannelLogout.allowedGroups)
	for _, groupIDs := range backchannelLogout.allowedGroups {
		require.Equal(t, []string{group.ID}, groupIDs, "users were logged out before the group's members were allowed in")
	}
}
