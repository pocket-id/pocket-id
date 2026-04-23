package service

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/go-ldap/ldap/v3"
	"github.com/google/uuid"
	"github.com/pocket-id/pocket-id/backend/internal/storage"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
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
	fileStorage      storage.FileStorage
	clientFactory    func() (ldapClient, error)
}

type savePicture struct {
	userID   string
	username string
	picture  string
}

type ldapDesiredUser struct {
	ldapID  string
	input   dto.UserCreateDto
	picture string
}

type ldapDesiredGroup struct {
	ldapID          string
	input           dto.UserGroupCreateDto
	memberUsernames []string
}

type ldapDesiredState struct {
	users    []ldapDesiredUser
	userIDs  map[string]struct{}
	groups   []ldapDesiredGroup
	groupIDs map[string]struct{}
}

type ldapClient interface {
	Search(searchRequest *ldap.SearchRequest) (*ldap.SearchResult, error)
	Bind(username, password string) error
	Close() error
}

func NewLdapService(db *gorm.DB, httpClient *http.Client, appConfigService *AppConfigService, userService *UserService, groupService *UserGroupService, fileStorage storage.FileStorage) *LdapService {
	service := &LdapService{
		db:               db,
		httpClient:       httpClient,
		appConfigService: appConfigService,
		userService:      userService,
		groupService:     groupService,
		fileStorage:      fileStorage,
	}

	service.clientFactory = service.createClient
	return service
}

func (s *LdapService) createClient() (ldapClient, error) {
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
	// Setup LDAP connection
	client, err := s.clientFactory()
	if err != nil {
		return fmt.Errorf("failed to create LDAP client: %w", err)
	}
	defer client.Close()

	// First, we fetch all users and group from LDAP, which is our "desired state"
	desiredState, err := s.fetchDesiredState(ctx, client)
	if err != nil {
		return fmt.Errorf("failed to fetch LDAP state: %w", err)
	}

	// Start a transaction
	tx := s.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return fmt.Errorf("failed to begin database transaction: %w", tx.Error)
	}
	defer tx.Rollback()

	// Reconcile users
	savePictures, deleteFiles, err := s.reconcileUsers(ctx, tx, desiredState.users, desiredState.userIDs)
	if err != nil {
		return fmt.Errorf("failed to sync users: %w", err)
	}

	// Reconcile groups
	err = s.reconcileGroups(ctx, tx, desiredState.groups, desiredState.groupIDs)
	if err != nil {
		return fmt.Errorf("failed to sync groups: %w", err)
	}

	// Commit the changes
	err = tx.Commit().Error
	if err != nil {
		return fmt.Errorf("failed to commit changes to database: %w", err)
	}

	// Now that we've committed the transaction, we can perform operations on the storage layer
	// First, save all new pictures
	for _, sp := range savePictures {
		err = s.saveProfilePicture(ctx, sp.userID, sp.picture)
		if err != nil {
			// This is not a fatal error
			slog.Warn("Error saving profile picture for LDAP user", slog.String("username", sp.username), slog.Any("error", err))
		}
	}

	// Delete all old files
	for _, path := range deleteFiles {
		err = s.fileStorage.Delete(ctx, path)
		if err != nil {
			// This is not a fatal error
			slog.Error("Failed to delete file after LDAP sync", slog.String("path", path), slog.Any("error", err))
		}
	}

	return nil
}

func (s *LdapService) fetchDesiredState(ctx context.Context, client ldapClient) (ldapDesiredState, error) {
	// Fetch users first so we can use their DNs when resolving group members
	users, userIDs, usernamesByDN, err := s.fetchUsersFromLDAP(ctx, client)
	if err != nil {
		return ldapDesiredState{}, err
	}

	// Then fetch groups to complete the desired LDAP state snapshot
	groups, groupIDs, err := s.fetchGroupsFromLDAP(ctx, client, usernamesByDN)
	if err != nil {
		return ldapDesiredState{}, err
	}

	// Apply user admin flags from the desired group membership snapshot.
	// This intentionally uses the configured group member attribute rather than
	// relying on a user-side reverse-membership attribute such as memberOf.
	s.applyAdminGroupMembership(users, groups)

	return ldapDesiredState{
		users:    users,
		userIDs:  userIDs,
		groups:   groups,
		groupIDs: groupIDs,
	}, nil
}

