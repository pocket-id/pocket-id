package service

import (
	"net/http"
	"testing"

	"github.com/go-ldap/ldap/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/model"
	"github.com/pocket-id/pocket-id/backend/internal/storage"
	testutils "github.com/pocket-id/pocket-id/backend/internal/utils/testing"
)

type fakeLDAPClient struct {
	searchFn func(searchRequest *ldap.SearchRequest) (*ldap.SearchResult, error)
}

func (c *fakeLDAPClient) Search(searchRequest *ldap.SearchRequest) (*ldap.SearchResult, error) {
	if c.searchFn == nil {
		return nil, nil
	}

	return c.searchFn(searchRequest)
}

func (c *fakeLDAPClient) Bind(_, _ string) error {
	return nil
}

func (c *fakeLDAPClient) Close() error {
	return nil
}

func TestLdapServiceSyncAllReconcilesUsersAndGroups(t *testing.T) {
	service, db := newTestLdapService(t, newFakeLDAPClient(
		ldapSearchResult(
			ldapEntry("uid=alice,ou=people,dc=example,dc=com", map[string][]string{
				"entryUUID":   {"u-alice"},
				"uid":         {"alice"},
				"mail":        {"alice@example.com"},
				"givenName":   {"Alice"},
				"sn":          {"Jones"},
				"displayName": {""},
			}),
			ldapEntry("uid=bob,ou=people,dc=example,dc=com", map[string][]string{
				"entryUUID":   {"u-bob"},
				"uid":         {"bob"},
				"mail":        {"bob@example.com"},
				"givenName":   {"Bob"},
				"sn":          {"Brown"},
				"displayName": {""},
			}),
		),
		ldapSearchResult(
			ldapEntry("cn=admins,ou=groups,dc=example,dc=com", map[string][]string{
				"entryUUID": {"g-admins"},
				"cn":        {"admins"},
				"member":    {"uid=alice,ou=people,dc=example,dc=com"},
			}),
			ldapEntry("cn=team,ou=groups,dc=example,dc=com", map[string][]string{
				"entryUUID": {"g-team"},
				"cn":        {"team"},
				"member": {
					"UID=Alice, OU=People, DC=example, DC=com",
					"uid=bob, ou=people, dc=example, dc=com",
				},
			}),
		),
	))

	aliceLdapID := "u-alice"
	missingLdapID := "u-missing"
	teamLdapID := "g-team"
	oldGroupLdapID := "g-old"

	require.NoError(t, db.Create(&model.User{
		Username:      "alice-old",
		Email:         new("alice-old@example.com"),
		EmailVerified: true,
		FirstName:     "Old",
		LastName:      "Name",
		DisplayName:   "Old Name",
		LdapID:        &aliceLdapID,
		Disabled:      true,
	}).Error)

	require.NoError(t, db.Create(&model.User{
		Username:      "missing",
		Email:         new("missing@example.com"),
		EmailVerified: true,
		FirstName:     "Missing",
		LastName:      "User",
		DisplayName:   "Missing User",
		LdapID:        &missingLdapID,
	}).Error)

	require.NoError(t, db.Create(&model.UserGroup{
		Name:         "team-old",
		FriendlyName: "team-old",
		LdapID:       &teamLdapID,
	}).Error)

	require.NoError(t, db.Create(&model.UserGroup{
		Name:         "old-group",
		FriendlyName: "old-group",
		LdapID:       &oldGroupLdapID,
	}).Error)

	require.NoError(t, service.SyncAll(t.Context()))

	var alice model.User
	require.NoError(t, db.First(&alice, "ldap_id = ?", aliceLdapID).Error)
	assert.Equal(t, "alice", alice.Username)
	assert.Equal(t, new("alice@example.com"), alice.Email)
	assert.Equal(t, "Alice", alice.FirstName)
	assert.Equal(t, "Jones", alice.LastName)
	assert.Equal(t, "Alice Jones", alice.DisplayName)
	assert.True(t, alice.IsAdmin)
	assert.False(t, alice.Disabled)

	var bob model.User
	require.NoError(t, db.First(&bob, "ldap_id = ?", "u-bob").Error)
	assert.Equal(t, "bob", bob.Username)
	assert.Equal(t, "Bob Brown", bob.DisplayName)

	var missing model.User
	require.NoError(t, db.First(&missing, "ldap_id = ?", missingLdapID).Error)
	assert.True(t, missing.Disabled)

	var oldGroupCount int64
	require.NoError(t, db.Model(&model.UserGroup{}).Where("ldap_id = ?", oldGroupLdapID).Count(&oldGroupCount).Error)
	assert.Zero(t, oldGroupCount)

	var team model.UserGroup
	require.NoError(t, db.Preload("Users").First(&team, "ldap_id = ?", teamLdapID).Error)
	assert.Equal(t, "team", team.Name)
	assert.Equal(t, "team", team.FriendlyName)
	assert.ElementsMatch(t, []string{"alice", "bob"}, usernames(team.Users))
}

