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
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-ldap/ldap/v3"
	"github.com/pocket-id/pocket-id/backend/internal/dto"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	"gorm.io/gorm"
)

type LdapService struct {
	db               *gorm.DB
	appConfigService *AppConfigService
	userService      *UserService
	groupService     *UserGroupService
}

func NewLdapService(db *gorm.DB, appConfigService *AppConfigService, userService *UserService, groupService *UserGroupService) *LdapService {
	return &LdapService{db: db, appConfigService: appConfigService, userService: userService, groupService: groupService}
}

func (s *LdapService) createClient() (*ldap.Conn, error) {
	dbConfig := s.appConfigService.GetDbConfig()

	if !dbConfig.LdapEnabled.IsTrue() {
		return nil, fmt.Errorf("LDAP is not enabled")
	}

	// Setup LDAP connection
	ldapURL := dbConfig.LdapUrl.Value
	skipTLSVerify := dbConfig.LdapSkipCertVerify.IsTrue()
	client, err := ldap.DialURL(ldapURL, ldap.DialWithTLSConfig(&tls.Config{
		InsecureSkipVerify: skipTLSVerify, //nolint:gosec
	}))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to LDAP: %w", err)
	}

	// Bind as service account
	bindDn := dbConfig.LdapBindDn.Value
	bindPassword := dbConfig.LdapBindPassword.Value
	err = client.Bind(bindDn, bindPassword)
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

	err := s.SyncUsers(ctx, tx)
	if err != nil {
		return fmt.Errorf("failed to sync users: %w", err)
	}

	err = s.SyncGroups(ctx, tx)
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
func (s *LdapService) SyncGroups(ctx context.Context, tx *gorm.DB) error {
	dbConfig := s.appConfigService.GetDbConfig()

	// Setup LDAP connection
	client, err := s.createClient()
	if err != nil {
		return fmt.Errorf("failed to create LDAP client: %w", err)
	}
	defer client.Close()

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
	ldapGroupIDs := make(map[string]bool)

	for _, value := range result.Entries {
		var membersUserId []string

		ldapId := value.GetAttributeValue(dbConfig.LdapAttributeGroupUniqueIdentifier.Value)

		// Skip groups without a valid LDAP ID
		if ldapId == "" {
			log.Printf("Skipping LDAP group without a valid unique identifier (attribute: %s)", dbConfig.LdapAttributeGroupUniqueIdentifier.Value)
			continue
		}

		ldapGroupIDs[ldapId] = true

		// Try to find the group in the database
		var databaseGroup model.UserGroup
		tx.WithContext(ctx).Where("ldap_id = ?", ldapId).First(&databaseGroup)

		// Get group members and add to the correct Group
		groupMembers := value.GetAttributeValues(dbConfig.LdapAttributeGroupMember.Value)
		for _, member := range groupMembers {
			// Normal output of this would be CN=username,ou=people,dc=example,dc=com
			// Splitting at the "=" and "," then just grabbing the username for that string
			singleMember := strings.Split(strings.Split(member, "=")[1], ",")[0]

			var databaseUser model.User
			err := tx.WithContext(ctx).Where("username = ? AND ldap_id IS NOT NULL", singleMember).First(&databaseUser).Error
			if err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					// The user collides with a non-LDAP user, so we skip it
					continue
				} else {
					return err
				}

			}

			membersUserId = append(membersUserId, databaseUser.ID)
		}

		syncGroup := dto.UserGroupCreateDto{
			Name:         value.GetAttributeValue(dbConfig.LdapAttributeGroupName.Value),
			FriendlyName: value.GetAttributeValue(dbConfig.LdapAttributeGroupName.Value),
			LdapID:       value.GetAttributeValue(dbConfig.LdapAttributeGroupUniqueIdentifier.Value),
		}

		if databaseGroup.ID == "" {
			newGroup, err := s.groupService.createInternal(ctx, syncGroup, tx)
			if err != nil {
				log.Printf("Error syncing group %s: %v", syncGroup.Name, err)
				continue
			}

			_, err = s.groupService.updateUsersInternal(ctx, newGroup.ID, membersUserId, tx)
			if err != nil {
				log.Printf("Error syncing group %s: %v", syncGroup.Name, err)
				continue
			}
		} else {
			_, err = s.groupService.updateInternal(ctx, databaseGroup.ID, syncGroup, true, tx)
			if err != nil {
				log.Printf("Error syncing group %s: %v", syncGroup.Name, err)
				continue
			}

			_, err = s.groupService.updateUsersInternal(ctx, databaseGroup.ID, membersUserId, tx)
			if err != nil {
				log.Printf("Error syncing group %s: %v", syncGroup.Name, err)
				continue
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
		log.Printf("Failed to fetch groups from database: %v", err)
	}

	// Delete groups that no longer exist in LDAP
	for _, group := range ldapGroupsInDb {
		if _, exists := ldapGroupIDs[*group.LdapID]; !exists {
			err = tx.
				WithContext(ctx).
				Delete(&model.UserGroup{}, "ldap_id = ?", group.LdapID).
				Error
			if err != nil {
				log.Printf("Failed to delete group %s with: %v", group.Name, err)
			} else {
				log.Printf("Deleted group %s", group.Name)
			}
		}
	}

	return nil
}