func (s *LdapService) applyAdminGroupMembership(desiredUsers []ldapDesiredUser, desiredGroups []ldapDesiredGroup) {
	dbConfig := s.appConfigService.GetDbConfig()
	if dbConfig.LdapAdminGroupName.Value == "" {
		return
	}

	adminUsernames := make(map[string]struct{})
	for _, group := range desiredGroups {
		if group.input.Name != dbConfig.LdapAdminGroupName.Value {
			continue
		}

		for _, username := range group.memberUsernames {
			adminUsernames[username] = struct{}{}
		}
	}

	for i := range desiredUsers {
		_, isAdmin := adminUsernames[desiredUsers[i].input.Username]
		desiredUsers[i].input.IsAdmin = desiredUsers[i].input.IsAdmin || isAdmin
	}
}

func (s *LdapService) fetchGroupsFromLDAP(ctx context.Context, client ldapClient, usernamesByDN map[string]string) (desiredGroups []ldapDesiredGroup, ldapGroupIDs map[string]struct{}, err error) {
	dbConfig := s.appConfigService.GetDbConfig()

	// Query LDAP for all groups we want to manage
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
		return nil, nil, fmt.Errorf("failed to query LDAP groups: %w", err)
	}

	// Build the in-memory desired state for groups
	ldapGroupIDs = make(map[string]struct{}, len(result.Entries))
	desiredGroups = make([]ldapDesiredGroup, 0, len(result.Entries))

	for _, value := range result.Entries {
		ldapID := convertLdapIdToString(value.GetAttributeValue(dbConfig.LdapAttributeGroupUniqueIdentifier.Value))

		// Skip groups without a valid LDAP ID
		if ldapID == "" {
			slog.Warn("Skipping LDAP group without a valid unique identifier", slog.String("attribute", dbConfig.LdapAttributeGroupUniqueIdentifier.Value))
			continue
		}

		ldapGroupIDs[ldapID] = struct{}{}

		// Get group members and add to the correct Group
		groupMembers := value.GetAttributeValues(dbConfig.LdapAttributeGroupMember.Value)
		memberUsernames := make([]string, 0, len(groupMembers))
		for _, member := range groupMembers {
			username := s.resolveGroupMemberUsername(ctx, client, member, usernamesByDN)
			if username == "" {
				continue
			}

			memberUsernames = append(memberUsernames, username)
		}

		syncGroup := dto.UserGroupCreateDto{
			Name:         value.GetAttributeValue(dbConfig.LdapAttributeGroupName.Value),
			FriendlyName: value.GetAttributeValue(dbConfig.LdapAttributeGroupName.Value),
			LdapID:       ldapID,
		}
		dto.Normalize(&syncGroup)

		err = syncGroup.Validate()
		if err != nil {
			slog.WarnContext(ctx, "LDAP user group object is not valid", slog.Any("error", err))
			continue
		}

		desiredGroups = append(desiredGroups, ldapDesiredGroup{
			ldapID:          ldapID,
			input:           syncGroup,
			memberUsernames: memberUsernames,
		})
	}

	return desiredGroups, ldapGroupIDs, nil
}

