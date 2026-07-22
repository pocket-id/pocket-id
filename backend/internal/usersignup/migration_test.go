package usersignup

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/utils"
	testutils "github.com/pocket-id/pocket-id/backend/internal/utils/testing"
)

// versionBeforeMoveTokens is the migration version right before the "actor tokens" migration.
const versionBeforeMoveTokens = 20260722120000

// seedSignupTokensForMigration seeds two signup tokens (one with a user group, one without) into the pre-migration schema.
func seedSignupTokensForMigration(t *testing.T, db *gorm.DB, createdAt, expiresAt time.Time) {
	t.Helper()

	// An unrelated, non-JSON kv entry, to ensure the freeze/restore queries don't choke on other kv keys
	err := db.Exec(
		`INSERT INTO kv ("key", "value") VALUES ('instance_id', ?)`,
		"not-json-instance-id",
	).Error
	require.NoError(t, err)

	// A user group referenced by one of the tokens
	err = db.Exec(
		`INSERT INTO user_groups (id, created_at, friendly_name, name) VALUES (?, ?, ?, ?)`,
		"grp-1", createdAt.Unix(), "Group One", "group-one",
	).Error
	require.NoError(t, err)

	// A token with a user group
	err = db.Exec(
		`INSERT INTO signup_tokens (id, created_at, token, expires_at, usage_limit, usage_count) VALUES (?, ?, ?, ?, ?, ?)`,
		"tok-1", createdAt.Unix(), "TOKENWITHGROUP01", expiresAt.Unix(), 3, 1,
	).Error
	require.NoError(t, err)
	err = db.Exec(
		`INSERT INTO signup_tokens_user_groups (signup_token_id, user_group_id) VALUES (?, ?)`,
		"tok-1", "grp-1",
	).Error
	require.NoError(t, err)

	// A token without user groups
	err = db.Exec(
		`INSERT INTO signup_tokens (id, created_at, token, expires_at, usage_limit, usage_count) VALUES (?, ?, ?, ?, ?, ?)`,
		"tok-2", createdAt.Unix(), "TOKENNOGROUP0002", expiresAt.Unix(), 1, 0,
	).Error
	require.NoError(t, err)
}

func TestLoadMigratedSignupTokens(t *testing.T) {
	createdAt := time.Now().Add(-time.Hour).Truncate(time.Second)
	expiresAt := time.Now().Add(24 * time.Hour).Truncate(time.Second)

	db := testutils.NewDatabaseForTestWithMigrationSeed(t, versionBeforeMoveTokens, func(t *testing.T, db *gorm.DB) {
		seedSignupTokensForMigration(t, db, createdAt, expiresAt)
	})

	// The migration must have dropped the signup_tokens tables
	ok := db.Migrator().HasTable("signup_tokens")
	require.False(t, ok, "signup_tokens table should have been dropped")
	ok = db.Migrator().HasTable("signup_tokens_user_groups")
	require.False(t, ok, "signup_tokens_user_groups table should have been dropped")

	tokens, err := loadMigratedSignupTokens(t.Context(), db)
	require.NoError(t, err)
	require.Len(t, tokens, 2)

	byID := make(map[string]storedSignupToken, len(tokens))
	for _, tok := range tokens {
		byID[tok.ID] = tok
	}

	tok1 := byID["tok-1"]
	require.Equal(t, "TOKENWITHGROUP01", tok1.Token)
	require.Equal(t, 3, tok1.UsageLimit)
	require.Equal(t, 1, tok1.UsageCount)
	require.Equal(t, []string{"grp-1"}, tok1.UserGroupIDs)
	require.Equal(t, expiresAt.Unix(), tok1.ExpiresAt.Unix())
	require.Equal(t, createdAt.Unix(), tok1.CreatedAt.Unix())

	tok2 := byID["tok-2"]
	require.Equal(t, "TOKENNOGROUP0002", tok2.Token)
	require.Equal(t, 1, tok2.UsageLimit)
	require.Equal(t, 0, tok2.UsageCount)
	require.Empty(t, tok2.UserGroupIDs)
}

// TestLoadMigratedSignupTokensEmpty verifies that when there were no signup tokens, nothing is frozen and nothing is loaded.
func TestLoadMigratedSignupTokensEmpty(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)

	tokens, err := loadMigratedSignupTokens(t.Context(), db)
	require.NoError(t, err)
	require.Empty(t, tokens)
}

// TestMoveTokensToActorStateDown verifies that rolling the migration back recreates the signup token tables and restores their contents from the frozen kv document.
func TestMoveTokensToActorStateDown(t *testing.T) {
	createdAt := time.Now().Add(-time.Hour).Truncate(time.Second)
	expiresAt := time.Now().Add(24 * time.Hour).Truncate(time.Second)

	db := testutils.NewDatabaseForTestWithMigrationSeed(t, versionBeforeMoveTokens, func(t *testing.T, db *gorm.DB) {
		seedSignupTokensForMigration(t, db, createdAt, expiresAt)
	})

	// The tables were frozen and dropped by the up migration
	ok := db.Migrator().HasTable("signup_tokens")
	require.False(t, ok)

	// Roll the migration back
	sqlDB, err := db.DB()
	require.NoError(t, err)
	m, cleanup, err := utils.GetEmbeddedMigrateInstance(t.Context(), sqlDB)
	require.NoError(t, err)
	defer cleanup()

	err = m.Migrate(versionBeforeMoveTokens)
	require.NoError(t, err)

	// The tables must have been recreated and repopulated from the frozen document
	ok = db.Migrator().HasTable("signup_tokens")
	require.True(t, ok)
	ok = db.Migrator().HasTable("signup_tokens_user_groups")
	require.True(t, ok)

	type row struct {
		ID         string
		Token      string
		UsageLimit int
		UsageCount int
	}
	var rows []row
	err = db.Raw(`SELECT id, token, usage_limit, usage_count FROM signup_tokens ORDER BY id`).Scan(&rows).Error
	require.NoError(t, err)
	require.Equal(t, []row{
		{ID: "tok-1", Token: "TOKENWITHGROUP01", UsageLimit: 3, UsageCount: 1},
		{ID: "tok-2", Token: "TOKENNOGROUP0002", UsageLimit: 1, UsageCount: 0},
	}, rows)

	var groupID string
	err = db.Raw(`SELECT user_group_id FROM signup_tokens_user_groups WHERE signup_token_id = ?`, "tok-1").Scan(&groupID).Error
	require.NoError(t, err)
	require.Equal(t, "grp-1", groupID)

	// The frozen document must have been removed from the kv table
	var kvCount int64
	err = db.Raw(`SELECT count(*) FROM kv WHERE "key" = ?`, signupTokensMigratedKey).Scan(&kvCount).Error
	require.NoError(t, err)
	require.Zero(t, kvCount)
}
