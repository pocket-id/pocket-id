package api

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	"github.com/pocket-id/pocket-id/backend/internal/oidc"
	testutils "github.com/pocket-id/pocket-id/backend/internal/utils/testing"
)

func TestAPICrudAndPermissionDiff(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	svc := New(Dependencies{DB: db}).service

	created, err := svc.Create(t.Context(), apiCreateDto{Name: "Orders API", Resource: "https://api.orders.example.com"})
	require.NoError(t, err)
	assert.NotEmpty(t, created.ID)

	// The resource is unique.
	_, err = svc.Create(t.Context(), apiCreateDto{Name: "Dup", Resource: "https://api.orders.example.com"})
	require.ErrorIs(t, err, &common.AlreadyInUseError{})

	desc := "Read orders"
	updated, err := svc.UpdatePermissions(t.Context(), created.ID, apiPermissionsUpdateDto{Permissions: []apiPermissionInputDto{
		{Key: "read:orders", Name: "Read orders", Description: &desc},
		{Key: "write:orders", Name: "Write orders"},
	}})
	require.NoError(t, err)
	assert.Len(t, updated.Permissions, 2)

	// Grant a client the read:orders permission for both subject types, then remove that permission
	// and confirm the grants are cleaned up while write:orders (and its key) survives.
	readPerm := findPermission(updated, "read:orders")
	require.NotNil(t, readPerm)
	require.NoError(t, db.Create(&model.OidcClient{Base: model.Base{ID: "client-1"}, Name: "Client 1"}).Error)
	require.NoError(t, db.Create(&OidcClientAllowedAPIPermission{OidcClientID: "client-1", APIPermissionID: readPerm.ID, SubjectType: oidc.SubjectTypeUser}).Error)
	require.NoError(t, db.Create(&OidcClientAllowedAPIPermission{OidcClientID: "client-1", APIPermissionID: readPerm.ID, SubjectType: oidc.SubjectTypeClient}).Error)

	updated, err = svc.UpdatePermissions(t.Context(), created.ID, apiPermissionsUpdateDto{Permissions: []apiPermissionInputDto{
		{Key: "write:orders", Name: "Write orders (renamed)"},
	}})
	require.NoError(t, err)
	require.Len(t, updated.Permissions, 1)
	assert.Equal(t, "write:orders", updated.Permissions[0].Key)
	assert.Equal(t, "Write orders (renamed)", updated.Permissions[0].Name)

	var grantCount int64
	require.NoError(t, db.Model(&OidcClientAllowedAPIPermission{}).Where("api_permission_id = ?", readPerm.ID).Count(&grantCount).Error)
	assert.Equal(t, int64(0), grantCount)

	renamed, err := svc.Update(t.Context(), created.ID, apiUpdateDto{Name: "Orders"})
	require.NoError(t, err)
	assert.Equal(t, "Orders", renamed.Name)
	require.NotNil(t, renamed.UpdatedAt)

	require.NoError(t, svc.Delete(t.Context(), created.ID))
	_, err = svc.Get(t.Context(), nil, created.ID)
	require.Error(t, err)
}

