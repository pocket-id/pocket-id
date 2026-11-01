package api

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/apperror"
	"github.com/pocket-id/pocket-id/backend/internal/authz"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	"github.com/pocket-id/pocket-id/backend/internal/oidc"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
	testutils "github.com/pocket-id/pocket-id/backend/internal/utils/testing"
)

const bootstrapTestAppURL = "https://id.example.com"

func loadBuiltInAPI(t *testing.T, db *gorm.DB) API {
	t.Helper()

	var api API
	require.NoError(t, db.Preload("Permissions").First(&api, "id = ?", BuiltInAPIID).Error)
	return api
}

func TestBootstrapInsertsBuiltInAPI(t *testing.T) {
	testutils.ForEachDatabase(t, func(t *testing.T, db *gorm.DB) {
		require.NoError(t, Bootstrap(t.Context(), db, bootstrapTestAppURL))
		require.NoError(t, Bootstrap(t.Context(), db, bootstrapTestAppURL))

		api := loadBuiltInAPI(t, db)
		assert.Equal(t, "https://id.example.com/api", api.Audience)

		keys := make([]string, len(api.Permissions))
		for i, permission := range api.Permissions {
			keys[i] = permission.Key
		}
		expected := make([]string, 0)
		for scope := range authz.UserScopes(true, authz.KindOAuthUser) {
			expected = append(expected, string(scope))
		}
		assert.ElementsMatch(t, expected, keys)
		assert.NotContains(t, keys, string(authz.AccountAPIKeysCreate), "session-only scopes must never be grantable to OAuth clients")
	})
}

func TestBootstrapFollowsAppURLAndKeepsAdminSettings(t *testing.T) {
	testutils.ForEachDatabase(t, func(t *testing.T, db *gorm.DB) {
		require.NoError(t, Bootstrap(t.Context(), db, bootstrapTestAppURL))

		// The admin opts the API in to CIMD clients and grants a client a permission
		usersRead := findPermission(loadBuiltInAPI(t, db), string(authz.UsersRead))
		require.NotNil(t, usersRead)
		require.NoError(t, db.Model(&API{}).Where("id = ?", BuiltInAPIID).Update("allow_cimd_clients", true).Error)
		require.NoError(t, db.Create(&model.OidcClient{Base: model.Base{ID: "client-1"}, Name: "Client 1"}).Error)
		require.NoError(t, db.Create(&OidcClientAllowedAPIPermission{OidcClientID: "client-1", APIPermissionID: usersRead.ID, SubjectType: oidc.SubjectTypeUser}).Error)

		require.NoError(t, Bootstrap(t.Context(), db, "https://new.example.com"))

		api := loadBuiltInAPI(t, db)
		assert.Equal(t, "https://new.example.com/api", api.Audience)
		assert.True(t, api.AllowCIMDClients)
		var grants int64
		require.NoError(t, db.Model(&OidcClientAllowedAPIPermission{}).Where("api_permission_id = ?", usersRead.ID).Count(&grants).Error)
		assert.EqualValues(t, 1, grants)
	})
}

func TestBootstrapRemovesScopesThatLeftTheCatalog(t *testing.T) {
	testutils.ForEachDatabase(t, func(t *testing.T, db *gorm.DB) {
		require.NoError(t, Bootstrap(t.Context(), db, bootstrapTestAppURL))

		// Simulate a scope that an older version had and a client was granted
		stale := Permission{APIID: BuiltInAPIID, Key: "legacy:read", Name: "legacy:read"}
		require.NoError(t, db.Create(&stale).Error)
		require.NoError(t, db.Create(&model.OidcClient{Base: model.Base{ID: "client-1"}, Name: "Client 1"}).Error)
		require.NoError(t, db.Create(&OidcClientAllowedAPI{OidcClientID: "client-1", APIID: BuiltInAPIID, SubjectType: oidc.SubjectTypeUser}).Error)
		require.NoError(t, db.Create(&OidcClientAllowedAPIPermission{OidcClientID: "client-1", APIPermissionID: stale.ID, SubjectType: oidc.SubjectTypeUser}).Error)

		require.NoError(t, Bootstrap(t.Context(), db, bootstrapTestAppURL))

		assert.Nil(t, findPermission(loadBuiltInAPI(t, db), "legacy:read"))
		var access int64
		require.NoError(t, db.Model(&OidcClientAllowedAPI{}).Where("api_id = ?", BuiltInAPIID).Count(&access).Error)
		assert.Zero(t, access, "the access implied only by the removed permission must go too")
	})
}

func TestBootstrapRefusesACustomAPIHoldingTheResource(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	require.NoError(t, db.Create(&API{Name: "Legacy", Audience: "https://id.example.com/api"}).Error)

	err := Bootstrap(t.Context(), db, bootstrapTestAppURL)
	require.ErrorContains(t, err, `"Legacy"`)
}

