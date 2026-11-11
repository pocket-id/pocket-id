package users

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/appconfig"
	"github.com/pocket-id/pocket-id/backend/internal/apperror"
	"github.com/pocket-id/pocket-id/backend/internal/auditlogs"
	"github.com/pocket-id/pocket-id/backend/internal/dto"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
	"github.com/pocket-id/pocket-id/backend/internal/service"
	"github.com/pocket-id/pocket-id/backend/internal/storage"
	testutils "github.com/pocket-id/pocket-id/backend/internal/utils/testing"
)

func newTestUserService(t *testing.T) *Service {
	t.Helper()

	db := testutils.NewDatabaseForTest(t)

	fileStorage, err := storage.NewDatabaseStorage(db)
	require.NoError(t, err)

	return newService(Dependencies{
		DB:           db,
		CustomClaims: service.NewCustomClaimService(db),
		AppAssets:    service.NewAppAssetsService(map[string]string{}, fileStorage),
		FileStorage:  fileStorage,
		AuditLog:     &fakeAuditLogger{},
	})
}

type fakeAuditLogger struct {
	events map[auditlogs.Event][]auditlogs.Data
}

func (f *fakeAuditLogger) Record(_ context.Context, event auditlogs.Event, _ auditlogs.Origin, _ string, data auditlogs.Data, _ *gorm.DB) {
	if f.events == nil {
		f.events = map[auditlogs.Event][]auditlogs.Data{}
	}
	f.events[event] = append(f.events[event], data)
}

func (*fakeAuditLogger) KeepUsername(context.Context, string, string, *gorm.DB) error {
	return nil
}

func createTestGroup(t *testing.T, db *gorm.DB, name, friendlyName string) model.UserGroup {
	t.Helper()

	group := model.UserGroup{Name: name, FriendlyName: friendlyName}
	require.NoError(t, db.Create(&group).Error)
	return group
}

func getTestGroup(t *testing.T, db *gorm.DB, id string) model.UserGroup {
	t.Helper()

	var group model.UserGroup
	require.NoError(t, db.Preload("Users").First(&group, "id = ?", id).Error)
	return group
}

func TestGetUserReturnsUserNotFound(t *testing.T) {
	userService := newTestUserService(t)

	_, err := userService.GetUser(t.Context(), "missing-user")
	require.True(t, apperror.IsCode(err, apperror.CodeUserNotFound))
}

func TestUpdateProfilePictureRejectsInvalidImageData(t *testing.T) {
	userService := newTestUserService(t)
	config := &appconfig.AppConfigModel{RequireUserEmail: "false"}
	user, err := userService.CreateUser(t.Context(), config, dto.UserCreateDto{
		ID:       uuid.NewV4().String(),
		Username: "image-test",
	}, auditlogs.Origin{})
	require.NoError(t, err)

	err = userService.UpdateProfilePicture(
		t.Context(),
		user.ID,
		strings.NewReader("not an image"),
	)

	require.True(t, apperror.IsCode(err, apperror.CodeInvalidImage))
}

func TestProfilePictureUpdatesRejectMissingUser(t *testing.T) {
	userService := newTestUserService(t)
	missingUserID := uuid.NewV4().String()

	err := userService.UpdateProfilePicture(t.Context(), missingUserID, strings.NewReader("not an image"))
	require.True(t, apperror.IsCode(err, apperror.CodeUserNotFound))

	err = userService.ResetProfilePicture(t.Context(), missingUserID)
	require.True(t, apperror.IsCode(err, apperror.CodeUserNotFound))
}

