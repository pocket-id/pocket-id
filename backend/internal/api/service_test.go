package api

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pocket-id/pocket-id/backend/internal/apperror"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	"github.com/pocket-id/pocket-id/backend/internal/oidc"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
	testutils "github.com/pocket-id/pocket-id/backend/internal/utils/testing"
)

func TestAPICrudAndPermissionDiff(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	// Removing a permission (or an API) relies on the ON DELETE CASCADE that production enforces, so exercise it here
	// The shared test harness disables foreign keys, so enable them for this connection
	require.NoError(t, db.Exec("PRAGMA foreign_keys = ON").Error)
	svc := New(Dependencies{DB: db}).service

	created, err := svc.Create(t.Context(), apiCreateDto{Name: "Orders API", Resource: "https://api.orders.example.com"})
	require.NoError(t, err)
	assert.NotEmpty(t, created.ID)

	// The resource is unique.
	_, err = svc.Create(t.Context(), apiCreateDto{Name: "Dup", Resource: "https://api.orders.example.com"})
	require.True(t, apperror.IsCode(err, apperror.CodeAlreadyInUse))

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
	require.True(t, apperror.IsCode(err, apperror.CodeNotFound))
}

// clientGrantFor returns the client's grant on one API from the client-side listing, which is what remains after the
// per-client bulk setter/getter were removed in favour of the single write path on /apis/:id/clients/:clientId.
func clientGrantFor(t *testing.T, svc *Service, clientID, apiID string) ClientAPIGrant {
	t.Helper()
	grants, err := svc.ListClientAPIs(t.Context(), clientID)
	require.NoError(t, err)
	for _, grant := range grants {
		if grant.API.ID == apiID {
			return grant
		}
	}
	return ClientAPIGrant{}
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

	// Unknown IDs are filtered out, the subject types are stored independently, and a permission implies access to its API.
	applied, err := svc.SetAPIClientAccess(t.Context(), orders.ID, "client-1", APIClientGrant{
		UserDelegatedPermissionIDs: []string{readID, "does-not-exist"},
		ClientPermissionIDs:        []string{writeID, "does-not-exist"},
	})
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{readID}, applied.UserDelegatedPermissionIDs)
	assert.ElementsMatch(t, []string{writeID}, applied.ClientPermissionIDs)
	assert.True(t, applied.UserDelegatedAccess)
	assert.True(t, applied.ClientAccess)

	got := clientGrantFor(t, svc, "client-1", orders.ID)
	assert.ElementsMatch(t, []string{readID}, got.UserDelegatedPermissionIDs)
	assert.ElementsMatch(t, []string{writeID}, got.ClientPermissionIDs)
	assert.True(t, got.UserDelegatedAccess)
	assert.True(t, got.ClientAccess)

	// The same permission can be granted for both subject types, and both sets are fully replaced on each call.
	_, err = svc.SetAPIClientAccess(t.Context(), orders.ID, "client-1", APIClientGrant{
		UserDelegatedPermissionIDs: []string{readID, writeID},
		ClientPermissionIDs:        []string{readID},
	})
	require.NoError(t, err)
	got = clientGrantFor(t, svc, "client-1", orders.ID)
	assert.ElementsMatch(t, []string{readID, writeID}, got.UserDelegatedPermissionIDs)
	assert.ElementsMatch(t, []string{readID}, got.ClientPermissionIDs)

	// Clearing one subject type leaves the other untouched.
	_, err = svc.SetAPIClientAccess(t.Context(), orders.ID, "client-1", APIClientGrant{ClientPermissionIDs: []string{readID}})
	require.NoError(t, err)
	got = clientGrantFor(t, svc, "client-1", orders.ID)
	assert.Empty(t, got.UserDelegatedPermissionIDs)
	assert.False(t, got.UserDelegatedAccess)
	assert.ElementsMatch(t, []string{readID}, got.ClientPermissionIDs)

	// Clearing everything drops the API from the client's list entirely.
	_, err = svc.SetAPIClientAccess(t.Context(), orders.ID, "client-1", APIClientGrant{})
	require.NoError(t, err)
	grants, err := svc.ListClientAPIs(t.Context(), "client-1")
	require.NoError(t, err)
	assert.Empty(t, grants)

	// An API can be granted on its own, without a single permission.
	applied, err = svc.SetAPIClientAccess(t.Context(), orders.ID, "client-1", APIClientGrant{UserDelegatedAccess: true})
	require.NoError(t, err)
	assert.True(t, applied.UserDelegatedAccess)
	assert.Empty(t, applied.UserDelegatedPermissionIDs)
	got = clientGrantFor(t, svc, "client-1", orders.ID)
	assert.True(t, got.UserDelegatedAccess)
	assert.False(t, got.ClientAccess)

	// An unknown client is rejected (surfaces as 404 at the HTTP layer).
	_, err = svc.SetAPIClientAccess(t.Context(), orders.ID, "nope", APIClientGrant{UserDelegatedPermissionIDs: []string{readID}})
	require.True(t, apperror.IsCode(err, apperror.CodeNotFound))

	_, err = svc.ListClientAPIs(t.Context(), "nope")
	require.True(t, apperror.IsCode(err, apperror.CodeNotFound))
}

