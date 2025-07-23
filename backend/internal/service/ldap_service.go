package service

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/go-ldap/ldap/v3"
	"github.com/google/uuid"
	"golang.org/x/text/unicode/norm"
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/dto"
	"github.com/pocket-id/pocket-id/backend/internal/model"
)

type LdapService struct {
	db               *gorm.DB
	httpClient       *http.Client
	appConfigService *AppConfigService
	userService      *UserService
	groupService     *UserGroupService
}

func NewLdapService(db *gorm.DB, httpClient *http.Client, appConfigService *AppConfigService, userService *UserService, groupService *UserGroupService) *LdapService {
	return &LdapService{
		db:               db,
		httpClient:       httpClient,
		appConfigService: appConfigService,
		userService:      userService,
		groupService:     groupService,
	}
}

func (s *LdapService) createClient() (*ldap.Conn, error) {
	dbConfig := s.appConfigService.GetDbConfig()

	if !dbConfig.LdapEnabled.IsTrue() {
		return nil, fmt.Errorf("LDAP is not enabled")
	}

	// Setup LDAP connection
	client, err := ldap.DialURL(dbConfig.LdapUrl.Value, ldap.DialWithTLSConfig(&tls.Config{
		InsecureSkipVerify: dbConfig.LdapSkipCertVerify.IsTrue(), //nolint:gosec
	}))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to LDAP: %w", err)
	}

	// Bind as service account
	err = client.Bind(dbConfig.LdapBindDn.Value, dbConfig.LdapBindPassword.Value)
	if err != nil {
		return nil, fmt.Errorf("failed to bind to LDAP: %w", err)
	}
	return client, nil
}

func (s *LdapService) SyncAll(ctx context.Context) error {
	// Start a transaction
	tx := s.db.Begin()
	defer func() {
		tx.Rollback()
	}()

	// Setup LDAP connection
	client, err := s.createClient()
	if err != nil {
		return fmt.Errorf("failed to create LDAP client: %w", err)
	}
	defer client.Close()

	err = s.SyncUsers(ctx, tx, client)
	if err != nil {
		return fmt.Errorf("failed to sync users: %w", err)
	}

	err = s.SyncGroups(ctx, tx, client)
	if err != nil {
		return fmt.Errorf("failed to sync groups: %w", err)
	}

	// Commit the changes
	err = tx.Commit().Error
	if err != nil {
		return fmt.Errorf("failed to commit changes to database: %w", err)
	}

	return nil
}