func (s *LdapService) fetchUsersFromLDAP(ctx context.Context, client ldapClient) (desiredUsers []ldapDesiredUser, ldapUserIDs map[string]struct{}, usernamesByDN map[string]string, err error) {
	dbConfig := s.appConfigService.GetDbConfig()

	// Query LDAP for all users we want to manage
	searchAttrs := []string{
		"sn",
		"cn",
		dbConfig.LdapAttributeUserUniqueIdentifier.Value,
		dbConfig.LdapAttributeUserUsername.Value,
		dbConfig.LdapAttributeUserEmail.Value,
		dbConfig.LdapAttributeUserFirstName.Value,
		dbConfig.LdapAttributeUserLastName.Value,
		dbConfig.LdapAttributeUserProfilePicture.Value,
		dbConfig.LdapAttributeUserDisplayName.Value,
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
		return nil, nil, nil, fmt.Errorf("failed to query LDAP users: %w", err)
	}

	// Build the in-memory desired state for users and a DN lookup for group membership resolution
	ldapUserIDs = make(map[string]struct{}, len(result.Entries))
	usernamesByDN = make(map[string]string, len(result.Entries))
	desiredUsers = make([]ldapDesiredUser, 0, len(result.Entries))

	for _, value := range result.Entries {
		username := norm.NFC.String(value.GetAttributeValue(dbConfig.LdapAttributeUserUsername.Value))
		if normalizedDN := normalizeLDAPDN(value.DN); normalizedDN != "" && username != "" {
			usernamesByDN[normalizedDN] = username
		}

		ldapID := convertLdapIdToString(value.GetAttributeValue(dbConfig.LdapAttributeUserUniqueIdentifier.Value))

		// Skip users without a valid LDAP ID
		if ldapID == "" {
			slog.Warn("Skipping LDAP user without a valid unique identifier", slog.String("attribute", dbConfig.LdapAttributeUserUniqueIdentifier.Value))
			continue
		}

		ldapUserIDs[ldapID] = struct{}{}

		newUser := dto.UserCreateDto{
			Username:      value.GetAttributeValue(dbConfig.LdapAttributeUserUsername.Value),
			Email:         utils.PtrOrNil(value.GetAttributeValue(dbConfig.LdapAttributeUserEmail.Value)),
			EmailVerified: true,
			FirstName:     value.GetAttributeValue(dbConfig.LdapAttributeUserFirstName.Value),
			LastName:      value.GetAttributeValue(dbConfig.LdapAttributeUserLastName.Value),
			DisplayName:   value.GetAttributeValue(dbConfig.LdapAttributeUserDisplayName.Value),
			// Admin status is computed after groups are loaded so it can use the
			// configured group member attribute instead of a hard-coded memberOf.
			IsAdmin: false,
			LdapID:  ldapID,
		}

		if newUser.DisplayName == "" {
			newUser.DisplayName = strings.TrimSpace(newUser.FirstName + " " + newUser.LastName)
		}

		dto.Normalize(&newUser)

		err = newUser.Validate()
		if err != nil {
			slog.WarnContext(ctx, "LDAP user object is not valid", slog.Any("error", err))
			continue
		}

		desiredUsers = append(desiredUsers, ldapDesiredUser{
			ldapID:  ldapID,
			input:   newUser,
			picture: value.GetAttributeValue(dbConfig.LdapAttributeUserProfilePicture.Value),
		})
	}

	return desiredUsers, ldapUserIDs, usernamesByDN, nil
}