func TestCreateUserBumpsGroupUpdatedAt(t *testing.T) {
	config := &appconfig.AppConfigModel{RequireUserEmail: "false"}
	userService := newTestUserService(t)

	group := createTestGroup(t, userService.db, "members", "Members")
	require.Nil(t, group.UpdatedAt, "a freshly created group has no UpdatedAt yet")

	// Create a user that is a member of the group
	// This mirrors signing up via an invite link that adds the user to a group
	email := "member@example.com"
	_, err := userService.CreateUser(t.Context(), config, dto.UserCreateDto{
		Username:     "member",
		Email:        &email,
		FirstName:    "Group",
		LastName:     "Member",
		UserGroupIds: []string{group.ID},
	}, auditlogs.Origin{})
	require.NoError(t, err)

	// The group's UpdatedAt must now be set
	updated := getTestGroup(t, userService.db, group.ID)
	require.NotNil(t, updated.UpdatedAt, "creating a group member must bump the group's UpdatedAt")
	require.False(t, updated.UpdatedAt.ToTime().Before(updated.CreatedAt.ToTime()), "group UpdatedAt must not predate its CreatedAt after a membership change")
	require.Len(t, updated.Users, 1, "the user should be a member of the group")
}

func TestCreateUserBumpsDefaultGroupUpdatedAt(t *testing.T) {
	config := &appconfig.AppConfigModel{RequireUserEmail: "false"}
	userService := newTestUserService(t)

	group := createTestGroup(t, userService.db, "default", "Default")
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
	}, auditlogs.Origin{})
	require.NoError(t, err)

	updated := getTestGroup(t, userService.db, group.ID)
	require.NotNil(t, updated.UpdatedAt, "adding a default group member must bump the group's UpdatedAt")
	require.Len(t, updated.Users, 1)
}

func TestUpdateUserGroupsBumpsRemovedGroupUpdatedAt(t *testing.T) {
	config := &appconfig.AppConfigModel{RequireUserEmail: "false"}
	userService := newTestUserService(t)

	oldGroup := createTestGroup(t, userService.db, "old", "Old")
	newGroup := createTestGroup(t, userService.db, "new", "New")

	user, err := userService.CreateUser(t.Context(), config, dto.UserCreateDto{
		Username:     "mover",
		FirstName:    "Group",
		LastName:     "Mover",
		UserGroupIds: []string{oldGroup.ID},
	}, auditlogs.Origin{})
	require.NoError(t, err)

	// Backdate both groups so a bump is visible regardless of the timestamp precision
	past := time.Now().Add(-time.Hour)
	require.NoError(t, userService.db.Model(&model.UserGroup{}).Where("id IN ?", []string{oldGroup.ID, newGroup.ID}).Update("updated_at", datatype.DateTime(past)).Error)

	// Move the user from the old group to the new one, as the user's group selection does
	_, err = userService.UpdateUserGroups(t.Context(), user.ID, []string{newGroup.ID}, auditlogs.Origin{})
	require.NoError(t, err)

	for _, groupID := range []string{oldGroup.ID, newGroup.ID} {
		updated := getTestGroup(t, userService.db, groupID)
		require.True(t, updated.UpdatedAt.ToTime().After(past), "group %s changed membership, so its UpdatedAt must be bumped", updated.Name)
	}
}

func TestDeleteUserBumpsGroupUpdatedAt(t *testing.T) {
	config := &appconfig.AppConfigModel{RequireUserEmail: "false"}
	userService := newTestUserService(t)

	group := createTestGroup(t, userService.db, "members", "Members")

	user, err := userService.CreateUser(t.Context(), config, dto.UserCreateDto{
		Username:     "leaver",
		FirstName:    "Group",
		LastName:     "Leaver",
		UserGroupIds: []string{group.ID},
	}, auditlogs.Origin{})
	require.NoError(t, err)

	// Backdate the group so a bump is visible regardless of the timestamp precision
	past := time.Now().Add(-time.Hour)
	require.NoError(t, userService.db.Model(&model.UserGroup{}).Where("id = ?", group.ID).Update("updated_at", datatype.DateTime(past)).Error)

	err = userService.DeleteUser(t.Context(), config, user.ID, false, auditlogs.Origin{})
	require.NoError(t, err)

	updated := getTestGroup(t, userService.db, group.ID)
	require.Empty(t, updated.Users)
	require.True(t, updated.UpdatedAt.ToTime().After(past), "deleting a member must bump the group's UpdatedAt")
}