//nolint:gocognit
func (s *LdapService) SyncGroups(ctx context.Context, tx *gorm.DB, client *ldap.Conn) error {
	dbConfig := s.appConfigService.GetDbConfig()

	searchAttrs := []string{
		dbConfig.LdapAttributeGroupName.Value,
		dbConfig.LdapAttributeGroupUniqueIdentifier.Value,
		dbConfig.LdapAttributeGroupMember.Value,
	}

	searchReq := ldap.NewSearchRequest(
		dbConfig.LdapBase.Value,
		ldap.ScopeWholeSubtree,
		0, 0, 0, false,
		dbConfig.LdapUserGroupSearchFilter.Value,
		searchAttrs,
		[]ldap.Control{},
	)
	result, err := client.Search(searchReq)
	if err != nil {
		return fmt.Errorf("failed to query LDAP: %w", err)
	}

	// Create a mapping for groups that exist
	ldapGroupIDs := make(map[string]struct{}, len(result.Entries))

	for _, value := range result.Entries {
		ldapId := convertLdapIdToString(value.GetAttributeValue(dbConfig.LdapAttributeGroupUniqueIdentifier.Value))

		// Skip groups without a valid LDAP ID
		if ldapId == "" {
			slog.Warn("Skipping LDAP group without a valid unique identifier", slog.String("attribute", dbConfig.LdapAttributeGroupUniqueIdentifier.Value))
			continue
		}

		ldapGroupIDs[ldapId] = struct{}{}

		// Try to find the group in the database
		var databaseGroup model.UserGroup
		err = tx.
			WithContext(ctx).
			Where("ldap_id = ?", ldapId).
			First(&databaseGroup).
			Error
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			// This could error with ErrRecordNotFound and we want to ignore that here
			return fmt.Errorf("failed to query for LDAP group ID '%s': %w", ldapId, err)
		}

		// Get group members and add to the correct Group
		groupMembers := value.GetAttributeValues(dbConfig.LdapAttributeGroupMember.Value)
		membersUserId := make([]string, 0, len(groupMembers))
		for _, member := range groupMembers {
			username := getDNProperty(dbConfig.LdapAttributeUserUsername.Value, member)

			// If username extraction fails, try to query LDAP directly for the user
			if username == "" {
				// Query LDAP to get the user by their DN
				userSearchReq := ldap.NewSearchRequest(
					member,
					ldap.ScopeBaseObject,
					0, 0, 0, false,
					"(objectClass=*)",
					[]string{dbConfig.LdapAttributeUserUsername.Value, dbConfig.LdapAttributeUserUniqueIdentifier.Value},
					[]ldap.Control{},
				)

				userResult, err := client.Search(userSearchReq)
				if err != nil || len(userResult.Entries) == 0 {
					log.Printf("Could not resolve group member DN '%s': %v", member, err)
					continue
				}

				username = userResult.Entries[0].GetAttributeValue(dbConfig.LdapAttributeUserUsername.Value)
				if username == "" {
					log.Printf("Could not extract username from group member DN '%s'", member)
					continue
				}
			}

			var databaseUser model.User
			err = tx.
				WithContext(ctx).
				Where("username = ? AND ldap_id IS NOT NULL", norm.NFC.String(username)).
				First(&databaseUser).
				Error
			if errors.Is(err, gorm.ErrRecordNotFound) {
				// The user collides with a non-LDAP user, so we skip it
				continue
			} else if err != nil {
				return fmt.Errorf("failed to query for existing user '%s': %w", username, err)
			}

			membersUserId = append(membersUserId, databaseUser.ID)
		}

		syncGroup := dto.UserGroupCreateDto{
			Name:         value.GetAttributeValue(dbConfig.LdapAttributeGroupName.Value),
			FriendlyName: value.GetAttributeValue(dbConfig.LdapAttributeGroupName.Value),
			LdapID:       ldapId,
		}
		dto.Normalize(syncGroup)

		if databaseGroup.ID == "" {
			newGroup, err := s.groupService.createInternal(ctx, syncGroup, tx)
			if err != nil {
				return fmt.Errorf("failed to create group '%s': %w", syncGroup.Name, err)
			}

			_, err = s.groupService.updateUsersInternal(ctx, newGroup.ID, membersUserId, tx)
			if err != nil {
				return fmt.Errorf("failed to sync users for group '%s': %w", syncGroup.Name, err)
			}
		} else {
			_, err = s.groupService.updateInternal(ctx, databaseGroup.ID, syncGroup, true, tx)
			if err != nil {
				return fmt.Errorf("failed to update group '%s': %w", syncGroup.Name, err)
			}

			_, err = s.groupService.updateUsersInternal(ctx, databaseGroup.ID, membersUserId, tx)
			if err != nil {
				return fmt.Errorf("failed to sync users for group '%s': %w", syncGroup.Name, err)
			}
		}
	}

	// Get all LDAP groups from the database
	var ldapGroupsInDb []model.UserGroup
	err = tx.
		WithContext(ctx).
		Find(&ldapGroupsInDb, "ldap_id IS NOT NULL").
		Select("ldap_id").
		Error
	if err != nil {
		return fmt.Errorf("failed to fetch groups from database: %w", err)
	}

	// Delete groups that no longer exist in LDAP
	for _, group := range ldapGroupsInDb {
		if _, exists := ldapGroupIDs[*group.LdapID]; exists {
			continue
		}

		err = tx.
			WithContext(ctx).
			Delete(&model.UserGroup{}, "ldap_id = ?", group.LdapID).
			Error
		if err != nil {
			return fmt.Errorf("failed to delete group '%s': %w", group.Name, err)
		}

		slog.Info("Deleted group", slog.String("group", group.Name))
	}

	return nil
}