func (s *LdapService) resolveGroupMemberUsername(ctx context.Context, client ldapClient, member string, usernamesByDN map[string]string) string {
	dbConfig := s.appConfigService.GetDbConfig()

	// First try the DN cache we built while loading users
	username, exists := usernamesByDN[normalizeLDAPDN(member)]
	if exists && username != "" {
		return username
	}

	// Then try to extract the username directly from the DN
	username = getDNProperty(dbConfig.LdapAttributeUserUsername.Value, member)
	if username != "" {
		return norm.NFC.String(username)
	}

	// posixGroup (and similar) stores bare usernames in memberUid, not DNs. Treat any value
	// that is not a valid DN as the username directly — see https://github.com/pocket-id/pocket-id/issues/1408
	if _, err := ldap.ParseDN(member); err != nil {
		return norm.NFC.String(member)
	}

	// As a fallback, query LDAP for the referenced entry
	userSearchReq := ldap.NewSearchRequest(
		member,
		ldap.ScopeBaseObject,
		0, 0, 0, false,
		"(objectClass=*)",
		[]string{dbConfig.LdapAttributeUserUsername.Value},
		[]ldap.Control{},
	)

	userResult, err := client.Search(userSearchReq)
	if err != nil || len(userResult.Entries) == 0 {
		slog.WarnContext(ctx, "Could not resolve group member DN", slog.String("member", member), slog.Any("error", err))
		return ""
	}

	username = userResult.Entries[0].GetAttributeValue(dbConfig.LdapAttributeUserUsername.Value)
	if username == "" {
		slog.WarnContext(ctx, "Could not extract username from group member DN", slog.String("member", member))
		return ""
	}

	return norm.NFC.String(username)
}

func (s *LdapService) reconcileGroups(ctx context.Context, tx *gorm.DB, desiredGroups []ldapDesiredGroup, ldapGroupIDs map[string]struct{}) error {
	// Load the current LDAP-managed state from the database
	ldapGroupsInDB, ldapGroupsByID, err := s.loadLDAPGroupsInDB(ctx, tx)
	if err != nil {
		return fmt.Errorf("failed to fetch groups from database: %w", err)
	}

	_, _, ldapUsersByUsername, err := s.loadLDAPUsersInDB(ctx, tx)
	if err != nil {
		return fmt.Errorf("failed to fetch users from database: %w", err)
	}

	// Apply creates and updates to match the desired LDAP group state
	for _, desiredGroup := range desiredGroups {
		memberUserIDs := make([]string, 0, len(desiredGroup.memberUsernames))
		for _, username := range desiredGroup.memberUsernames {
			databaseUser, exists := ldapUsersByUsername[username]
			if !exists {
				// The user collides with a non-LDAP user or was skipped during user sync, so we ignore it
				continue
			}

			memberUserIDs = append(memberUserIDs, databaseUser.ID)
		}

		databaseGroup := ldapGroupsByID[desiredGroup.ldapID]
		if databaseGroup.ID == "" {
			newGroup, err := s.groupService.createInternal(ctx, desiredGroup.input, tx)
			if err != nil {
				return fmt.Errorf("failed to create group '%s': %w", desiredGroup.input.Name, err)
			}
			ldapGroupsByID[desiredGroup.ldapID] = newGroup

			_, err = s.groupService.updateUsersInternal(ctx, newGroup.ID, memberUserIDs, tx)
			if err != nil {
				return fmt.Errorf("failed to sync users for group '%s': %w", desiredGroup.input.Name, err)
			}
			continue
		}

		_, err = s.groupService.updateInternal(ctx, databaseGroup.ID, desiredGroup.input, true, tx)
		if err != nil {
			return fmt.Errorf("failed to update group '%s': %w", desiredGroup.input.Name, err)
		}

		_, err = s.groupService.updateUsersInternal(ctx, databaseGroup.ID, memberUserIDs, tx)
		if err != nil {
			return fmt.Errorf("failed to sync users for group '%s': %w", desiredGroup.input.Name, err)
		}
	}

	// Delete groups that are no longer present in LDAP
	for _, group := range ldapGroupsInDB {
		if group.LdapID == nil {
			continue
		}

		if _, exists := ldapGroupIDs[*group.LdapID]; exists {
			continue
		}

		err = tx.
			WithContext(ctx).
			Delete(&model.UserGroup{}, "ldap_id = ?", *group.LdapID).
			Error
		if err != nil {
			return fmt.Errorf("failed to delete group '%s': %w", group.Name, err)
		}

		slog.Info("Deleted group", slog.String("group", group.Name))
	}

	return nil
}

