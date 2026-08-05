package bootstrap

import (
	"bytes"
	"encoding/hex"
	"path/filepath"
	"testing"

	"github.com/libtnb/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/common"
)

func TestNewActorsOptsGetPSKUsesStableValue(t *testing.T) {
	opts := NewActorsOpts{
		EnvConfig: &common.EnvConfigSchema{
			EncryptionKey: []byte("test-encryption-key"),
		},
		// Constant value for this test
		InstanceID: "ee05c3eb-8129-47a6-a1c7-849998b6f876",
	}

	expectedHex := "db09067fa194c3731bf77b6415a1c5d903f03d4557605ba3236b31f6eddfc8d7"
	expected, err := hex.DecodeString(expectedHex)
	require.NoError(t, err)

	actual, err := opts.getPSK()
	require.NoError(t, err)
	require.Equalf(t, expected, actual, "actual result: %s", actual)
}

// TestNewActorsBackupProvider covers the provider the export and import use to back up and restore the actor host's data.
// It builds the provider from the same options the actor host uses, so a mismatch between those options and the concrete provider would otherwise only surface at runtime, when an export or import is attempted.
func TestNewActorsBackupProvider(t *testing.T) {
	// Foreign keys must be enabled, which the provider validates on init and which the application enables on every connection
	dbPath := filepath.Join(t.TempDir(), "pocket-id.db")
	dsn := "file:" + dbPath + "?_txlock=immediate&_pragma=busy_timeout(2500)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	t.Cleanup(func() {
		sqlDB, dbErr := db.DB()
		if dbErr == nil {
			_ = sqlDB.Close()
		}
	})

	providerOpts, err := ActorsProviderOptions(db, nil)
	require.NoError(t, err)

	provider, err := NewActorsBackupProvider(t.Context(), providerOpts)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = provider.Close()
	})

	// Init applies the actor host's schema migrations, so an export also works against a database the actor host has never run against
	var tables int64
	err = db.Raw(`SELECT count(*) FROM sqlite_master WHERE type = 'table' AND name LIKE 'francis_%'`).Scan(&tables).Error
	require.NoError(t, err)
	require.Positive(t, tables, "the provider must create the actor host's own tables")

	// Even an empty cluster produces a valid backup stream, which an import can restore
	buf := &bytes.Buffer{}
	err = provider.Backup(t.Context(), buf)
	require.NoError(t, err)
	require.Contains(t, buf.String(), "francis-backup")

	err = provider.Restore(t.Context(), bytes.NewReader(buf.Bytes()))
	require.NoError(t, err)
}