func TestBuiltInAPIIsReadOnly(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	svc := New(Dependencies{DB: db, Issuer: bootstrapTestAppURL}).service
	require.NoError(t, Bootstrap(t.Context(), db, bootstrapTestAppURL))

	_, err := svc.Update(t.Context(), BuiltInAPIID, apiUpdateDto{Name: "Renamed"})
	require.True(t, apperror.IsCode(err, apperror.CodeBuiltInAPIReadOnly))
	_, err = svc.UpdatePermissions(t.Context(), BuiltInAPIID, apiPermissionsUpdateDto{})
	require.True(t, apperror.IsCode(err, apperror.CodeBuiltInAPIReadOnly))
	err = svc.Delete(t.Context(), BuiltInAPIID)
	require.True(t, apperror.IsCode(err, apperror.CodeBuiltInAPIReadOnly))
}

func TestBuiltInAPICantBeDeletedThroughAnotherSpellingOfItsID(t *testing.T) {
	testutils.ForEachDatabase(t, func(t *testing.T, db *gorm.DB) {
		svc := New(Dependencies{DB: db, Issuer: bootstrapTestAppURL}).service
		require.NoError(t, Bootstrap(t.Context(), db, bootstrapTestAppURL))

		// Postgres accepts a UUID without hyphens, while SQLite finds no row at all
		require.Error(t, svc.Delete(t.Context(), strings.ReplaceAll(BuiltInAPIID, "-", "")))
		loadBuiltInAPI(t, db)
	})
}

func TestBuiltInAPIGrantsClientsOnlyAdminScopesForThemselves(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	svc := New(Dependencies{DB: db, Issuer: bootstrapTestAppURL}).service
	require.NoError(t, Bootstrap(t.Context(), db, bootstrapTestAppURL))
	require.NoError(t, db.Create(&model.OidcClient{Base: model.Base{ID: "confidential-1"}, Name: "Confidential"}).Error)

	builtIn := loadBuiltInAPI(t, db)
	usersRead := findPermission(builtIn, string(authz.UsersRead))
	accountRead := findPermission(builtIn, string(authz.AccountRead))
	applied, err := svc.SetAPIClientAccess(t.Context(), BuiltInAPIID, "confidential-1", APIClientGrant{
		UserDelegatedPermissionIDs: []string{accountRead.ID},
		ClientPermissionIDs:        []string{usersRead.ID, accountRead.ID},
	})
	require.NoError(t, err)
	assert.Equal(t, []string{accountRead.ID}, applied.UserDelegatedPermissionIDs)
	assert.True(t, applied.ClientAccess)
	assert.Equal(t, []string{usersRead.ID}, applied.ClientPermissionIDs, "a client acting as itself has no account")
}

func TestListPutsTheBuiltInAPIFirstWithoutASort(t *testing.T) {
	testutils.ForEachDatabase(t, func(t *testing.T, db *gorm.DB) {
		svc := New(Dependencies{DB: db, Issuer: bootstrapTestAppURL}).service
		require.NoError(t, Bootstrap(t.Context(), db, bootstrapTestAppURL))
		for _, name := range []string{"Zebra", "Orders"} {
			_, err := svc.Create(t.Context(), apiCreateDto{Name: name, Resource: "https://" + strings.ToLower(name) + ".example.com"})
			require.NoError(t, err)
		}

		names := func(sortColumn string) []string {
			var options utils.ListRequestOptions
			options.Sort.Column = sortColumn
			apis, _, err := svc.List(t.Context(), "", options)
			require.NoError(t, err)
			result := make([]string, len(apis))
			for i, api := range apis {
				result[i] = api.Name
			}
			return result
		}

		assert.Equal(t, []string{"Pocket ID", "Orders", "Zebra"}, names(""))
		assert.Equal(t, []string{"Orders", "Pocket ID", "Zebra"}, names("name"))
	})
}

func TestHeldScopesFollowsTheUsersRole(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	svc := New(Dependencies{DB: db, Issuer: bootstrapTestAppURL}).service
	user := model.User{Username: "regular"}
	admin := model.User{Username: "admin", IsAdmin: true}
	require.NoError(t, db.Create(&user).Error)
	require.NoError(t, db.Create(&admin).Error)

	requested := []string{"openid", string(authz.AccountRead), string(authz.UsersRead)}

	held, err := svc.HeldScopes(t.Context(), nil, user.ID, "https://id.example.com/api/", requested)
	require.NoError(t, err)
	assert.Equal(t, []string{"openid", string(authz.AccountRead)}, held)

	held, err = svc.HeldScopes(t.Context(), nil, admin.ID, "https://id.example.com/api", requested)
	require.NoError(t, err)
	assert.Equal(t, requested, held)

	// A custom API's permission keys are its own, even when one looks like a Pocket ID scope
	held, err = svc.HeldScopes(t.Context(), nil, user.ID, "https://api.orders.example.com", requested)
	require.NoError(t, err)
	assert.Equal(t, requested, held)
}