//nolint:gocognit
func (s *LdapService) reconcileUsers(ctx context.Context, tx *gorm.DB, desiredUsers []ldapDesiredUser, ldapUserIDs map[string]struct{}) (savePictures []savePicture, deleteFiles []string, err error) {
	dbConfig := s.appConfigService.GetDbConfig()

	// Load the current LDAP-managed state from the database
	ldapUsersInDB, ldapUsersByID, _, err := s.loadLDAPUsersInDB(ctx, tx)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to fetch users from database: %w", err)
	}

	// Apply creates and updates to match the desired LDAP user state
	savePictures = make([]savePicture, 0, len(desiredUsers))

	for _, desiredUser := range desiredUsers {
		databaseUser := ldapUsersByID[desiredUser.ldapID]

		// If a user is found (even if disabled), enable them since they're now back in LDAP.
		if databaseUser.ID != "" && databaseUser.Disabled {
			err = tx.
				WithContext(ctx).
				Model(&model.User{}).
				Where("id = ?", databaseUser.ID).
				Update("disabled", false).
				Error
			if err != nil {
				return nil, nil, fmt.Errorf("failed to enable user %s: %w", databaseUser.Username, err)
			}

			databaseUser.Disabled = false
			ldapUsersByID[desiredUser.ldapID] = databaseUser
		}

		userID := databaseUser.ID
		if databaseUser.ID == "" {
			createdUser, err := s.userService.createUserInternal(ctx, desiredUser.input, true, tx)
			if errors.Is(err, &common.AlreadyInUseError{}) {
				slog.Warn("Skipping creating LDAP user", slog.String("username", desiredUser.input.Username), slog.Any("error", err))
				continue
			} else if err != nil {
				return nil, nil, fmt.Errorf("error creating user '%s': %w", desiredUser.input.Username, err)
			}

			userID = createdUser.ID
			ldapUsersByID[desiredUser.ldapID] = createdUser
		} else {
			_, err = s.userService.updateUserInternal(ctx, databaseUser.ID, desiredUser.input, false, true, tx)
			if errors.Is(err, &common.AlreadyInUseError{}) {
				slog.Warn("Skipping updating LDAP user", slog.String("username", desiredUser.input.Username), slog.Any("error", err))
				continue
			} else if err != nil {
				return nil, nil, fmt.Errorf("error updating user '%s': %w", desiredUser.input.Username, err)
			}
		}

		if desiredUser.picture != "" {
			savePictures = append(savePictures, savePicture{
				userID:   userID,
				username: desiredUser.input.Username,
				picture:  desiredUser.picture,
			})
		}
	}

	// Disable or delete users that are no longer present in LDAP
	deleteFiles = make([]string, 0, len(ldapUsersInDB))
	for _, user := range ldapUsersInDB {
		if user.LdapID == nil {
			continue
		}

		if _, exists := ldapUserIDs[*user.LdapID]; exists {
			continue
		}

		if dbConfig.LdapSoftDeleteUsers.IsTrue() {
			err = s.userService.disableUserInternal(ctx, tx, user.ID)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to disable user %s: %w", user.Username, err)
			}

			slog.Info("Disabled user", slog.String("username", user.Username))
			continue
		}

		err = s.userService.deleteUserInternal(ctx, tx, user.ID, true)
		if err != nil {
			target := &common.LdapUserUpdateError{}
			if errors.As(err, &target) {
				return nil, nil, fmt.Errorf("failed to delete user %s: LDAP user must be disabled before deletion", user.Username)
			}
			return nil, nil, fmt.Errorf("failed to delete user %s: %w", user.Username, err)
		}

		slog.Info("Deleted user", slog.String("username", user.Username))
		deleteFiles = append(deleteFiles, path.Join("profile-pictures", user.ID+".png"))
	}

	return savePictures, deleteFiles, nil
}