func TestClientApiAccessAllowList(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	svc := New(Dependencies{DB: db}).service

	require.NoError(t, db.Create(&model.OidcClient{Base: model.Base{ID: "client-1"}, Name: "Client 1"}).Error)

	orders, err := svc.Create(t.Context(), apiCreateDto{Name: "Orders", Resource: "https://api.orders.example.com"})
	require.NoError(t, err)
	orders, err = svc.UpdatePermissions(t.Context(), orders.ID, apiPermissionsUpdateDto{Permissions: []apiPermissionInputDto{
		{Key: "read:orders", Name: "Read"},
		{Key: "write:orders", Name: "Write"},
	}})
	require.NoError(t, err)
	readID := findPermission(orders, "read:orders").ID
	writeID := findPermission(orders, "write:orders").ID

	// Unknown IDs are filtered out, and the subject types are stored independently.
	applied, err := svc.SetClientAPIAccess(t.Context(), "client-1", ClientAPIAccess{
		UserDelegatedPermissionIDs: []string{readID, "does-not-exist"},
		ClientPermissionIDs:        []string{writeID, "does-not-exist"},
	})
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{readID}, applied.UserDelegatedPermissionIDs)
	assert.ElementsMatch(t, []string{writeID}, applied.ClientPermissionIDs)

	got, err := svc.GetClientAPIAccess(t.Context(), "client-1")
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{readID}, got.UserDelegatedPermissionIDs)
	assert.ElementsMatch(t, []string{writeID}, got.ClientPermissionIDs)

	// The same permission can be granted for both subject types, and both sets are fully replaced on each call.
	_, err = svc.SetClientAPIAccess(t.Context(), "client-1", ClientAPIAccess{
		UserDelegatedPermissionIDs: []string{readID, writeID},
		ClientPermissionIDs:        []string{readID},
	})
	require.NoError(t, err)
	got, err = svc.GetClientAPIAccess(t.Context(), "client-1")
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{readID, writeID}, got.UserDelegatedPermissionIDs)
	assert.ElementsMatch(t, []string{readID}, got.ClientPermissionIDs)

	// Clearing one subject type leaves the other untouched.
	_, err = svc.SetClientAPIAccess(t.Context(), "client-1", ClientAPIAccess{ClientPermissionIDs: []string{readID}})
	require.NoError(t, err)
	got, err = svc.GetClientAPIAccess(t.Context(), "client-1")
	require.NoError(t, err)
	assert.Empty(t, got.UserDelegatedPermissionIDs)
	assert.ElementsMatch(t, []string{readID}, got.ClientPermissionIDs)

	// Clearing everything.
	_, err = svc.SetClientAPIAccess(t.Context(), "client-1", ClientAPIAccess{})
	require.NoError(t, err)
	got, err = svc.GetClientAPIAccess(t.Context(), "client-1")
	require.NoError(t, err)
	assert.Empty(t, got.UserDelegatedPermissionIDs)
	assert.Empty(t, got.ClientPermissionIDs)

	// An unknown client is rejected (surfaces as 404 at the HTTP layer).
	_, err = svc.SetClientAPIAccess(t.Context(), "nope", ClientAPIAccess{UserDelegatedPermissionIDs: []string{readID}})
	require.Error(t, err)
}

// TestAllowedScopesForAudienceFiltersBySubjectType guards that the scopes resolved for a flow
// only come from the grants of that flow's subject type.
func TestAllowedScopesForAudienceFiltersBySubjectType(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	svc := New(Dependencies{DB: db}).service

	require.NoError(t, db.Create(&model.OidcClient{Base: model.Base{ID: "client-1"}, Name: "Client 1"}).Error)

	orders, err := svc.Create(t.Context(), apiCreateDto{Name: "Orders", Resource: "https://api.orders.example.com"})
	require.NoError(t, err)
	orders, err = svc.UpdatePermissions(t.Context(), orders.ID, apiPermissionsUpdateDto{Permissions: []apiPermissionInputDto{
		{Key: "read:orders", Name: "Read"},
		{Key: "write:orders", Name: "Write"},
	}})
	require.NoError(t, err)
	readID := findPermission(orders, "read:orders").ID
	writeID := findPermission(orders, "write:orders").ID

	_, err = svc.SetClientAPIAccess(t.Context(), "client-1", ClientAPIAccess{
		UserDelegatedPermissionIDs: []string{readID},
		ClientPermissionIDs:        []string{writeID},
	})
	require.NoError(t, err)

	userScopes, exists, err := svc.AllowedScopesForAudience(t.Context(), "client-1", "https://api.orders.example.com", oidc.SubjectTypeUser)
	require.NoError(t, err)
	require.True(t, exists)
	assert.ElementsMatch(t, []string{"read:orders"}, userScopes)

	clientScopes, exists, err := svc.AllowedScopesForAudience(t.Context(), "client-1", "https://api.orders.example.com", oidc.SubjectTypeClient)
	require.NoError(t, err)
	require.True(t, exists)
	assert.ElementsMatch(t, []string{"write:orders"}, clientScopes)

	// The fosite widening still sees the union of both subject types.
	scopes, audiences, err := svc.ClientAPIScopesAndAudiences(t.Context(), nil, "client-1")
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"read:orders", "write:orders"}, scopes)
	assert.ElementsMatch(t, []string{"https://api.orders.example.com"}, audiences)
}

