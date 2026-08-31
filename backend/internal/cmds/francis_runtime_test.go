package cmds

import (
	"archive/zip"
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pocket-id/pocket-id/backend/internal/service"
)

func TestEnsureNoActorsBackup(t *testing.T) {
	buildZip := func(t *testing.T, names ...string) *zip.Reader {
		t.Helper()

		buf := &bytes.Buffer{}
		zw := zip.NewWriter(buf)
		for _, name := range names {
			w, err := zw.Create(name)
			require.NoError(t, err)
			_, err = w.Write([]byte("payload"))
			require.NoError(t, err)
		}
		require.NoError(t, zw.Close())

		zr, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
		require.NoError(t, err)

		return zr
	}

	t.Run("accepts an archive without the actor data", func(t *testing.T) {
		err := ensureNoActorsBackup(buildZip(t, "database.json", "uploads/logo.png"))
		require.NoError(t, err)
	})

	t.Run("rejects an archive carrying the actor data", func(t *testing.T) {
		err := ensureNoActorsBackup(buildZip(t, "database.json", service.ActorsBackupFileName))
		require.Error(t, err)
		require.ErrorContains(t, err, service.ActorsBackupFileName)
	})
}