func (s *LdapService) loadLDAPUsersInDB(ctx context.Context, tx *gorm.DB) (users []model.User, byLdapID map[string]model.User, byUsername map[string]model.User, err error) {
	// Load all LDAP-managed users and index them by LDAP ID and by username
	err = tx.
		WithContext(ctx).
		Select("id, username, ldap_id, disabled").
		Where("ldap_id IS NOT NULL").
		Find(&users).
		Error
	if err != nil {
		return nil, nil, nil, err
	}

	byLdapID = make(map[string]model.User, len(users))
	byUsername = make(map[string]model.User, len(users))
	for _, user := range users {
		byLdapID[*user.LdapID] = user
		byUsername[user.Username] = user
	}

	return users, byLdapID, byUsername, nil
}

func (s *LdapService) loadLDAPGroupsInDB(ctx context.Context, tx *gorm.DB) ([]model.UserGroup, map[string]model.UserGroup, error) {
	var groups []model.UserGroup

	// Load all LDAP-managed groups and index them by LDAP ID
	err := tx.
		WithContext(ctx).
		Select("id, name, ldap_id").
		Where("ldap_id IS NOT NULL").
		Find(&groups).
		Error
	if err != nil {
		return nil, nil, err
	}

	groupsByID := make(map[string]model.UserGroup, len(groups))
	for _, group := range groups {
		groupsByID[*group.LdapID] = group
	}

	return groups, groupsByID, nil
}

func (s *LdapService) saveProfilePicture(parentCtx context.Context, userId string, pictureString string) error {
	var reader io.ReadSeeker

	// Accept either a URL, a base64-encoded payload, or raw binary data
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

		data, err := io.ReadAll(res.Body)
		if err != nil {
			return fmt.Errorf("failed to read profile picture: %w", err)
		}

		reader = bytes.NewReader(data)
	} else if decodedPhoto, err := base64.StdEncoding.DecodeString(pictureString); err == nil {
		// If the photo is a base64 encoded string, decode it
		reader = bytes.NewReader(decodedPhoto)
	} else {
		// If the photo is a string, we assume that it's a binary string
		reader = bytes.NewReader([]byte(pictureString))
	}

	// Update the profile picture
	err = s.userService.UpdateProfilePicture(parentCtx, userId, reader)
	if err != nil {
		return fmt.Errorf("failed to update profile picture: %w", err)
	}

	return nil
}

// normalizeLDAPDN returns a canonical lowercase form of a DN for use as a map key.
// Different LDAP servers may format the same DN with varying attribute type casing (e.g. "CN=" vs "cn=") or extra whitespace (e.g. "dc=example, dc=com").
// Without normalization, cache lookups in usernamesByDN would miss when a member attribute value uses a different format than the DN returned in the search entry
//
// ldap.ParseDN is used instead of simple lowercasing because it correctly handles multi-valued RDNs (joined with "+") and strips inter-component whitespace.
// If parsing fails for any reason, we fall back to a simple lowercase+trim.
func normalizeLDAPDN(dn string) string {
	parsed, err := ldap.ParseDN(dn)
	if err != nil {
		return strings.ToLower(strings.TrimSpace(dn))
	}

	// Reconstruct the DN in a canonical form: lowercase type=lowercase value, with RDN components separated by "," and multi-value attributes by "+"
	parts := make([]string, 0, len(parsed.RDNs))
	for _, rdn := range parsed.RDNs {
		attrs := make([]string, 0, len(rdn.Attributes))
		for _, attr := range rdn.Attributes {
			attrs = append(attrs, strings.ToLower(attr.Type)+"="+strings.ToLower(attr.Value))
		}
		parts = append(parts, strings.Join(attrs, "+"))
	}

	return strings.Join(parts, ",")
}

// getDNProperty returns the value of a property from a LDAP identifier
// See: https://learn.microsoft.com/en-us/previous-versions/windows/desktop/ldap/distinguished-names
func getDNProperty(property string, str string) string {
	// Example format is "CN=username,ou=people,dc=example,dc=com"
	// First we split at the comma
	property = strings.ToLower(property)
	l := len(property) + 1
	for v := range strings.SplitSeq(str, ",") {
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