type recordingEmailVerificationSender struct {
	sent       []string
	requested  []string
	allowErr   error
	requestErr error
}

func (r *recordingEmailVerificationSender) AllowSend(_ context.Context, _ string) error {
	return r.allowErr
}

func (r *recordingEmailVerificationSender) Send(_ context.Context, _ *appconfig.AppConfigModel, userID string) error {
	r.sent = append(r.sent, userID)
	return nil
}

func (r *recordingEmailVerificationSender) RequestEmailChange(_ context.Context, _ *appconfig.AppConfigModel, _, newEmail string) error {
	r.requested = append(r.requested, newEmail)
	return r.requestErr
}

func TestUpdateOwnEmailWithVerification(t *testing.T) {
	// setup creates a user with the given verification state and an update that changes only their email
	setup := func(t *testing.T, mode string, emailVerified bool) (*Service, *recordingEmailVerificationSender, *appconfig.AppConfigModel, model.User, dto.UserCreateDto) {
		t.Helper()

		config := appconfig.NewTestConfig(nil)
		config.EmailVerification = appconfig.AppConfigValue(mode)
		userService := newTestUserService(t)
		sender := &recordingEmailVerificationSender{}
		userService.emailVerification = sender

		user, err := userService.CreateUser(t.Context(), config, dto.UserCreateDto{
			Username:      "changer",
			Email:         new("old@example.com"),
			EmailVerified: emailVerified,
			FirstName:     "Email",
			LastName:      "Changer",
		}, auditlogs.Origin{})
		require.NoError(t, err)

		update := dto.UserCreateDto{
			Username:  user.Username,
			Email:     new("new@example.com"),
			FirstName: user.FirstName,
			LastName:  user.LastName,
		}
		return userService, sender, config, user, update
	}

	t.Run("a verified email stays in effect until the new address is confirmed", func(t *testing.T) {
		userService, sender, config, user, update := setup(t, appconfig.EmailVerificationRequired, true)

		updated, err := userService.UpdateUser(t.Context(), config, user.ID, update, true, false, auditlogs.Origin{})
		require.NoError(t, err)

		require.Equal(t, "old@example.com", *updated.Email)
		require.True(t, updated.EmailVerified)
		require.Equal(t, []string{"new@example.com"}, sender.requested)
		require.Empty(t, sender.sent)
		require.Empty(t, userService.auditLog.(*fakeAuditLogger).events[auditlogs.EventEmailChanged])
	})

	t.Run("a deferred change to an address another user has is rejected right away", func(t *testing.T) {
		userService, sender, config, user, update := setup(t, appconfig.EmailVerificationRequired, true)
		_, err := userService.CreateUser(t.Context(), config, dto.UserCreateDto{Username: "other", Email: new("new@example.com"), FirstName: "Other"}, auditlogs.Origin{})
		require.NoError(t, err)

		_, err = userService.UpdateUser(t.Context(), config, user.ID, update, true, false, auditlogs.Origin{})
		require.True(t, apperror.IsCode(err, apperror.CodeAlreadyInUse))
		require.Empty(t, sender.requested)
	})

	t.Run("an unverified email is replaced right away and the new one has to be verified", func(t *testing.T) {
		userService, sender, config, user, update := setup(t, appconfig.EmailVerificationRequired, false)

		updated, err := userService.UpdateUser(t.Context(), config, user.ID, update, true, false, auditlogs.Origin{})
		require.NoError(t, err)

		require.Equal(t, "new@example.com", *updated.Email)
		require.False(t, updated.EmailVerified)
		require.Equal(t, []string{user.ID}, sender.sent)
		require.Empty(t, sender.requested)
		require.Equal(t, []auditlogs.Data{{"email": "new@example.com", "previousEmail": "old@example.com"}}, userService.auditLog.(*fakeAuditLogger).events[auditlogs.EventEmailChanged])
	})

	t.Run("the other fields are saved while the email change waits for confirmation", func(t *testing.T) {
		userService, _, config, user, update := setup(t, appconfig.EmailVerificationRequired, true)
		update.FirstName = "Renamed"

		updated, err := userService.UpdateUser(t.Context(), config, user.ID, update, true, false, auditlogs.Origin{})
		require.NoError(t, err)

		require.Equal(t, "Renamed", updated.FirstName)
		require.Equal(t, "old@example.com", *updated.Email)
	})

	t.Run("a throttled email change rejects the whole update", func(t *testing.T) {
		userService, sender, config, user, update := setup(t, appconfig.EmailVerificationRequired, true)
		sender.allowErr = apperror.TooManyRequests()
		update.FirstName = "Renamed"

		_, err := userService.UpdateUser(t.Context(), config, user.ID, update, true, false, auditlogs.Origin{})
		require.True(t, apperror.IsCode(err, apperror.CodeRateLimited))
		require.Empty(t, sender.requested)

		stored, err := userService.GetUser(t.Context(), user.ID)
		require.NoError(t, err)
		require.Equal(t, user.FirstName, stored.FirstName)
	})

	t.Run("a change link that can't be sent is reported apart from the saved fields", func(t *testing.T) {
		userService, sender, config, user, update := setup(t, appconfig.EmailVerificationRequired, true)
		sender.requestErr = errors.New("smtp unavailable")
		update.FirstName = "Renamed"

		updated, err := userService.UpdateUser(t.Context(), config, user.ID, update, true, false, auditlogs.Origin{})
		require.True(t, apperror.IsCode(err, apperror.CodeEmailChangeNotStarted))
		require.Equal(t, "Renamed", updated.FirstName)

		stored, err := userService.GetUser(t.Context(), user.ID)
		require.NoError(t, err)
		require.Equal(t, "Renamed", stored.FirstName)
		require.Equal(t, "old@example.com", *stored.Email)
	})

	t.Run("no change link is sent when the user may not edit their own account", func(t *testing.T) {
		userService, sender, config, user, update := setup(t, appconfig.EmailVerificationRequired, true)
		config.AllowOwnAccountEdit = "false"

		updated, err := userService.UpdateUser(t.Context(), config, user.ID, update, true, false, auditlogs.Origin{})
		require.NoError(t, err)

		require.Equal(t, "old@example.com", *updated.Email)
		require.Empty(t, sender.requested)
		require.Empty(t, sender.sent)
	})

	t.Run("an admin changing another user's email replaces it right away", func(t *testing.T) {
		userService, sender, config, user, update := setup(t, appconfig.EmailVerificationRequired, true)
		update.EmailVerified = true

		updated, err := userService.UpdateUser(t.Context(), config, user.ID, update, false, false, auditlogs.Origin{})
		require.NoError(t, err)

		require.Equal(t, "new@example.com", *updated.Email)
		require.Empty(t, sender.requested)
		require.Empty(t, sender.sent)
	})

	t.Run("the email is replaced right away when verification is optional", func(t *testing.T) {
		userService, sender, config, user, update := setup(t, appconfig.EmailVerificationOptional, true)

		updated, err := userService.UpdateUser(t.Context(), config, user.ID, update, true, false, auditlogs.Origin{})
		require.NoError(t, err)

		require.Equal(t, "new@example.com", *updated.Email)
		require.False(t, updated.EmailVerified)
		require.Empty(t, sender.sent)
		require.Empty(t, sender.requested)
	})
}

