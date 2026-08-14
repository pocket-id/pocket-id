package service

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/pocket-id/pocket-id/backend/internal/appconfig"
	"github.com/pocket-id/pocket-id/backend/internal/apperror"
	"github.com/pocket-id/pocket-id/backend/internal/dto"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
	"github.com/pocket-id/pocket-id/backend/internal/storage"
	testutils "github.com/pocket-id/pocket-id/backend/internal/utils/testing"
)

func newTestUserService(t *testing.T) (*UserService, *UserGroupService) {
	t.Helper()

	db := testutils.NewDatabaseForTest(t)

	fileStorage, err := storage.NewDatabaseStorage(db)
	require.NoError(t, err)

	userService := NewUserService(
		db,
		nil,
		nil,
		NewCustomClaimService(db),
		NewAppImagesService(map[string]string{}, fileStorage),
		nil,
		fileStorage,
	)
	groupService := NewUserGroupService(db, nil)

	return userService, groupService
}

func TestUserAndGroupLookupsReturnSpecificNotFoundErrors(t *testing.T) {
	userService, groupService := newTestUserService(t)

	_, err := userService.GetUser(t.Context(), "missing-user")
	require.True(t, apperror.IsCode(err, apperror.CodeUserNotFound))

	_, err = groupService.Get(t.Context(), "missing-group")
	require.True(t, apperror.IsCode(err, apperror.CodeNotFound))
	var appErr *apperror.Error
	require.ErrorAs(t, err, &appErr)
	require.Equal(t, "User group", appErr.Details()["resource"])
}

func TestUpdateProfilePictureRejectsInvalidImageData(t *testing.T) {
	userService, _ := newTestUserService(t)
	config := &appconfig.AppConfigModel{RequireUserEmail: "false"}
	user, err := userService.CreateUser(t.Context(), config, dto.UserCreateDto{
		ID:       uuid.NewString(),
		Username: "image-test",
	})
	require.NoError(t, err)

	err = userService.UpdateProfilePicture(
		t.Context(),
		user.ID,
		strings.NewReader("not an image"),
	)

	require.True(t, apperror.IsCode(err, apperror.CodeInvalidImage))
}

func TestProfilePictureUpdatesRejectMissingUser(t *testing.T) {
	userService, _ := newTestUserService(t)
	missingUserID := uuid.NewString()

	err := userService.UpdateProfilePicture(t.Context(), missingUserID, strings.NewReader("not an image"))
	require.True(t, apperror.IsCode(err, apperror.CodeUserNotFound))

	err = userService.ResetProfilePicture(t.Context(), missingUserID)
	require.True(t, apperror.IsCode(err, apperror.CodeUserNotFound))
}

func TestAgentSelectorPathChangesRequireNoActiveCredentials(t *testing.T) {
	userService, _ := newTestUserService(t)
	config := &appconfig.AppConfigModel{RequireUserEmail: "false"}
	user, err := userService.CreateUser(t.Context(), config, dto.UserCreateDto{ID: uuid.NewString(), Username: "vex", IsAgent: true})
	require.NoError(t, err)
	require.True(t, user.IsAgent)

	clearAgentSelector := dto.UserCreateDto{Username: user.Username}
	passkey := model.WebauthnCredential{Name: "Existing passkey", CredentialID: []byte("credential"), PublicKey: []byte("public-key"), UserID: user.ID}
	err = userService.db.Create(&passkey).Error
	require.ErrorContains(t, err, "passkeys are not allowed on the runtime authentication path")

	publicKey := make([]byte, 32)
	runtimeCredential := model.RuntimeCredential{Name: "Runtime", Algorithm: model.RuntimeCredentialAlgorithmEd25519, PublicKey: publicKey, UserID: user.ID}
	require.NoError(t, userService.db.Create(&runtimeCredential).Error)
	_, err = userService.UpdateUser(t.Context(), config, user.ID, clearAgentSelector, false, false)
	require.ErrorIs(t, err, apperror.AuthenticationPathChangeBlocked())

	now := datatype.DateTime(time.Now())
	require.NoError(t, userService.db.Model(&runtimeCredential).Update("revoked_at", now).Error)
	updated, err := userService.UpdateUser(t.Context(), config, user.ID, clearAgentSelector, false, false)
	require.NoError(t, err)
	require.False(t, updated.IsAgent)

	passkey = model.WebauthnCredential{Name: "Existing passkey", CredentialID: []byte("credential"), PublicKey: []byte("public-key"), UserID: user.ID}
	require.NoError(t, userService.db.Create(&passkey).Error)
	_, err = userService.UpdateUser(t.Context(), config, user.ID, dto.UserCreateDto{Username: user.Username, IsAgent: true}, false, false)
	require.ErrorIs(t, err, apperror.AuthenticationPathChangeBlocked())
}

func TestCreateUserBumpsGroupUpdatedAt(t *testing.T) {
	config := &appconfig.AppConfigModel{RequireUserEmail: "false"}
	userService, groupService := newTestUserService(t)

	group, err := groupService.Create(t.Context(), dto.UserGroupCreateDto{
		Name:         "members",
		FriendlyName: "Members",
	})
	require.NoError(t, err)
	require.Nil(t, group.UpdatedAt, "a freshly created group has no UpdatedAt yet")

	// Create a user that is a member of the group
	// This mirrors signing up via an invite link that adds the user to a group
	email := "member@example.com"
	_, err = userService.CreateUser(t.Context(), config, dto.UserCreateDto{
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
	config := &appconfig.AppConfigModel{RequireUserEmail: "false"}
	userService, groupService := newTestUserService(t)

	group, err := groupService.Create(t.Context(), dto.UserGroupCreateDto{
		Name:         "default",
		FriendlyName: "Default",
	})
	require.NoError(t, err)
	require.Nil(t, group.UpdatedAt)

	// Configure the group as a default signup group
	defaultGroups, err := json.Marshal([]string{group.ID})
	require.NoError(t, err)
	config.SignupDefaultUserGroupIDs = appconfig.AppConfigValue(defaultGroups)

	// Create a user without explicit group IDs, so the default groups apply
	email := "default@example.com"
	_, err = userService.CreateUser(t.Context(), config, dto.UserCreateDto{
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
