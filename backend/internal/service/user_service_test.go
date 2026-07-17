package service

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pocket-id/pocket-id/backend/internal/dto"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	"github.com/pocket-id/pocket-id/backend/internal/storage"
	testutils "github.com/pocket-id/pocket-id/backend/internal/utils/testing"
)

func newTestUserService(t *testing.T, appConfig *AppConfigService) (*UserService, *UserGroupService) {
	t.Helper()

	db := testutils.NewDatabaseForTest(t)

	fileStorage, err := storage.NewDatabaseStorage(db)
	require.NoError(t, err)

	userService := NewUserService(
		db,
		nil,
		nil,
		nil,
		appConfig,
		NewCustomClaimService(db),
		NewAppImagesService(map[string]string{}, fileStorage),
		nil,
		fileStorage,
	)
	groupService := NewUserGroupService(db, appConfig, nil)

	return userService, groupService
}

func TestCreateUserBumpsGroupUpdatedAt(t *testing.T) {
	appConfig := NewTestAppConfigService(&model.AppConfig{
		RequireUserEmail: model.AppConfigVariable{Value: "false"},
	})
	userService, groupService := newTestUserService(t, appConfig)

	group, err := groupService.Create(t.Context(), dto.UserGroupCreateDto{
		Name:         "members",
		FriendlyName: "Members",
	})
	require.NoError(t, err)
	require.Nil(t, group.UpdatedAt, "a freshly created group has no UpdatedAt yet")

	// Create a user that is a member of the group
	// This mirrors signing up via an invite link that adds the user to a group
	email := "member@example.com"
	_, err = userService.CreateUser(t.Context(), dto.UserCreateDto{
		Username:     "member",
		Email:        &email,
		FirstName:    "Group",
		LastName:     "Member",
		UserGroupIds: []string{group.ID},
	})
	require.NoError(t, err)

	// The group's UpdatedAt must now be set
	updated, err := groupService.Get(t.Context(), group.ID)
	require.NoError(t, err)
	require.NotNil(t, updated.UpdatedAt, "creating a group member must bump the group's UpdatedAt")
	require.False(t, updated.LastModified().Before(updated.CreatedAt.ToTime()), "group LastModified must not predate its CreatedAt after a membership change")
	require.Len(t, updated.Users, 1, "the user should be a member of the group")
}

func TestCreateUserBumpsDefaultGroupUpdatedAt(t *testing.T) {
	appConfig := NewTestAppConfigService(&model.AppConfig{
		RequireUserEmail: model.AppConfigVariable{Value: "false"},
	})
	userService, groupService := newTestUserService(t, appConfig)

	group, err := groupService.Create(t.Context(), dto.UserGroupCreateDto{
		Name:         "default",
		FriendlyName: "Default",
	})
	require.NoError(t, err)
	require.Nil(t, group.UpdatedAt)

	// Configure the group as a default signup group
	defaultGroups, err := json.Marshal([]string{group.ID})
	require.NoError(t, err)
	appConfig.dbConfig.Load().SignupDefaultUserGroupIDs.Value = string(defaultGroups)

	// Create a user without explicit group IDs, so the default groups apply
	email := "default@example.com"
	_, err = userService.CreateUser(t.Context(), dto.UserCreateDto{
		Username:  "defaultmember",
		Email:     &email,
		FirstName: "Default",
		LastName:  "Member",
	})
	require.NoError(t, err)

	updated, err := groupService.Get(t.Context(), group.ID)
	require.NoError(t, err)
	require.NotNil(t, updated.UpdatedAt, "adding a default group member must bump the group's UpdatedAt")
	require.Len(t, updated.Users, 1)
}