func TestMarkingEmailVerifiedClearsSignupDeadline(t *testing.T) {
	config := appconfig.NewTestConfig(nil)
	userService := newTestUserService(t)

	user, err := userService.CreateUser(t.Context(), config, dto.UserCreateDto{
		Username:  "pending",
		Email:     new("pending@example.com"),
		FirstName: "Pending",
		LastName:  "User",
	}, auditlogs.Origin{})
	require.NoError(t, err)
	deadline := datatype.DateTime(time.Now().Add(time.Hour))
	require.NoError(t, userService.db.Model(&model.User{}).Where("id = ?", user.ID).Update("email_verification_deadline", deadline).Error)

	updated, err := userService.UpdateUser(t.Context(), config, user.ID, dto.UserCreateDto{
		Username:      user.Username,
		Email:         user.Email,
		EmailVerified: true,
		FirstName:     user.FirstName,
		LastName:      user.LastName,
	}, false, false, auditlogs.Origin{})
	require.NoError(t, err)
	require.True(t, updated.EmailVerified)

	stored, err := userService.GetUser(t.Context(), user.ID)
	require.NoError(t, err)
	require.Nil(t, stored.EmailVerificationDeadline)
}

func TestUpdateUserReturnsGroupsAndCustomClaims(t *testing.T) {
	userService := newTestUserService(t)
	group := createTestGroup(t, userService.db, "family", "Family")
	user := model.User{
		Username:     "member",
		UserGroups:   []model.UserGroup{group},
		CustomClaims: []model.CustomClaim{{Key: "office", Value: "Zurich"}},
	}
	require.NoError(t, userService.db.Create(&user).Error)

	updated, err := userService.UpdateUser(t.Context(), &appconfig.AppConfigModel{RequireUserEmail: "false"}, user.ID, dto.UserCreateDto{Username: "member", FirstName: "Updated"}, false, false, auditlogs.Origin{})
	require.NoError(t, err)

	require.Equal(t, "Updated", updated.FirstName)
	require.Len(t, updated.UserGroups, 1)
	require.Equal(t, group.ID, updated.UserGroups[0].ID)
	require.Len(t, updated.CustomClaims, 1)
	require.Equal(t, "office", updated.CustomClaims[0].Key)
}

