package usergroups

import (
	"context"
	"strconv"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/appconfig"
	"github.com/pocket-id/pocket-id/backend/internal/apperror"
	"github.com/pocket-id/pocket-id/backend/internal/auditlogs"
	"github.com/pocket-id/pocket-id/backend/internal/dto"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
	testutils "github.com/pocket-id/pocket-id/backend/internal/utils/testing"
)

func TestGetReturnsUserGroupNotFound(t *testing.T) {
	groupService := newService(testutils.NewDatabaseForTest(t), nil, nil, &fakeAuditLogger{})

	_, err := groupService.Get(t.Context(), "missing-group")
	require.True(t, apperror.IsCode(err, apperror.CodeNotFound))
	var appErr *apperror.Error
	require.ErrorAs(t, err, &appErr)
	require.Equal(t, "User group", appErr.Details()["resource"])
}

func TestUserGroupListSearchMatchesFriendlyName(t *testing.T) {
	groupService := newService(testutils.NewDatabaseForTest(t), nil, nil, &fakeAuditLogger{})

	_, err := groupService.Create(t.Context(), dto.UserGroupCreateDto{
		Name:         "qa_team",
		FriendlyName: "QA Engineering",
	}, auditlogs.Origin{})

	require.NoError(t, err)

	// A search for the friendly name must find the group, since the
	// friendly name is the name shown in the UI.
	groups, pagination, err := groupService.List(t.Context(), "Engineering", utils.ListRequestOptions{})
	require.NoError(t, err)
	require.Len(t, groups, 1)
	require.Equal(t, "QA Engineering", groups[0].FriendlyName)
	require.EqualValues(t, 1, pagination.TotalItems)

	// The internal name must remain searchable too.
	groups, _, err = groupService.List(t.Context(), "qa_team", utils.ListRequestOptions{})
	require.NoError(t, err)
	require.Len(t, groups, 1)
	require.Equal(t, "QA Engineering", groups[0].FriendlyName)

	// A fragment matching neither field returns nothing.
	groups, _, err = groupService.List(t.Context(), "Design", utils.ListRequestOptions{})
	require.NoError(t, err)
	require.Empty(t, groups)
}

func TestChangeUsersKeepsConcurrentChanges(t *testing.T) {
	testutils.ForEachDatabase(t, func(t *testing.T, db *gorm.DB) {
		groupService := newService(db, nil, nil, &fakeAuditLogger{})
		group, err := groupService.Create(t.Context(), dto.UserGroupCreateDto{Name: "family", FriendlyName: "Family"}, auditlogs.Origin{})
		require.NoError(t, err)

		userIDs := make([]string, 10)
		for i := range userIDs {
			user := model.User{Username: "user" + strconv.Itoa(i)}
			require.NoError(t, db.Create(&user).Error)
			userIDs[i] = user.ID
		}

		var wg sync.WaitGroup
		for _, userID := range userIDs {
			wg.Go(func() {
				_, err := groupService.ChangeUsers(t.Context(), group.ID, func(memberIDs []string) []string {
					return append(memberIDs, userID)
				}, auditlogs.Origin{})
				assert.NoError(t, err)
			})
		}
		wg.Wait()

		memberIDs, err := groupService.memberIDs(t.Context(), db, group.ID)
		require.NoError(t, err)
		require.ElementsMatch(t, userIDs, memberIDs)
	})
}

type recordedEntry struct {
	event  auditlogs.Event
	userID string
	data   auditlogs.Data
}

type fakeAuditLogger struct {
	mu      sync.Mutex
	entries []recordedEntry
}

func (f *fakeAuditLogger) Record(_ context.Context, event auditlogs.Event, _ auditlogs.Origin, userID string, data auditlogs.Data, _ *gorm.DB) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.entries = append(f.entries, recordedEntry{event: event, userID: userID, data: data})
}

func TestMembershipAndClientChangesAreRecorded(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	auditLog := &fakeAuditLogger{}
	groupService := newService(db, nil, nil, auditLog)
	leaver := model.User{Username: "leaver"}
	joiner := model.User{Username: "joiner"}
	require.NoError(t, db.Create(&leaver).Error)
	require.NoError(t, db.Create(&joiner).Error)
	client := model.OidcClient{Name: "Photos"}
	require.NoError(t, db.Create(&client).Error)
	group := model.UserGroup{Name: "family", FriendlyName: "Family", Users: []model.User{leaver}}
	require.NoError(t, db.Create(&group).Error)

	_, err := groupService.UpdateUsers(t.Context(), group.ID, []string{joiner.ID}, auditlogs.Origin{})
	require.NoError(t, err)
	_, err = groupService.UpdateAllowedOidcClient(t.Context(), group.ID, dto.UserGroupUpdateAllowedOidcClientsDto{OidcClientIDs: []string{client.ID}}, auditlogs.Origin{})
	require.NoError(t, err)

	assert.Equal(t, []recordedEntry{
		{event: auditlogs.EventGroupMemberAdded, userID: joiner.ID, data: auditlogs.Data{"groupName": "Family"}},
		{event: auditlogs.EventGroupMemberRemoved, userID: leaver.ID, data: auditlogs.Data{"groupName": "Family"}},
		{event: auditlogs.EventClientGroupAdded, data: auditlogs.Data{"clientID": client.ID, "clientName": "Photos", "groupName": "Family"}},
	}, auditLog.entries)
}

func TestUpdatesGroupWithThousandsOfMembers(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	groupService := newService(db, nil, nil, &fakeAuditLogger{})
	group, err := groupService.Create(t.Context(), dto.UserGroupCreateDto{Name: "staff", FriendlyName: "Staff"}, auditlogs.Origin{})
	require.NoError(t, err)

	users := make([]model.User, 2500)
	for i := range users {
		users[i] = model.User{Username: "user" + strconv.Itoa(i)}
	}
	require.NoError(t, db.CreateInBatches(&users, 500).Error)
	userIDs := make([]string, len(users))
	for i, user := range users {
		userIDs[i] = user.ID
	}

	_, err = groupService.UpdateUsers(t.Context(), group.ID, userIDs, auditlogs.Origin{})
	require.NoError(t, err)
	_, err = groupService.Update(t.Context(), &appconfig.AppConfigModel{}, group.ID, dto.UserGroupCreateDto{Name: "staff", FriendlyName: "All staff"})
	require.NoError(t, err)
	_, err = groupService.UpdateAllowedOidcClient(t.Context(), group.ID, dto.UserGroupUpdateAllowedOidcClientsDto{}, auditlogs.Origin{})
	require.NoError(t, err)

	memberIDs, err := groupService.memberIDs(t.Context(), db, group.ID)
	require.NoError(t, err)
	require.ElementsMatch(t, userIDs, memberIDs)
}