// TestSetAPIClientAccessDropsClientGrantsForPublicClients guards that a machine-to-machine grant cannot be written for a
// client that can never authenticate for the client credentials grant, even by a direct API call that bypasses the UI.
func TestSetAPIClientAccessDropsClientGrantsForPublicClients(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	svc := New(Dependencies{DB: db}).service

	require.NoError(t, db.Create(&model.OidcClient{Base: model.Base{ID: "public-1"}, Name: "Public", IsPublic: true}).Error)

	orders, err := svc.Create(t.Context(), apiCreateDto{Name: "Orders", Resource: "https://api.orders.example.com"})
	require.NoError(t, err)
	orders, err = svc.UpdatePermissions(t.Context(), orders.ID, apiPermissionsUpdateDto{Permissions: []apiPermissionInputDto{
		{Key: "write:orders", Name: "Write"},
	}})
	require.NoError(t, err)
	writeID := findPermission(orders, "write:orders").ID

	applied, err := svc.SetAPIClientAccess(t.Context(), orders.ID, "public-1", APIClientGrant{
		UserDelegatedAccess: true,
		ClientAccess:        true,
		ClientPermissionIDs: []string{writeID},
	})
	require.NoError(t, err)
	assert.False(t, applied.ClientAccess)
	assert.Empty(t, applied.ClientPermissionIDs)
	assert.True(t, applied.UserDelegatedAccess)

	_, _, hasAccess, err := svc.AllowedScopesForAudience(t.Context(), nil, "public-1", "https://api.orders.example.com", oidc.SubjectTypeClient)
	require.NoError(t, err)
	assert.False(t, hasAccess)
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

	_, err = svc.SetAPIClientAccess(t.Context(), orders.ID, "client-1", APIClientGrant{
		UserDelegatedPermissionIDs: []string{readID},
		ClientPermissionIDs:        []string{writeID},
	})
	require.NoError(t, err)

	userScopes, exists, hasAccess, err := svc.AllowedScopesForAudience(t.Context(), nil, "client-1", "https://api.orders.example.com", oidc.SubjectTypeUser)
	require.NoError(t, err)
	require.True(t, exists)
	require.True(t, hasAccess)
	assert.ElementsMatch(t, []string{"read:orders"}, userScopes)

	clientScopes, exists, hasAccess, err := svc.AllowedScopesForAudience(t.Context(), nil, "client-1", "https://api.orders.example.com", oidc.SubjectTypeClient)
	require.NoError(t, err)
	require.True(t, exists)
	require.True(t, hasAccess)
	assert.ElementsMatch(t, []string{"write:orders"}, clientScopes)

	// The fosite widening still sees the union of both subject types.
	scopes, audiences, err := svc.ClientAPIScopesAndAudiences(t.Context(), nil, "client-1", false)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"read:orders", "write:orders"}, scopes)
	assert.ElementsMatch(t, []string{"https://api.orders.example.com"}, audiences)
}