// Regression: posixGroup uses memberUid (bare uid values), not member DNs — issue #1408.
func TestLdapServiceSyncAllMapsPosixGroupMemberUid(t *testing.T) {
	appCfg := defaultTestLDAPAppConfig()
	appCfg.LdapUserGroupSearchFilter = model.AppConfigVariable{Value: "(objectClass=posixGroup)"}
	appCfg.LdapAttributeGroupMember = model.AppConfigVariable{Value: "memberUid"}

	service, db := newTestLdapServiceWithAppConfig(t, appCfg, newFakeLDAPClient(
		ldapSearchResult(
			ldapEntry("uid=alice,ou=users,dc=example,dc=com", map[string][]string{
				"entryUUID":   {"u-alice"},
				"uid":         {"alice"},
				"mail":        {"alice@example.com"},
				"givenName":   {"Alice"},
				"sn":          {"Jones"},
				"displayName": {""},
			}),
			ldapEntry("uid=bob,ou=users,dc=example,dc=com", map[string][]string{
				"entryUUID":   {"u-bob"},
				"uid":         {"bob"},
				"mail":        {"bob@example.com"},
				"givenName":   {"Bob"},
				"sn":          {"Brown"},
				"displayName": {""},
			}),
		),
		ldapSearchResult(
			ldapEntry("cn=users,ou=groups,dc=example,dc=com", map[string][]string{
				"entryUUID": {"g-users"},
				"cn":        {"users"},
				"memberUid": {"alice", "bob", "unknown"},
			}),
		),
	))

	require.NoError(t, service.SyncAll(t.Context()))

	var group model.UserGroup
	require.NoError(t, db.Preload("Users").First(&group, "ldap_id = ?", "g-users").Error)
	assert.Equal(t, "users", group.Name)
	assert.ElementsMatch(t, []string{"alice", "bob"}, usernames(group.Users))
}

func TestLdapServiceSyncAllHandlesDuplicateLDAPIDsInSingleRun(t *testing.T) {
	service, db := newTestLdapService(t, newFakeLDAPClient(
		ldapSearchResult(
			ldapEntry("uid=alice,ou=people,dc=example,dc=com", map[string][]string{
				"entryUUID":   {"u-dup"},
				"uid":         {"alice"},
				"mail":        {"alice@example.com"},
				"givenName":   {"Alice"},
				"sn":          {"Doe"},
				"displayName": {"Alice Doe"},
			}),
			ldapEntry("uid=alice,ou=people,dc=example,dc=com", map[string][]string{
				"entryUUID":   {"u-dup"},
				"uid":         {"alice"},
				"mail":        {"alice@example.com"},
				"givenName":   {"Alicia"},
				"sn":          {"Doe"},
				"displayName": {"Alicia Doe"},
			}),
		),
		ldapSearchResult(
			ldapEntry("cn=team,ou=groups,dc=example,dc=com", map[string][]string{
				"entryUUID": {"g-dup"},
				"cn":        {"team"},
				"member":    {"uid=alice,ou=people,dc=example,dc=com"},
			}),
			ldapEntry("cn=team,ou=groups,dc=example,dc=com", map[string][]string{
				"entryUUID": {"g-dup"},
				"cn":        {"team-renamed"},
				"member":    {"uid=alice,ou=people,dc=example,dc=com"},
			}),
		),
	))

	require.NoError(t, service.SyncAll(t.Context()))

	var users []model.User
	require.NoError(t, db.Find(&users, "ldap_id = ?", "u-dup").Error)
	require.Len(t, users, 1)
	assert.Equal(t, "alice", users[0].Username)
	assert.Equal(t, "Alicia", users[0].FirstName)
	assert.Equal(t, "Alicia Doe", users[0].DisplayName)

	var groups []model.UserGroup
	require.NoError(t, db.Preload("Users").Find(&groups, "ldap_id = ?", "g-dup").Error)
	require.Len(t, groups, 1)
	assert.Equal(t, "team-renamed", groups[0].Name)
	assert.Equal(t, "team-renamed", groups[0].FriendlyName)
	assert.ElementsMatch(t, []string{"alice"}, usernames(groups[0].Users))
}