//nolint:gocognit
func (s *LdapService) SyncUsers(ctx context.Context, tx *gorm.DB, client *ldap.Conn) error {
	dbConfig := s.appConfigService.GetDbConfig()

	searchAttrs := []string{
		"memberOf",
		"sn",
		"cn",
		dbConfig.LdapAttributeUserUniqueIdentifier.Value,
		dbConfig.LdapAttributeUserUsername.Value,
		dbConfig.LdapAttributeUserEmail.Value,
		dbConfig.LdapAttributeUserFirstName.Value,
		dbConfig.LdapAttributeUserLastName.Value,
		dbConfig.LdapAttributeUserProfilePicture.Value,
	}

	// Filters must start and finish with ()!
	searchReq := ldap.NewSearchRequest(
		dbConfig.LdapBase.Value,
		ldap.ScopeWholeSubtree,
		0, 0, 0, false,
		dbConfig.LdapUserSearchFilter.Value,
		searchAttrs,
		[]ldap.Control{},
	)

	result, err := client.Search(searchReq)
	if err != nil {
		return fmt.Errorf("failed to query LDAP: %w", err)
	}

	// Create a mapping for users that exist
	ldapUserIDs := make(map[string]struct{}, len(result.Entries))

	for _, value := range result.Entries {
		ldapId := convertLdapIdToString(value.GetAttributeValue(dbConfig.LdapAttributeUserUniqueIdentifier.Value))

		// Skip users without a valid LDAP ID
		if ldapId == "" {
			slog.Warn("Skipping LDAP user without a valid unique identifier", slog.String("attribute", dbConfig.LdapAttributeUserUniqueIdentifier.Value))
			continue
		}

		ldapUserIDs[ldapId] = struct{}{}

		// Get the user from the database
		var databaseUser model.User
		err = tx.
			WithContext(ctx).
			Where("ldap_id = ?", ldapId).
			First(&databaseUser).
			Error

		// If a user is found (even if disabled), enable them since they're now back in LDAP
		if databaseUser.ID != "" && databaseUser.Disabled {
			err = tx.
				WithContext(ctx).
				Model(&model.User{}).
				Where("id = ?", databaseUser.ID).
				Update("disabled", false).
				Error

			if err != nil {
				return fmt.Errorf("failed to enable user %s: %w", databaseUser.Username, err)
			}
		}

		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			// This could error with ErrRecordNotFound and we want to ignore that here
			return fmt.Errorf("failed to query for LDAP user ID '%s': %w", ldapId, err)
		}

		// Check if user is admin by checking if they are in the admin group
		isAdmin := false
		for _, group := range value.GetAttributeValues("memberOf") {
			if getDNProperty(dbConfig.LdapAttributeGroupName.Value, group) == dbConfig.LdapAttributeAdminGroup.Value {
				isAdmin = true
				break
			}
		}

		newUser := dto.UserCreateDto{
			Username:  value.GetAttributeValue(dbConfig.LdapAttributeUserUsername.Value),
			Email:     value.GetAttributeValue(dbConfig.LdapAttributeUserEmail.Value),
			FirstName: value.GetAttributeValue(dbConfig.LdapAttributeUserFirstName.Value),
			LastName:  value.GetAttributeValue(dbConfig.LdapAttributeUserLastName.Value),
			IsAdmin:   isAdmin,
			LdapID:    ldapId,
		}
		dto.Normalize(newUser)

		if databaseUser.ID == "" {
			_, err = s.userService.createUserInternal(ctx, newUser, true, tx)
			if errors.Is(err, &common.AlreadyInUseError{}) {
				slog.Warn("Skipping creating LDAP user", slog.String("username", newUser.Username), slog.Any("error", err))
				continue
			} else if err != nil {
				return fmt.Errorf("error creating user '%s': %w", newUser.Username, err)
			}
		} else {
			_, err = s.userService.updateUserInternal(ctx, databaseUser.ID, newUser, false, true, tx)
			if errors.Is(err, &common.AlreadyInUseError{}) {
				slog.Warn("Skipping updating LDAP user", slog.String("username", newUser.Username), slog.Any("error", err))
				continue
			} else if err != nil {
				return fmt.Errorf("error updating user '%s': %w", newUser.Username, err)
			}
		}

		// Save profile picture
		pictureString := value.GetAttributeValue(dbConfig.LdapAttributeUserProfilePicture.Value)
		if pictureString != "" {
			err = s.saveProfilePicture(ctx, databaseUser.ID, pictureString)
			if err != nil {
				// This is not a fatal error
				slog.Warn("Error saving profile picture for user", slog.String("username", newUser.Username), slog.Any("error", err))
			}
		}
	}

	// Get all LDAP users from the database
	var ldapUsersInDb []model.User
	err = tx.
		WithContext(ctx).
		Find(&ldapUsersInDb, "ldap_id IS NOT NULL").
		Select("id, username, ldap_id, disabled").
		Error
	if err != nil {
		return fmt.Errorf("failed to fetch users from database: %w", err)
	}

	// Mark users as disabled or delete users that no longer exist in LDAP
	for _, user := range ldapUsersInDb {
		// Skip if the user ID exists in the fetched LDAP results
		if _, exists := ldapUserIDs[*user.LdapID]; exists {
			continue
		}

		if dbConfig.LdapSoftDeleteUsers.IsTrue() {
			err = s.userService.disableUserInternal(ctx, user.ID, tx)
			if err != nil {
				return fmt.Errorf("failed to disable user %s: %w", user.Username, err)
			}

			slog.Info("Disabled user", slog.String("username", user.Username))
		} else {
			err = s.userService.deleteUserInternal(ctx, user.ID, true, tx)
			target := &common.LdapUserUpdateError{}
			if errors.As(err, &target) {
				return fmt.Errorf("failed to delete user %s: LDAP user must be disabled before deletion", user.Username)
			} else if err != nil {
				return fmt.Errorf("failed to delete user %s: %w", user.Username, err)
			}

			slog.Info("Deleted user", slog.String("username", user.Username))
		}
	}

	return nil
}