// TestAccessWithoutPermissions covers granting an API to a client without any permission, which is what
// an MCP client needs: the resource is reachable and the token simply carries no custom scope.
func TestAccessWithoutPermissions(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	svc := New(Dependencies{DB: db}).service

	const resource = "https://api.orders.example.com"
	require.NoError(t, db.Create(&model.OidcClient{Base: model.Base{ID: "client-1"}, Name: "Client 1"}).Error)

	orders, err := svc.Create(t.Context(), apiCreateDto{Name: "Orders", Resource: resource})
	require.NoError(t, err)
	orders, err = svc.UpdatePermissions(t.Context(), orders.ID, apiPermissionsUpdateDto{Permissions: []apiPermissionInputDto{
		{Key: "read:orders", Name: "Read"},
	}})
	require.NoError(t, err)

	// Without any grant the client cannot reach the API at all
	scopes, exists, hasAccess, err := svc.AllowedScopesForAudience(t.Context(), nil, "client-1", resource, oidc.SubjectTypeUser)
	require.NoError(t, err)
	require.True(t, exists)
	assert.False(t, hasAccess)
	assert.Empty(t, scopes)

	_, err = svc.SetAPIClientAccess(t.Context(), orders.ID, "client-1", APIClientGrant{UserDelegatedAccess: true})
	require.NoError(t, err)

	// Access is granted for user-delegated flows only, and it comes with no scope
	scopes, _, hasAccess, err = svc.AllowedScopesForAudience(t.Context(), nil, "client-1", resource, oidc.SubjectTypeUser)
	require.NoError(t, err)
	assert.True(t, hasAccess)
	assert.Empty(t, scopes)

	_, _, hasAccess, err = svc.AllowedScopesForAudience(t.Context(), nil, "client-1", resource, oidc.SubjectTypeClient)
	require.NoError(t, err)
	assert.False(t, hasAccess)

	// Fosite still has to accept the audience, otherwise the request is rejected before the resource is resolved
	scopes, audiences, err := svc.ClientAPIScopesAndAudiences(t.Context(), nil, "client-1", false)
	require.NoError(t, err)
	assert.Empty(t, scopes)
	assert.ElementsMatch(t, []string{resource}, audiences)
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
		require.True(t, apperror.IsCode(err, apperror.CodeValidationFailed))
	}
}