func TestLdapServiceSyncAllSetsAdminFromGroupMembership(t *testing.T) {
	tests := []struct {
		name        string
		appConfig   *model.AppConfig
		groupEntry  *ldap.Entry
		groupName   string
		groupLookup string
	}{
		{
			name:      "memberOf missing on user",
			appConfig: defaultTestLDAPAppConfig(),
			groupEntry: ldapEntry("cn=admins,ou=groups,dc=example,dc=com", map[string][]string{
				"entryUUID": {"g-admins"},
				"cn":        {"admins"},
				"member":    {"uid=testadmin,ou=people,dc=example,dc=com"},
			}),
			groupName:   "admins",
			groupLookup: "g-admins",
		},
		{
			name: "configured group name attribute differs from DN RDN",
			appConfig: func() *model.AppConfig {
				cfg := defaultTestLDAPAppConfig()
				cfg.LdapAttributeGroupName = model.AppConfigVariable{Value: "displayName"}
				cfg.LdapAdminGroupName = model.AppConfigVariable{Value: "pocketid.admin"}
				return cfg
			}(),
			groupEntry: ldapEntry("cn=admins,ou=groups,dc=example,dc=com", map[string][]string{
				"entryUUID":   {"g-display-admins"},
				"cn":          {"admins"},
				"displayName": {"pocketid.admin"},
				"member":      {"uid=testadmin,ou=people,dc=example,dc=com"},
			}),
			groupName:   "pocketid.admin",
			groupLookup: "g-display-admins",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, db := newTestLdapServiceWithAppConfig(t, tt.appConfig, newFakeLDAPClient(
				ldapSearchResult(
					ldapEntry("uid=testadmin,ou=people,dc=example,dc=com", map[string][]string{
						"entryUUID":   {"u-testadmin"},
						"uid":         {"testadmin"},
						"mail":        {"testadmin@example.com"},
						"givenName":   {"Test"},
						"sn":          {"Admin"},
						"displayName": {""},
					}),
				),
				ldapSearchResult(tt.groupEntry),
			))

			require.NoError(t, service.SyncAll(t.Context()))

			var user model.User
			require.NoError(t, db.First(&user, "ldap_id = ?", "u-testadmin").Error)
			assert.True(t, user.IsAdmin)

			var group model.UserGroup
			require.NoError(t, db.Preload("Users").First(&group, "ldap_id = ?", tt.groupLookup).Error)
			assert.Equal(t, tt.groupName, group.Name)
			assert.ElementsMatch(t, []string{"testadmin"}, usernames(group.Users))
		})
	}
}

func newTestLdapService(t *testing.T, client ldapClient) (*LdapService, *gorm.DB) {
	t.Helper()

	return newTestLdapServiceWithAppConfig(t, defaultTestLDAPAppConfig(), client)
}

func newTestLdapServiceWithAppConfig(t *testing.T, appConfigModel *model.AppConfig, client ldapClient) (*LdapService, *gorm.DB) {
	t.Helper()

	db := testutils.NewDatabaseForTest(t)

	fileStorage, err := storage.NewDatabaseStorage(db)
	require.NoError(t, err)

	appConfig := NewTestAppConfigService(appConfigModel)

	groupService := NewUserGroupService(db, appConfig, nil)
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

	service := NewLdapService(db, &http.Client{}, appConfig, userService, groupService, fileStorage)
	service.clientFactory = func() (ldapClient, error) {
		return client, nil
	}

	return service, db
}

func defaultTestLDAPAppConfig() *model.AppConfig {
	return &model.AppConfig{
		RequireUserEmail:                   model.AppConfigVariable{Value: "false"},
		LdapEnabled:                        model.AppConfigVariable{Value: "true"},
		LdapBase:                           model.AppConfigVariable{Value: "dc=example,dc=com"},
		LdapUserSearchFilter:               model.AppConfigVariable{Value: "(objectClass=person)"},
		LdapUserGroupSearchFilter:          model.AppConfigVariable{Value: "(objectClass=groupOfNames)"},
		LdapAttributeUserUniqueIdentifier:  model.AppConfigVariable{Value: "entryUUID"},
		LdapAttributeUserUsername:          model.AppConfigVariable{Value: "uid"},
		LdapAttributeUserEmail:             model.AppConfigVariable{Value: "mail"},
		LdapAttributeUserFirstName:         model.AppConfigVariable{Value: "givenName"},
		LdapAttributeUserLastName:          model.AppConfigVariable{Value: "sn"},
		LdapAttributeUserDisplayName:       model.AppConfigVariable{Value: "displayName"},
		LdapAttributeUserProfilePicture:    model.AppConfigVariable{Value: "jpegPhoto"},
		LdapAttributeGroupMember:           model.AppConfigVariable{Value: "member"},
		LdapAttributeGroupUniqueIdentifier: model.AppConfigVariable{Value: "entryUUID"},
		LdapAttributeGroupName:             model.AppConfigVariable{Value: "cn"},
		LdapAdminGroupName:                 model.AppConfigVariable{Value: "admins"},
		LdapSoftDeleteUsers:                model.AppConfigVariable{Value: "true"},
	}
}