//nolint:gocognit
func (s *LdapService) SyncUsers(ctx context.Context, tx *gorm.DB) error {
	dbConfig := s.appConfigService.GetDbConfig()

	// Setup LDAP connection
	client, err := s.createClient()
	if err != nil {
		return fmt.Errorf("failed to create LDAP client: %w", err)
	}
	defer client.Close()

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
		fmt.Println(fmt.Errorf("failed to query LDAP: %w", err))
	}

	// Create a mapping for users that exist
	ldapUserIDs := make(map[string]bool)

	for _, value := range result.Entries {
		ldapId := value.GetAttributeValue(dbConfig.LdapAttributeUserUniqueIdentifier.Value)

		// Skip users without a valid LDAP ID
		if ldapId == "" {
			log.Printf("Skipping LDAP user without a valid unique identifier (attribute: %s)", dbConfig.LdapAttributeUserUniqueIdentifier.Value)
			continue
		}

		ldapUserIDs[ldapId] = true

		// Get the user from the database
		var databaseUser model.User
		tx.WithContext(ctx).Where("ldap_id = ?", ldapId).First(&databaseUser)

		// Check if user is admin by checking if they are in the admin group
		isAdmin := false
		for _, group := range value.GetAttributeValues("memberOf") {
			if strings.Contains(group, dbConfig.LdapAttributeAdminGroup.Value) {
				isAdmin = true
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

		if databaseUser.ID == "" {
			_, err = s.userService.createUserInternal(ctx, newUser, tx)
			if err != nil {
				log.Printf("Error syncing user %s: %v", newUser.Username, err)
			}
		} else {
			_, err = s.userService.updateUserInternal(ctx, databaseUser.ID, newUser, false, true, tx)
			if err != nil {
				log.Printf("Error syncing user %s: %v", newUser.Username, err)
			}
		}

		// Save profile picture
		if pictureString := value.GetAttributeValue(dbConfig.LdapAttributeUserProfilePicture.Value); pictureString != "" {
			if err := s.saveProfilePicture(ctx, databaseUser.ID, pictureString); err != nil {
				log.Printf("Error saving profile picture for user %s: %v", newUser.Username, err)
			}
		}
	}

	// Get all LDAP users from the database
	var ldapUsersInDb []model.User
	err = tx.
		WithContext(ctx).
		Find(&ldapUsersInDb, "ldap_id IS NOT NULL").
		Select("ldap_id").
		Error
	if err != nil {
		log.Printf("Failed to fetch users from database: %v", err)
	}

	// Delete users that no longer exist in LDAP
	for _, user := range ldapUsersInDb {
		if _, exists := ldapUserIDs[*user.LdapID]; !exists {
			if err := s.userService.deleteUserInternal(ctx, user.ID, true, tx); err != nil {
				log.Printf("Failed to delete user %s with: %v", user.Username, err)
			} else {
				log.Printf("Deleted user %s", user.Username)
			}
		}
	}

	return nil
}

func (s *LdapService) saveProfilePicture(parentCtx context.Context, userId string, pictureString string) error {
	var reader io.Reader

	_, err := url.ParseRequestURI(pictureString)
	if err == nil {
		ctx, cancel := context.WithTimeout(parentCtx, 5*time.Second)
		defer cancel()

		var req *http.Request
		req, err = http.NewRequestWithContext(ctx, http.MethodGet, pictureString, nil)
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}

		var res *http.Response
		res, err = http.DefaultClient.Do(req)
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
	if err := s.userService.UpdateProfilePicture(userId, reader); err != nil {
		return fmt.Errorf("failed to update profile picture: %w", err)
	}

	return nil
}
