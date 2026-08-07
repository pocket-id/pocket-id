//go:build unit

package service

import (
	"strings"
	"testing"

	"github.com/ory/fosite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/appconfig"
	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	testutils "github.com/pocket-id/pocket-id/backend/internal/utils/testing"
)

// TestUpdateDynamicClientDoesNotClobberAdminFields guards the RFC 7592 update path
// against writing back stale copies of columns the client does not own. The update
// must persist only client-managed metadata, so an administrator change that lands
// while the client is updating itself survives.
func TestUpdateDynamicClientDoesNotClobberAdminFields(t *testing.T) {
	original := common.EnvConfig
	t.Cleanup(func() { common.EnvConfig = original })
	common.EnvConfig.UiConfigDisabled = true

	db := testutils.NewDatabaseForTest(t)
	cfg := appconfig.NewTestConfig(nil)
	cfg.DynamicClientRedirectUriAllowlist = appconfig.AppConfigValue(`["https://app.example.com/**"]`)

	svc, err := NewOidcService(db, nil, nil, nil, nil, nil, nil, nil)
	require.NoError(t, err)
	svc.appConfigService = appconfig.NewTestAppConfigService(cfg)

	client, _, regToken, err := svc.RegisterDynamicClient(t.Context(), fosite.ClientRegistrationRequest{
		RedirectURIs:            []string{"https://app.example.com/cb"},
		ClientName:              "Dynamic Client",
		TokenEndpointAuthMethod: "client_secret_basic",
	})
	require.NoError(t, err)

	// Simulate an admin tightening security-relevant columns the client must not own.
	// These are written directly, mirroring a concurrent admin action that the client's
	// in-memory copy has not observed.
	require.NoError(t, db.Model(&model.OidcClient{}).Where("id = ?", client.ID).
		Updates(map[string]any{
			"is_group_restricted": true,
			"skip_consent":        false,
			"description":         "managed by admin",
		}).Error)

	// The client now updates its own metadata via RFC 7592.
	updated, _, _, err := svc.UpdateDynamicClient(t.Context(), client.ID, regToken, fosite.ClientRegistrationRequest{
		RedirectURIs:            []string{"https://app.example.com/other"},
		ClientName:              "Renamed",
		TokenEndpointAuthMethod: "client_secret_basic",
	})
	require.NoError(t, err)
	assert.Equal(t, "Renamed", updated.Name)

	var stored model.OidcClient
	require.NoError(t, db.First(&stored, "id = ?", client.ID).Error)

	// Client-owned metadata is applied
	assert.Equal(t, "Renamed", stored.Name)
	assert.Equal(t, []string{"https://app.example.com/other"}, []string(stored.CallbackURLs))

	// Admin-owned columns are untouched
	assert.True(t, stored.IsGroupRestricted, "admin group restriction must survive a client self-update")
	assert.Equal(t, "managed by admin", stored.Description, "admin description must survive a client self-update")
	assert.Equal(t, model.OidcClientTypeDynamic, stored.ClientType)
}

// TestUpdateDynamicClientWritesOnlyClientOwnedColumns is the precise regression
// guard. It calls the update with a client row whose in-memory copy carries stale
// values for admin-owned columns, which is exactly the state a concurrent admin
// write produces. A full-model Save() writes those stale values back; persisting
// only client-owned columns cannot.
func TestUpdateDynamicClientWritesOnlyClientOwnedColumns(t *testing.T) {
	original := common.EnvConfig
	t.Cleanup(func() { common.EnvConfig = original })
	common.EnvConfig.UiConfigDisabled = true

	db := testutils.NewDatabaseForTest(t)
	cfg := appconfig.NewTestConfig(nil)
	cfg.DynamicClientRedirectUriAllowlist = appconfig.AppConfigValue(`["https://app.example.com/**"]`)

	svc, err := NewOidcService(db, nil, nil, nil, nil, nil, nil, nil)
	require.NoError(t, err)
	svc.appConfigService = appconfig.NewTestAppConfigService(cfg)

	client, _, regToken, err := svc.RegisterDynamicClient(t.Context(), fosite.ClientRegistrationRequest{
		RedirectURIs:            []string{"https://app.example.com/cb"},
		ClientName:              "Dynamic Client",
		TokenEndpointAuthMethod: "client_secret_basic",
	})
	require.NoError(t, err)
	require.False(t, client.IsGroupRestricted)

	// Determine which columns the update statement actually touches. This is the
	// property under test: the write must be scoped to client-owned metadata rather
	// than the whole model, so no admin-owned column can be carried along.
	// Every statement is captured, not just the last one: the update also rotates the
	// registration access token in a separate write, and the metadata statement is the
	// one whose column scoping matters.
	var updateStatements []string
	const hookName = "test:capture_update_sql"
	require.NoError(t, db.Callback().Update().After("gorm:update").Register(hookName, func(tx *gorm.DB) {
		if tx.Statement != nil && tx.Statement.Table == "oidc_clients" {
			updateStatements = append(updateStatements, tx.Statement.SQL.String())
		}
	}))
	t.Cleanup(func() { _ = db.Callback().Update().Remove(hookName) })

	_, _, _, err = svc.UpdateDynamicClient(t.Context(), client.ID, regToken, fosite.ClientRegistrationRequest{
		RedirectURIs:            []string{"https://app.example.com/other"},
		ClientName:              "Renamed",
		TokenEndpointAuthMethod: "client_secret_basic",
	})
	require.NoError(t, err)
	require.NotEmpty(t, updateStatements, "expected an UPDATE against oidc_clients")

	// The metadata write is the statement carrying the client-owned columns.
	var updateSQL string
	for _, stmt := range updateStatements {
		if strings.Contains(stmt, "name") {
			updateSQL = stmt
			break
		}
	}
	require.NotEmpty(t, updateSQL, "expected an UPDATE writing client metadata, got %v", updateStatements)

	// Client-owned columns are written
	for _, col := range []string{"name", "callback_urls", "is_public", "pkce_enabled", "secret"} {
		assert.Contains(t, updateSQL, col, "client-owned column %q should be written", col)
	}

	// Admin-owned columns must never appear in a client-initiated update
	for _, col := range []string{
		"is_group_restricted",
		"skip_consent",
		"client_type",
		"requires_reauthentication",
		"requires_pushed_authorization_requests",
		"registration_access_token_hash",
		"description",
		"launch_url",
	} {
		assert.NotContains(t, updateSQL, col, "admin-owned column %q must not be written by a client self-update", col)
	}

	// The token rotation is a separate, narrowly scoped statement of its own.
	var rotationSQL string
	for _, stmt := range updateStatements {
		if strings.Contains(stmt, "registration_access_token_hash") {
			rotationSQL = stmt
			break
		}
	}
	require.NotEmpty(t, rotationSQL, "the update must rotate the registration access token")
	for _, col := range []string{"name", "callback_urls", "is_public", "secret"} {
		assert.NotContains(t, rotationSQL, col, "the rotation write must touch only the token hash, not %q", col)
	}

	var stored model.OidcClient
	require.NoError(t, db.First(&stored, "id = ?", client.ID).Error)
	assert.Equal(t, "Renamed", stored.Name, "client-owned metadata must still be applied")
}