func (s *LdapService) saveProfilePicture(parentCtx context.Context, userId string, pictureString string) error {
	var reader io.Reader

	_, err := url.ParseRequestURI(pictureString)
	if err == nil {
		ctx, cancel := context.WithTimeout(parentCtx, 15*time.Second)
		defer cancel()

		var req *http.Request
		req, err = http.NewRequestWithContext(ctx, http.MethodGet, pictureString, nil)
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}

		var res *http.Response
		res, err = s.httpClient.Do(req)
		if err != nil {
			return fmt.Errorf("failed to download profile picture: %w", err)
		}
		defer res.Body.Close()

		reader = res.Body
	} else if decodedPhoto, err := base64.StdEncoding.DecodeString(pictureString); err == nil {
		// If the photo is a base64 encoded string, decode it
		reader = bytes.NewReader(decodedPhoto)
	} else {
		// If the photo is a string, we assume that it's a binary string
		reader = bytes.NewReader([]byte(pictureString))
	}

	// Update the profile picture
	err = s.userService.UpdateProfilePicture(userId, reader)
	if err != nil {
		return fmt.Errorf("failed to update profile picture: %w", err)
	}

	return nil
}

// getDNProperty returns the value of a property from a LDAP identifier
// See: https://learn.microsoft.com/en-us/previous-versions/windows/desktop/ldap/distinguished-names
func getDNProperty(property string, str string) string {
	// Example format is "CN=username,ou=people,dc=example,dc=com"
	// First we split at the comma
	property = strings.ToLower(property)
	l := len(property) + 1
	for _, v := range strings.Split(str, ",") {
		v = strings.TrimSpace(v)
		if len(v) > l && strings.ToLower(v)[0:l] == property+"=" {
			return v[l:]
		}
	}

	// CN not found, return an empty string
	return ""
}

// convertLdapIdToString converts LDAP IDs to valid UTF-8 strings.
// LDAP servers may return binary UUIDs (16 bytes) or other non-UTF-8 data.
func convertLdapIdToString(ldapId string) string {
	if utf8.ValidString(ldapId) {
		return norm.NFC.String(ldapId)
	}

	// Try to parse as binary UUID (16 bytes)
	if len(ldapId) == 16 {
		if parsedUUID, err := uuid.FromBytes([]byte(ldapId)); err == nil {
			return parsedUUID.String()
		}
	}

	// As a last resort, encode as base64 to make it UTF-8 safe
	return base64.StdEncoding.EncodeToString([]byte(ldapId))
}
