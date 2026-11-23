package cmds

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pocket-id/pocket-id/backend/internal/bootstrap"
	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/storage"
)

func TestFailedExportKeepsPreviousArchive(t *testing.T) {
	originalEnvConfig := common.EnvConfig
	t.Cleanup(func() { common.EnvConfig = originalEnvConfig })

	dir := t.TempDir()
	common.EnvConfig.DbProvider = common.DbProviderSqlite
	common.EnvConfig.DbConnectionString = "file:" + filepath.Join(dir, "pocket-id.db")
	common.EnvConfig.FileBackend = storage.TypeDatabase
	common.EnvConfig.EncryptionKey = []byte("test-encryption-key-characters-long")

	// A timestamp in year 10000 can't be encoded as JSON, which fails the export
	db, _, err := bootstrap.NewDatabase(t.Context())
	require.NoError(t, err)
	require.NoError(t, db.Exec("INSERT INTO users (id, username, first_name, last_name, display_name, created_at) VALUES ('user', 'user', '', '', '', 253402304399)").Error)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	require.NoError(t, sqlDB.Close())

	path := filepath.Join(dir, "export.zip")
	require.NoError(t, os.WriteFile(path, []byte("previous archive"), 0o600))

	err = runExport(t.Context(), exportFlags{Path: path})
	require.Error(t, err)

	content, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, "previous archive", string(content))
}