func TestCreateUserReturnsDefaultCustomClaims(t *testing.T) {
	userService := newTestUserService(t)
	config := &appconfig.AppConfigModel{RequireUserEmail: "false", SignupDefaultCustomClaims: `[{"key":"office","value":"Zurich"}]`}

	user, err := userService.CreateUser(t.Context(), config, dto.UserCreateDto{Username: "newcomer", FirstName: "New"}, auditlogs.Origin{})
	require.NoError(t, err)

	require.Len(t, user.CustomClaims, 1)
	require.Equal(t, "office", user.CustomClaims[0].Key)
}

func TestAccessChangesAreRecorded(t *testing.T) {
	config := &appconfig.AppConfigModel{RequireUserEmail: "false"}
	userService := newTestUserService(t)
	auditLog := userService.auditLog.(*fakeAuditLogger)
	oldGroup := createTestGroup(t, userService.db, "old", "Old")
	newGroup := createTestGroup(t, userService.db, "new", "New")
	user := model.User{Username: "member", UserGroups: []model.UserGroup{oldGroup}}
	require.NoError(t, userService.db.Create(&user).Error)

	_, err := userService.UpdateUser(t.Context(), config, user.ID, dto.UserCreateDto{Username: "member", IsAdmin: true, Disabled: true}, false, false, auditlogs.Origin{})
	require.NoError(t, err)
	_, err = userService.UpdateUserGroups(t.Context(), user.ID, []string{newGroup.ID}, auditlogs.Origin{})
	require.NoError(t, err)
	require.NoError(t, userService.DeleteUser(t.Context(), config, user.ID, false, auditlogs.Origin{}))

	assert.Len(t, auditLog.events[auditlogs.EventAdminGranted], 1)
	assert.Len(t, auditLog.events[auditlogs.EventAccountDisabled], 1)
	assert.Equal(t, []auditlogs.Data{{"groupName": "New"}}, auditLog.events[auditlogs.EventGroupMemberAdded])
	assert.Equal(t, []auditlogs.Data{{"groupName": "Old"}}, auditLog.events[auditlogs.EventGroupMemberRemoved])
	assert.Len(t, auditLog.events[auditlogs.EventAccountDeleted], 1)
}