func TestUpdatePermissionsRejectsDuplicateKeys(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	svc := New(Dependencies{DB: db}).service

	orders, err := svc.Create(t.Context(), apiCreateDto{Name: "Orders", Resource: "https://api.orders.example.com"})
	require.NoError(t, err)

	// Two rows with the same key must be rejected rather than silently coalesced last-wins
	_, err = svc.UpdatePermissions(t.Context(), orders.ID, apiPermissionsUpdateDto{Permissions: []apiPermissionInputDto{
		{Key: "read:orders", Name: "Read"},
		{Key: "read:orders", Name: "Read again"},
	}})
	require.Error(t, err)
	require.True(t, apperror.IsCode(err, apperror.CodeValidationFailed))
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
		require.True(t, apperror.IsCode(err, apperror.CodeValidationFailed))
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
		require.True(t, apperror.IsCode(err, apperror.CodeValidationFailed))
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

// TestCimdClientAccess covers the API-wide opt-in that lets every metadata document client reach an API without an individual grant.
func TestCimdClientAccess(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	svc := New(Dependencies{DB: db}).service

	const cimdClientID = "https://app.example.com/oauth-client.json"
	const resource = "https://api.orders.example.com"
	require.NoError(t, db.Create(&model.OidcClient{Base: model.Base{ID: cimdClientID}, Name: "MCP client", ClientType: model.OidcClientTypeCIMD, IsPublic: true}).Error)
	require.NoError(t, db.Create(&model.OidcClient{Base: model.Base{ID: "client-1"}, Name: "Client 1"}).Error)

	orders, err := svc.Create(t.Context(), apiCreateDto{Name: "Orders", Resource: resource})
	require.NoError(t, err)
	orders, err = svc.UpdatePermissions(t.Context(), orders.ID, apiPermissionsUpdateDto{Permissions: []apiPermissionInputDto{
		{Key: "read:orders", Name: "Read"},
		{Key: "write:orders", Name: "Write"},
	}})
	require.NoError(t, err)
	readID := findPermission(orders, "read:orders").ID
	writeID := findPermission(orders, "write:orders").ID

	// Before the API opts in, a metadata document client has no more access than any other client.
	scopes, exists, hasAccess, err := svc.AllowedScopesForAudience(t.Context(), nil, cimdClientID, resource, oidc.SubjectTypeUser)
	require.NoError(t, err)
	require.True(t, exists)
	assert.False(t, hasAccess)
	assert.Empty(t, scopes)

	// Permissions of other APIs are ignored, the same way unknown client grants are.
	updated, err := svc.SetCIMDAccess(t.Context(), orders.ID, apiCimdAccessUpdateDto{Enabled: true, PermissionIDs: []string{readID, "does-not-exist"}})
	require.NoError(t, err)
	assert.True(t, updated.AllowCIMDClients)
	assert.True(t, findPermission(updated, "read:orders").AllowedForCIMDClients)
	assert.False(t, findPermission(updated, "write:orders").AllowedForCIMDClients)

	// The metadata document client now reaches the API without an individual grant, but only with the selected permissions.
	scopes, exists, hasAccess, err = svc.AllowedScopesForAudience(t.Context(), nil, cimdClientID, resource, oidc.SubjectTypeUser)
	require.NoError(t, err)
	require.True(t, exists)
	assert.True(t, hasAccess)
	assert.ElementsMatch(t, []string{"read:orders"}, scopes)

	// The opt-in never covers the client credentials grant, and never applies to regularly registered clients.
	scopes, _, hasAccess, err = svc.AllowedScopesForAudience(t.Context(), nil, cimdClientID, resource, oidc.SubjectTypeClient)
	require.NoError(t, err)
	assert.False(t, hasAccess)
	assert.Empty(t, scopes)
	scopes, _, hasAccess, err = svc.AllowedScopesForAudience(t.Context(), nil, "client-1", resource, oidc.SubjectTypeUser)
	require.NoError(t, err)
	assert.False(t, hasAccess)
	assert.Empty(t, scopes)

	// The fosite scope and audience widening has to see the implicit access too, otherwise the request is rejected before the resource is resolved.
	widened, audiences, err := svc.ClientAPIScopesAndAudiences(t.Context(), nil, cimdClientID, true)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"read:orders"}, widened)
	assert.ElementsMatch(t, []string{resource}, audiences)

	// An individual grant of the same permission does not show up twice.
	_, err = svc.SetAPIClientAccess(t.Context(), orders.ID, cimdClientID, APIClientGrant{UserDelegatedPermissionIDs: []string{readID}, ClientPermissionIDs: []string{writeID}})
	require.NoError(t, err)
	scopes, _, _, err = svc.AllowedScopesForAudience(t.Context(), nil, cimdClientID, resource, oidc.SubjectTypeUser)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"read:orders"}, scopes)

	// The admin view keeps implicit access apart from the grants that can be edited per client.
	grant := clientGrantFor(t, svc, cimdClientID, orders.ID)
	assert.True(t, grant.CIMDGrantedAccess)
	assert.ElementsMatch(t, []string{readID}, grant.CIMDGrantedPermissionIDs)
	assert.ElementsMatch(t, []string{readID}, grant.UserDelegatedPermissionIDs)
	grant = clientGrantFor(t, svc, "client-1", orders.ID)
	assert.False(t, grant.CIMDGrantedAccess)
	assert.Empty(t, grant.CIMDGrantedPermissionIDs)

	// Opening the API without selecting any permission still lets a metadata document client reach it, only without a scope.
	_, err = svc.SetAPIClientAccess(t.Context(), orders.ID, cimdClientID, APIClientGrant{})
	require.NoError(t, err)
	_, err = svc.SetCIMDAccess(t.Context(), orders.ID, apiCimdAccessUpdateDto{Enabled: true})
	require.NoError(t, err)
	scopes, _, hasAccess, err = svc.AllowedScopesForAudience(t.Context(), nil, cimdClientID, resource, oidc.SubjectTypeUser)
	require.NoError(t, err)
	assert.True(t, hasAccess)
	assert.Empty(t, scopes)
	_, audiences, err = svc.ClientAPIScopesAndAudiences(t.Context(), nil, cimdClientID, true)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{resource}, audiences)

	// Switching the access off revokes it but keeps the selection, so it can be turned back on unchanged.
	updated, err = svc.SetCIMDAccess(t.Context(), orders.ID, apiCimdAccessUpdateDto{Enabled: false, PermissionIDs: []string{readID}})
	require.NoError(t, err)
	assert.False(t, updated.AllowCIMDClients)
	assert.True(t, findPermission(updated, "read:orders").AllowedForCIMDClients)
	_, _, hasAccess, err = svc.AllowedScopesForAudience(t.Context(), nil, cimdClientID, resource, oidc.SubjectTypeUser)
	require.NoError(t, err)
	assert.False(t, hasAccess)
	grant = clientGrantFor(t, svc, cimdClientID, orders.ID)
	assert.False(t, grant.CIMDGrantedAccess)
	assert.Empty(t, grant.CIMDGrantedPermissionIDs)

	// An API reached only through the opt-in is still listed for the client, marked as coming from it
	_, err = svc.SetCIMDAccess(t.Context(), orders.ID, apiCimdAccessUpdateDto{Enabled: true, PermissionIDs: []string{readID}})
	require.NoError(t, err)
	clientAPIs, err := svc.ListClientAPIs(t.Context(), cimdClientID)
	require.NoError(t, err)
	require.Len(t, clientAPIs, 1)
	assert.Equal(t, orders.ID, clientAPIs[0].API.ID)
	assert.True(t, clientAPIs[0].CIMDGrantedAccess)
	assert.ElementsMatch(t, []string{readID}, clientAPIs[0].CIMDGrantedPermissionIDs)
	assert.False(t, clientAPIs[0].UserDelegatedAccess)
	assert.Empty(t, clientAPIs[0].UserDelegatedPermissionIDs)

	// It is listed already, so the selection does not offer it a second time
	assignable, _, err := svc.ListAssignableAPIs(t.Context(), cimdClientID, "", utils.ListRequestOptions{})
	require.NoError(t, err)
	assert.Empty(t, assignable)
	assignable, _, err = svc.ListAssignableAPIs(t.Context(), "client-1", "", utils.ListRequestOptions{})
	require.NoError(t, err)
	require.Len(t, assignable, 1)
}