func TestUpdatePermissionsRejectsReservedKeys(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	svc := New(Dependencies{DB: db}).service

	orders, err := svc.Create(t.Context(), apiCreateDto{Name: "Orders", Resource: "https://api.orders.example.com"})
	require.NoError(t, err)

	for _, key := range []string{"openid", "profile", "email", "email_verified", "groups", "offline_access", "Email"} {
		_, err := svc.UpdatePermissions(t.Context(), orders.ID, apiPermissionsUpdateDto{Permissions: []apiPermissionInputDto{
			{Key: key, Name: "Reserved"},
		}})
		require.Error(t, err, "key %q must be rejected", key)
		var validationErr *common.ValidationError
		require.ErrorAs(t, err, &validationErr)
	}
}

func TestUpdatePermissionsRejectsInvalidKeyCharacters(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	svc := New(Dependencies{DB: db}).service

	orders, err := svc.Create(t.Context(), apiCreateDto{Name: "Orders", Resource: "https://api.orders.example.com"})
	require.NoError(t, err)

	// A space corrupts the space-delimited scope claim, and the unit separator is the consent delimiter
	for _, key := range []string{"read orders", "read\x1forders", "read\"orders", "bad\\key", "tab\tkey"} {
		_, err := svc.UpdatePermissions(t.Context(), orders.ID, apiPermissionsUpdateDto{Permissions: []apiPermissionInputDto{
			{Key: key, Name: "Invalid"},
		}})
		require.Error(t, err, "key %q must be rejected", key)
		var validationErr *common.ValidationError
		require.ErrorAs(t, err, &validationErr)
	}

	// A valid scope-token key is accepted
	_, err = svc.UpdatePermissions(t.Context(), orders.ID, apiPermissionsUpdateDto{Permissions: []apiPermissionInputDto{
		{Key: "read:orders", Name: "Read"},
	}})
	require.NoError(t, err)
}

func TestCreateRejectsIssuerResource(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	const issuer = "https://id.example.com"
	svc := New(Dependencies{DB: db, Issuer: issuer}).service

	// The issuer itself, a trailing-slash variant, and a different-cased variant are all reserved
	for _, resource := range []string{issuer, issuer + "/", "https://ID.example.com"} {
		_, err := svc.Create(t.Context(), apiCreateDto{Name: "Reserved", Resource: resource})
		require.Error(t, err, "resource %q must be rejected", resource)
		var validationErr *common.ValidationError
		require.ErrorAs(t, err, &validationErr)
	}

	// A normal resource is accepted
	_, err := svc.Create(t.Context(), apiCreateDto{Name: "Orders", Resource: "https://api.orders.example.com"})
	require.NoError(t, err)
}

func TestCreateAcceptsAbsoluteResourceURIs(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	svc := New(Dependencies{DB: db}).service

	for _, resource := range []string{"https://api.orders.example.com", "api://PocketID", "urn:my-app"} {
		_, err := svc.Create(t.Context(), apiCreateDto{Name: "Orders", Resource: resource})
		require.NoError(t, err, "resource %q must be accepted", resource)
	}
}

func TestDescribePermissions(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	svc := New(Dependencies{DB: db}).service

	orders, err := svc.Create(t.Context(), apiCreateDto{Name: "Orders", Resource: "https://api.orders.example.com"})
	require.NoError(t, err)
	desc := "Read orders"
	_, err = svc.UpdatePermissions(t.Context(), orders.ID, apiPermissionsUpdateDto{Permissions: []apiPermissionInputDto{
		{Key: "read:orders", Name: "Read orders", Description: &desc},
		{Key: "write:orders", Name: "Write orders"},
	}})
	require.NoError(t, err)

	infos, err := svc.DescribePermissions(t.Context(), "https://api.orders.example.com", []string{"read:orders", "unknown"})
	require.NoError(t, err)
	require.Len(t, infos, 1)
	assert.Equal(t, "read:orders", infos[0].Key)
	assert.Equal(t, "Read orders", infos[0].Name)
	require.NotNil(t, infos[0].Description)
	assert.Equal(t, "Read orders", *infos[0].Description)
}

func findPermission(api API, key string) *Permission {
	for i := range api.Permissions {
		if api.Permissions[i].Key == key {
			return &api.Permissions[i]
		}
	}
	return nil
}
