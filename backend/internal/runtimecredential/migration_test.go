//go:build unit

package runtimecredential

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pocket-id/pocket-id/backend/internal/utils"
	testutils "github.com/pocket-id/pocket-id/backend/internal/utils/testing"
)

const versionBeforeRuntimeCredentials = 20260807120000
const versionWithFreeFormAgentIdentifier = 20260814050000

func TestRuntimeCredentialMigrationDownAndUp(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	require.True(t, db.Migrator().HasColumn("users", "is_agent"))
	require.False(t, db.Migrator().HasColumn("users", "agent_identifier"))
	require.True(t, db.Migrator().HasTable("runtime_credentials"))
	require.True(t, db.Migrator().HasTable("runtime_credential_challenges"))

	sqlDB, err := db.DB()
	require.NoError(t, err)
	migration, cleanup, err := utils.GetEmbeddedMigrateInstance(t.Context(), sqlDB)
	require.NoError(t, err)
	defer cleanup()

	require.NoError(t, migration.Migrate(versionBeforeRuntimeCredentials))
	require.False(t, db.Migrator().HasColumn("users", "is_agent"))
	require.False(t, db.Migrator().HasColumn("users", "agent_identifier"))
	require.False(t, db.Migrator().HasTable("runtime_credentials"))
	require.False(t, db.Migrator().HasTable("runtime_credential_challenges"))

	require.NoError(t, migration.Up())
	require.True(t, db.Migrator().HasColumn("users", "is_agent"))
	require.False(t, db.Migrator().HasColumn("users", "agent_identifier"))
	require.True(t, db.Migrator().HasTable("runtime_credentials"))
	require.True(t, db.Migrator().HasTable("runtime_credential_challenges"))
}

func TestBinaryAgentSelectorMigrationPreservesEnabledPath(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	migration, cleanup, err := utils.GetEmbeddedMigrateInstance(t.Context(), sqlDB)
	require.NoError(t, err)
	defer cleanup()

	require.NoError(t, migration.Migrate(versionWithFreeFormAgentIdentifier))
	require.NoError(t, db.Exec(`INSERT INTO users (id, created_at, username, email_verified, first_name, last_name, display_name, is_admin, disabled, agent_identifier) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, "legacy-runtime-user", 1, "legacy-runtime", false, "Legacy", "Runtime", "Legacy Runtime", false, false, "legacy-runtime").Error)

	require.NoError(t, migration.Up())
	var isAgent bool
	require.NoError(t, db.Raw(`SELECT is_agent FROM users WHERE id = ?`, "legacy-runtime-user").Scan(&isAgent).Error)
	require.True(t, isAgent)
}