// TestApiSideClientGrants covers managing the client grants from the API's side of the relation.
func TestApiSideClientGrants(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	svc := New(Dependencies{DB: db}).service

	require.NoError(t, db.Create(&model.OidcClient{Base: model.Base{ID: "client-1"}, Name: "Zulu"}).Error)
	require.NoError(t, db.Create(&model.OidcClient{Base: model.Base{ID: "client-2"}, Name: "Alpha"}).Error)

	orders, err := svc.Create(t.Context(), apiCreateDto{Name: "Orders", Resource: "https://api.orders.example.com"})
	require.NoError(t, err)
	orders, err = svc.UpdatePermissions(t.Context(), orders.ID, apiPermissionsUpdateDto{Permissions: []apiPermissionInputDto{
		{Key: "read:orders", Name: "Read"},
		{Key: "write:orders", Name: "Write"},
	}})
	require.NoError(t, err)
	readID := findPermission(orders, "read:orders").ID
	writeID := findPermission(orders, "write:orders").ID

	billing, err := svc.Create(t.Context(), apiCreateDto{Name: "Billing", Resource: "https://api.billing.example.com"})
	require.NoError(t, err)
	billing, err = svc.UpdatePermissions(t.Context(), billing.ID, apiPermissionsUpdateDto{Permissions: []apiPermissionInputDto{
		{Key: "read:invoices", Name: "Read invoices"},
	}})
	require.NoError(t, err)
	invoicesID := findPermission(billing, "read:invoices").ID

	// An API without grants lists no clients.
	clients, _, err := svc.ListAPIClients(t.Context(), orders.ID, "", utils.ListRequestOptions{})
	require.NoError(t, err)
	assert.Empty(t, clients)

	// Permissions of another API cannot be granted through this API, and a permission implies access for its subject type.
	applied, err := svc.SetAPIClientAccess(t.Context(), orders.ID, "client-1", APIClientGrant{
		UserDelegatedPermissionIDs: []string{readID, invoicesID},
		ClientPermissionIDs:        []string{writeID},
	})
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{readID}, applied.UserDelegatedPermissionIDs)
	assert.ElementsMatch(t, []string{writeID}, applied.ClientPermissionIDs)
	assert.True(t, applied.UserDelegatedAccess)
	assert.True(t, applied.ClientAccess)

	_, err = svc.SetAPIClientAccess(t.Context(), billing.ID, "client-1", APIClientGrant{UserDelegatedPermissionIDs: []string{invoicesID}})
	require.NoError(t, err)

	// Replacing the grants of one API leaves what the client holds on other APIs untouched.
	_, err = svc.SetAPIClientAccess(t.Context(), orders.ID, "client-1", APIClientGrant{UserDelegatedPermissionIDs: []string{readID, writeID}})
	require.NoError(t, err)
	ordersGrant := clientGrantFor(t, svc, "client-1", orders.ID)
	billingGrant := clientGrantFor(t, svc, "client-1", billing.ID)
	assert.ElementsMatch(t, []string{readID, writeID}, ordersGrant.UserDelegatedPermissionIDs)
	assert.ElementsMatch(t, []string{invoicesID}, billingGrant.UserDelegatedPermissionIDs)
	assert.True(t, ordersGrant.UserDelegatedAccess)
	assert.True(t, billingGrant.UserDelegatedAccess)
	assert.Empty(t, ordersGrant.ClientPermissionIDs)
	assert.False(t, ordersGrant.ClientAccess)

	// Each client is listed once with only the permissions it holds on this API, ordered by name.
	_, err = svc.SetAPIClientAccess(t.Context(), orders.ID, "client-2", APIClientGrant{ClientPermissionIDs: []string{writeID}})
	require.NoError(t, err)
	clients, _, err = svc.ListAPIClients(t.Context(), orders.ID, "", utils.ListRequestOptions{})
	require.NoError(t, err)
	require.Len(t, clients, 2)
	assert.Equal(t, "client-2", clients[0].Client.ID)
	assert.Empty(t, clients[0].UserDelegatedPermissionIDs)
	assert.ElementsMatch(t, []string{writeID}, clients[0].ClientPermissionIDs)
	assert.Equal(t, "client-1", clients[1].Client.ID)
	assert.ElementsMatch(t, []string{readID, writeID}, clients[1].UserDelegatedPermissionIDs)

	// A client granted access without a single permission still shows up as having access to the API.
	_, err = svc.SetAPIClientAccess(t.Context(), billing.ID, "client-2", APIClientGrant{UserDelegatedAccess: true})
	require.NoError(t, err)
	billingClients, _, err := svc.ListAPIClients(t.Context(), billing.ID, "", utils.ListRequestOptions{})
	require.NoError(t, err)
	require.Len(t, billingClients, 2)
	assert.Equal(t, "client-2", billingClients[0].Client.ID)
	assert.True(t, billingClients[0].UserDelegatedAccess)
	assert.False(t, billingClients[0].ClientAccess)
	assert.Empty(t, billingClients[0].UserDelegatedPermissionIDs)

	// Revoking access to one API keeps the grants of the other one.
	require.NoError(t, svc.RemoveAPIClientAccess(t.Context(), orders.ID, "client-1"))
	remaining, err := svc.ListClientAPIs(t.Context(), "client-1")
	require.NoError(t, err)
	require.Len(t, remaining, 1)
	assert.Equal(t, billing.ID, remaining[0].API.ID)
	assert.ElementsMatch(t, []string{invoicesID}, remaining[0].UserDelegatedPermissionIDs)

	// The client's side of the relation lists the same grants, one row per API it may reach.
	clientAPIs, err := svc.ListClientAPIs(t.Context(), "client-2")
	require.NoError(t, err)
	require.Len(t, clientAPIs, 2)
	assert.Equal(t, billing.ID, clientAPIs[0].API.ID)
	assert.True(t, clientAPIs[0].UserDelegatedAccess)
	assert.Empty(t, clientAPIs[0].UserDelegatedPermissionIDs)
	assert.Equal(t, orders.ID, clientAPIs[1].API.ID)
	assert.Len(t, clientAPIs[1].API.Permissions, 2)
	assert.True(t, clientAPIs[1].ClientAccess)
	assert.ElementsMatch(t, []string{writeID}, clientAPIs[1].ClientPermissionIDs)

	// The selections an admin picks from only offer what is not granted yet, so pagination reflects what can still be added.
	assignableClients, pagination, err := svc.ListAssignableClients(t.Context(), orders.ID, "", utils.ListRequestOptions{})
	require.NoError(t, err)
	require.Len(t, assignableClients, 1)
	assert.Equal(t, "client-1", assignableClients[0].ID)
	assert.Equal(t, int64(1), pagination.TotalItems)

	assignableAPIs, _, err := svc.ListAssignableAPIs(t.Context(), "client-2", "", utils.ListRequestOptions{})
	require.NoError(t, err)
	require.Empty(t, assignableAPIs)
	assignableAPIs, _, err = svc.ListAssignableAPIs(t.Context(), "client-1", "", utils.ListRequestOptions{})
	require.NoError(t, err)
	require.Len(t, assignableAPIs, 1)
	assert.Equal(t, orders.ID, assignableAPIs[0].ID)

	// An unknown API or client is rejected (surfaces as 404 at the HTTP layer).
	_, _, err = svc.ListAPIClients(t.Context(), "nope", "", utils.ListRequestOptions{})
	require.True(t, apperror.IsCode(err, apperror.CodeNotFound))
	_, err = svc.SetAPIClientAccess(t.Context(), "nope", "client-1", APIClientGrant{})
	require.True(t, apperror.IsCode(err, apperror.CodeNotFound))
	_, err = svc.SetAPIClientAccess(t.Context(), orders.ID, "nope", APIClientGrant{})
	require.True(t, apperror.IsCode(err, apperror.CodeNotFound))
	require.True(t, apperror.IsCode(svc.RemoveAPIClientAccess(t.Context(), "nope", "client-1"), apperror.CodeNotFound))
	_, err = svc.ListClientAPIs(t.Context(), "nope")
	require.True(t, apperror.IsCode(err, apperror.CodeNotFound))
	_, _, err = svc.ListAssignableClients(t.Context(), "nope", "", utils.ListRequestOptions{})
	require.True(t, apperror.IsCode(err, apperror.CodeNotFound))
	_, _, err = svc.ListAssignableAPIs(t.Context(), "nope", "", utils.ListRequestOptions{})
	require.True(t, apperror.IsCode(err, apperror.CodeNotFound))
}

func findPermission(api API, key string) *Permission {
	for i := range api.Permissions {
		if api.Permissions[i].Key == key {
			return &api.Permissions[i]
		}
	}
	return nil
}