func newFakeLDAPClient(userResult, groupResult *ldap.SearchResult) ldapClient {
	return &fakeLDAPClient{
		searchFn: func(searchRequest *ldap.SearchRequest) (*ldap.SearchResult, error) {
			switch searchRequest.Filter {
			case "(objectClass=person)":
				return userResult, nil
			case "(objectClass=groupOfNames)", "(objectClass=posixGroup)":
				return groupResult, nil
			default:
				return &ldap.SearchResult{}, nil
			}
		},
	}
}

func ldapSearchResult(entries ...*ldap.Entry) *ldap.SearchResult {
	return &ldap.SearchResult{Entries: entries}
}

func ldapEntry(dn string, attrs map[string][]string) *ldap.Entry {
	entry := &ldap.Entry{
		DN:         dn,
		Attributes: make([]*ldap.EntryAttribute, 0, len(attrs)),
	}

	for name, values := range attrs {
		entry.Attributes = append(entry.Attributes, &ldap.EntryAttribute{
			Name:   name,
			Values: values,
		})
	}

	return entry
}

func usernames(users []model.User) []string {
	result := make([]string, 0, len(users))
	for _, user := range users {
		result = append(result, user.Username)
	}

	return result
}

func TestGetDNProperty(t *testing.T) {
	tests := []struct {
		name           string
		property       string
		dn             string
		expectedResult string
	}{
		{
			name:           "simple case",
			property:       "cn",
			dn:             "cn=username,ou=people,dc=example,dc=com",
			expectedResult: "username",
		},
		{
			name:           "property not found",
			property:       "uid",
			dn:             "cn=username,ou=people,dc=example,dc=com",
			expectedResult: "",
		},
		{
			name:           "mixed case property",
			property:       "CN",
			dn:             "cn=username,ou=people,dc=example,dc=com",
			expectedResult: "username",
		},
		{
			name:           "mixed case DN",
			property:       "cn",
			dn:             "CN=username,OU=people,DC=example,DC=com",
			expectedResult: "username",
		},
		{
			name:           "spaces in DN",
			property:       "cn",
			dn:             "cn=username, ou=people, dc=example, dc=com",
			expectedResult: "username",
		},
		{
			name:           "value with special characters",
			property:       "cn",
			dn:             "cn=user.name+123,ou=people,dc=example,dc=com",
			expectedResult: "user.name+123",
		},
		{
			name:           "empty DN",
			property:       "cn",
			dn:             "",
			expectedResult: "",
		},
		{
			name:           "empty property",
			property:       "",
			dn:             "cn=username,ou=people,dc=example,dc=com",
			expectedResult: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getDNProperty(tt.property, tt.dn)
			assert.Equalf(t, tt.expectedResult, result, "getDNProperty(%q, %q)", tt.property, tt.dn)
		})
	}
}

func TestNormalizeLDAPDN(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "already normalized",
			input:    "cn=alice,dc=example,dc=com",
			expected: "cn=alice,dc=example,dc=com",
		},
		{
			name:     "uppercase attribute types",
			input:    "CN=Alice,DC=example,DC=com",
			expected: "cn=alice,dc=example,dc=com",
		},
		{
			name:     "spaces after commas",
			input:    "cn=alice, dc=example, dc=com",
			expected: "cn=alice,dc=example,dc=com",
		},
		{
			name:     "uppercase types and spaces",
			input:    "CN=Alice, DC=example, DC=com",
			expected: "cn=alice,dc=example,dc=com",
		},
		{
			name:     "multi-valued RDN",
			input:    "cn=alice+uid=a123,dc=example,dc=com",
			expected: "cn=alice+uid=a123,dc=example,dc=com",
		},
		{
			name:     "invalid DN falls back to lowercase+trim",
			input:    "  NOT A VALID DN  ",
			expected: "not a valid dn",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeLDAPDN(tt.input)
			assert.Equalf(t, tt.expected, result, "normalizeLDAPDN(%q)", tt.input)
		})
	}
}

func TestConvertLdapIdToString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "valid UTF-8 string",
			input:    "simple-utf8-id",
			expected: "simple-utf8-id",
		},
		{
			name:     "binary UUID (16 bytes)",
			input:    string([]byte{0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc, 0xde, 0xf0, 0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc, 0xde, 0xf1}),
			expected: "12345678-9abc-def0-1234-56789abcdef1",
		},
		{
			name:     "non-UTF8, non-UUID returns base64",
			input:    string([]byte{0xff, 0xfe, 0xfd, 0xfc}),
			expected: "//79/A==",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := convertLdapIdToString(tt.input)
			assert.Equal(t, tt.expected, got)
		})
	}
}
