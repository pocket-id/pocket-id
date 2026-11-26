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

	"github.com/pocket-id/pocket-id/backend/internal/appassets"
	"github.com/pocket-id/pocket-id/backend/internal/appconfig"
	"github.com/pocket-id/pocket-id/backend/internal/apperror"
	"github.com/pocket-id/pocket-id/backend/internal/auditlogs"
	"github.com/pocket-id/pocket-id/backend/internal/authz"
	"github.com/pocket-id/pocket-id/backend/internal/customclaims"
	"github.com/pocket-id/pocket-id/backend/internal/dto"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
	"github.com/pocket-id/pocket-id/backend/internal/storage"
	testutils "github.com/pocket-id/pocket-id/backend/internal/utils/testing"
)

func newTestUserService(t *testing.T) *Service {
	t.Helper()

	return newTestUserServiceOn(t, testutils.NewDatabaseForTest(t))
}

func newTestUserServiceOn(t *testing.T, db *gorm.DB) *Service {
	t.Helper()

	fileStorage, err := storage.NewDatabaseStorage(db)
	require.NoError(t, err)
	appAssets, err := appassets.New(t.Context(), fileStorage)
	require.NoError(t, err)

	return newService(Dependencies{
		DB:           db,
		CustomClaims: customclaims.New(db),
		AppAssets:    appAssets,
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

func TestCanEditOwnProfilePicture(t *testing.T) {
	ldapID := "ldap-id"
	ldapUser := &model.User{LdapID: &ldapID}
	cfg := func(override, ownEdit appconfig.AppConfigValue) *appconfig.AppConfigModel {
		return &appconfig.AppConfigModel{LdapEnabled: "true", LdapAllowProfilePictureOverride: override, AllowOwnAccountEdit: ownEdit}
	}

	assert.False(t, canEditOwnProfilePicture(cfg("false", "true"), ldapUser))
	assert.False(t, canEditOwnProfilePicture(cfg("true", "false"), ldapUser))
	assert.True(t, canEditOwnProfilePicture(cfg("true", "true"), ldapUser))
	assert.False(t, canEditOwnProfilePicture(cfg("false", "false"), &model.User{}))
	assert.True(t, canEditOwnProfilePicture(cfg("false", "true"), &model.User{}))
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
	allow      func(ctx context.Context) error
	requestErr error
}

func (r *recordingEmailVerificationSender) AllowSend(ctx context.Context, _ string) error {
	if r.allow != nil {
		return r.allow(ctx)
	}
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
	// A user changes their own email in their browser session
	session := auditlogs.Origin{Principal: authz.Principal{Kind: authz.KindSession}}

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

		updated, err := userService.UpdateUser(t.Context(), config, user.ID, dto.Replace(update), true, false, session)
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

		_, err = userService.UpdateUser(t.Context(), config, user.ID, dto.Replace(update), true, false, session)
		require.True(t, apperror.IsCode(err, apperror.CodeAlreadyInUse))
		require.Empty(t, sender.requested)
	})

	t.Run("an unverified email is replaced right away and the new one has to be verified", func(t *testing.T) {
		userService, sender, config, user, update := setup(t, appconfig.EmailVerificationRequired, false)

		updated, err := userService.UpdateUser(t.Context(), config, user.ID, dto.Replace(update), true, false, session)
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

		updated, err := userService.UpdateUser(t.Context(), config, user.ID, dto.Replace(update), true, false, session)
		require.NoError(t, err)

		require.Equal(t, "Renamed", updated.FirstName)
		require.Equal(t, "old@example.com", *updated.Email)
	})

	t.Run("a throttled email change rejects the whole update", func(t *testing.T) {
		userService, sender, config, user, update := setup(t, appconfig.EmailVerificationRequired, true)
		sender.allowErr = apperror.TooManyRequests()
		update.FirstName = "Renamed"

		_, err := userService.UpdateUser(t.Context(), config, user.ID, dto.Replace(update), true, false, session)
		require.True(t, apperror.IsCode(err, apperror.CodeRateLimited))
		require.Empty(t, sender.requested)

		stored, err := userService.GetUser(t.Context(), user.ID)
		require.NoError(t, err)
		require.Equal(t, user.FirstName, stored.FirstName)
	})

	t.Run("the send limiter can write to the database", func(t *testing.T) {
		userService, sender, config, user, update := setup(t, appconfig.EmailVerificationRequired, true)

		// The limiter is an actor that persists its state, which waits for the SQLite write lock
		sender.allow = func(ctx context.Context) error {
			ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
			defer cancel()
			return userService.db.WithContext(ctx).Exec("CREATE TABLE IF NOT EXISTS limiter_probe (id INTEGER)").Error
		}

		_, err := userService.UpdateUser(t.Context(), config, user.ID, dto.Replace(update), true, false, session)
		require.NoError(t, err)
		require.Equal(t, []string{"new@example.com"}, sender.requested)
	})

	t.Run("a change link that can't be sent is reported apart from the saved fields", func(t *testing.T) {
		userService, sender, config, user, update := setup(t, appconfig.EmailVerificationRequired, true)
		sender.requestErr = errors.New("smtp unavailable")
		update.FirstName = "Renamed"

		updated, err := userService.UpdateUser(t.Context(), config, user.ID, dto.Replace(update), true, false, session)
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

		updated, err := userService.UpdateUser(t.Context(), config, user.ID, dto.Replace(update), true, false, session)
		require.NoError(t, err)

		require.Equal(t, "old@example.com", *updated.Email)
		require.Empty(t, sender.requested)
		require.Empty(t, sender.sent)
	})

	t.Run("an admin changing another user's email replaces it right away", func(t *testing.T) {
		userService, sender, config, user, update := setup(t, appconfig.EmailVerificationRequired, true)
		update.EmailVerified = true

		updated, err := userService.UpdateUser(t.Context(), config, user.ID, dto.Replace(update), false, false, auditlogs.Origin{})
		require.NoError(t, err)

		require.Equal(t, "new@example.com", *updated.Email)
		require.Empty(t, sender.requested)
		require.Empty(t, sender.sent)
	})

	t.Run("the email is replaced right away when verification is optional", func(t *testing.T) {
		userService, sender, config, user, update := setup(t, appconfig.EmailVerificationOptional, true)

		updated, err := userService.UpdateUser(t.Context(), config, user.ID, dto.Replace(update), true, false, session)
		require.NoError(t, err)

		require.Equal(t, "new@example.com", *updated.Email)
		require.False(t, updated.EmailVerified)
		require.Empty(t, sender.sent)
		require.Empty(t, sender.requested)
	})
}

type denyingLimiter struct {
	calls int
}

func (l *denyingLimiter) Allow(context.Context, string) (bool, time.Duration, error) {
	l.calls++
	return false, time.Minute, nil
}

func TestOwnUsernameAndEmailChangesAreThrottled(t *testing.T) {
	config := appconfig.NewTestConfig(nil)
	userService := newTestUserService(t)
	limiter := &denyingLimiter{}
	userService.ownAccountLimiter = limiter

	user, err := userService.CreateUser(t.Context(), config, dto.UserCreateDto{Username: "prober", Email: new("prober@example.com"), FirstName: "Prober"}, auditlogs.Origin{})
	require.NoError(t, err)
	update := dto.UserCreateDto{Username: user.Username, Email: user.Email, FirstName: "Renamed"}

	_, err = userService.UpdateUser(t.Context(), config, user.ID, dto.Replace(update), true, false, auditlogs.Origin{})
	require.NoError(t, err, "an update that keeps the username and email must not be throttled")
	require.Zero(t, limiter.calls)

	withUsername := update
	withUsername.Username = "admin"
	_, err = userService.UpdateUser(t.Context(), config, user.ID, dto.Replace(withUsername), true, false, auditlogs.Origin{})
	require.True(t, apperror.IsCode(err, apperror.CodeRateLimited))

	withEmail := update
	withEmail.Email = new("someone@example.com")
	_, err = userService.UpdateUser(t.Context(), config, user.ID, dto.Replace(withEmail), true, false, auditlogs.Origin{})
	require.True(t, apperror.IsCode(err, apperror.CodeRateLimited))
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

	updated, err := userService.UpdateUser(t.Context(), config, user.ID, dto.Replace(dto.UserCreateDto{
		Username:      user.Username,
		Email:         user.Email,
		EmailVerified: true,
		FirstName:     user.FirstName,
		LastName:      user.LastName,
	}), false, false, auditlogs.Origin{})
	require.NoError(t, err)
	require.True(t, updated.EmailVerified)

	stored, err := userService.GetUser(t.Context(), user.ID)
	require.NoError(t, err)
	require.Nil(t, stored.EmailVerificationDeadline)
}

// The setting can be turned on while users without an email exist, and an admin must still be able to change or disable them
func TestUpdateUserWithoutEmailWhileEmailsAreRequired(t *testing.T) {
	userService := newTestUserService(t)
	config := &appconfig.AppConfigModel{RequireUserEmail: "true"}
	withoutEmail := model.User{Username: "without-email"}
	require.NoError(t, userService.db.Create(&withoutEmail).Error)
	withEmail := model.User{Username: "with-email", Email: new("with-email@example.com")}
	require.NoError(t, userService.db.Create(&withEmail).Error)

	updated, err := userService.UpdateUser(t.Context(), config, withoutEmail.ID, dto.Replace(dto.UserCreateDto{Username: "without-email", Disabled: true}), false, false, auditlogs.Origin{})
	require.NoError(t, err)
	require.True(t, updated.Disabled)

	_, err = userService.UpdateUser(t.Context(), config, withEmail.ID, dto.Replace(dto.UserCreateDto{Username: "with-email"}), false, false, auditlogs.Origin{})
	require.True(t, apperror.IsCode(err, apperror.CodeValidationFailed))
}

// Relying parties often compare addresses without their letter case, so two accounts must not hold spellings of one address
func TestEmailIsUniqueWithoutRegardToLetterCase(t *testing.T) {
	testutils.ForEachDatabase(t, func(t *testing.T, db *gorm.DB) {
		userService := newTestUserServiceOn(t, db)
		config := &appconfig.AppConfigModel{RequireUserEmail: "false"}

		kim, err := userService.CreateUser(t.Context(), config, dto.UserCreateDto{Username: "kim", Email: new("Kim@example.com")}, auditlogs.Origin{})
		require.NoError(t, err)
		other, err := userService.CreateUser(t.Context(), config, dto.UserCreateDto{Username: "other"}, auditlogs.Origin{})
		require.NoError(t, err)

		_, err = userService.CreateUser(t.Context(), config, dto.UserCreateDto{Username: "third", Email: new("kim@EXAMPLE.com")}, auditlogs.Origin{})
		require.True(t, apperror.IsCode(err, apperror.CodeAlreadyInUse))
		_, err = userService.UpdateUser(t.Context(), config, other.ID, dto.Replace(dto.UserCreateDto{Username: "other", Email: new("KIM@example.com")}), false, false, auditlogs.Origin{})
		require.True(t, apperror.IsCode(err, apperror.CodeAlreadyInUse))

		// The spelling of the own address can still be changed
		updated, err := userService.UpdateUser(t.Context(), config, kim.ID, dto.Replace(dto.UserCreateDto{Username: "kim", Email: new("kim@example.com")}), false, false, auditlogs.Origin{})
		require.NoError(t, err)
		require.Equal(t, "kim@example.com", *updated.Email)
	})
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

	updated, err := userService.UpdateUser(t.Context(), &appconfig.AppConfigModel{RequireUserEmail: "false"}, user.ID, dto.Replace(dto.UserCreateDto{Username: "member", FirstName: "Updated"}), false, false, auditlogs.Origin{})
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

	admin := auditlogs.Origin{Principal: authz.Principal{Scopes: authz.NewScopeSet(authz.AdminAccessWrite)}}
	_, err := userService.UpdateUser(t.Context(), config, user.ID, dto.Replace(dto.UserCreateDto{Username: "member", IsAdmin: true, Disabled: true}), false, false, admin)
	require.NoError(t, err)
	_, err = userService.UpdateUserGroups(t.Context(), user.ID, []string{newGroup.ID}, auditlogs.Origin{})
	require.NoError(t, err)
	require.NoError(t, userService.DeleteUser(t.Context(), config, user.ID, false, admin))

	assert.Len(t, auditLog.events[auditlogs.EventAdminGranted], 1)
	assert.Len(t, auditLog.events[auditlogs.EventAccountDisabled], 1)
	assert.Equal(t, []auditlogs.Data{{"groupName": "New"}}, auditLog.events[auditlogs.EventGroupMemberAdded])
	assert.Equal(t, []auditlogs.Data{{"groupName": "Old"}}, auditLog.events[auditlogs.EventGroupMemberRemoved])
	assert.Len(t, auditLog.events[auditlogs.EventAccountDeleted], 1)
}

func TestManagingAdminsRequiresAdminAccess(t *testing.T) {
	config := &appconfig.AppConfigModel{RequireUserEmail: "false"}
	userService := newTestUserService(t)
	usersWrite := auditlogs.Origin{Principal: authz.Principal{Scopes: authz.NewScopeSet(authz.UsersWrite)}}
	admin := model.User{Username: "admin", IsAdmin: true}
	require.NoError(t, userService.db.Create(&admin).Error)
	member := model.User{Username: "member"}
	require.NoError(t, userService.db.Create(&member).Error)

	requireAdminAccess := func(t *testing.T, err error) {
		t.Helper()
		appErr, ok := errors.AsType[*apperror.Error](err)
		require.True(t, ok, err)
		require.Equal(t, string(authz.AdminAccessWrite), appErr.Details()["required_scope"])
	}

	_, err := userService.CreateUser(t.Context(), config, dto.UserCreateDto{Username: "second-admin", IsAdmin: true}, usersWrite)
	requireAdminAccess(t, err)
	_, err = userService.UpdateUser(t.Context(), config, member.ID, dto.Replace(dto.UserCreateDto{Username: "member", IsAdmin: true}), false, false, usersWrite)
	requireAdminAccess(t, err)
	_, err = userService.UpdateUser(t.Context(), config, admin.ID, dto.Replace(dto.UserCreateDto{Username: "renamed", IsAdmin: true}), false, false, usersWrite)
	requireAdminAccess(t, err)
	requireAdminAccess(t, userService.requireAdminAccessFor(t.Context(), usersWrite.Principal, admin.ID))

	// An admin's own credential without admin access can't move their email to an address where a login code could be received
	ownAccount := auditlogs.Origin{Principal: authz.Principal{UserID: admin.ID, Scopes: authz.NewScopeSet(authz.AccountWrite)}}
	_, err = userService.UpdateUser(t.Context(), config, admin.ID, dto.Replace(dto.UserCreateDto{Username: "admin", IsAdmin: true, Email: new("attacker@example.com")}), true, false, ownAccount)
	requireAdminAccess(t, err)

	// Regular users stay manageable without admin access
	require.NoError(t, userService.requireAdminAccessFor(t.Context(), usersWrite.Principal, member.ID))
	_, err = userService.UpdateUser(t.Context(), config, member.ID, dto.Replace(dto.UserCreateDto{Username: "renamed-member"}), false, false, usersWrite)
	require.NoError(t, err)

	// A key or token that may edit the profile of a regular user can't move their email either
	memberKey := auditlogs.Origin{Principal: authz.Principal{Kind: authz.KindAPIKey, UserID: member.ID, Scopes: authz.NewScopeSet(authz.AccountWrite)}}
	_, err = userService.UpdateUser(t.Context(), config, member.ID, dto.Replace(dto.UserCreateDto{Username: "renamed-member", Email: new("attacker@example.com")}), true, false, memberKey)
	require.True(t, apperror.IsCode(err, apperror.CodeAPIKeyAuthNotAllowed), err)

	// A browser session with admin access also has to be in sudo mode, but only to act on an admin
	session := authz.Principal{Kind: authz.KindSession, Scopes: authz.NewScopeSet(authz.UsersWrite, authz.AdminAccessWrite)}
	err = userService.requireAdminAccessFor(t.Context(), session, admin.ID)
	require.True(t, apperror.IsCode(err, apperror.CodeSudoRequired), err)
	require.NoError(t, userService.requireAdminAccessFor(t.Context(), session, member.ID))
	session.SudoUntil = time.Now().Add(time.Minute)
	require.NoError(t, userService.requireAdminAccessFor(t.Context(), session, admin.ID))
}

func TestGroupMembershipSavesWriteOnlyReferences(t *testing.T) {
	userService := newTestUserService(t)
	config := &appconfig.AppConfigModel{}
	group := model.UserGroup{Name: "family", FriendlyName: "Family"}
	require.NoError(t, userService.db.Create(&group).Error)

	writes := testutils.RecordWrites(t, userService.db)
	user, err := userService.CreateUser(t.Context(), config, dto.UserCreateDto{Username: "member", UserGroupIds: []string{group.ID}}, auditlogs.Origin{})
	require.NoError(t, err)
	assert.NotContains(t, *writes, "create user_groups")

	*writes = nil
	_, err = userService.UpdateUserGroups(t.Context(), user.ID, nil, auditlogs.Origin{})
	require.NoError(t, err)
	assert.NotContains(t, *writes, "create user_groups")
	assert.NotContains(t, *writes, "update users")
}
