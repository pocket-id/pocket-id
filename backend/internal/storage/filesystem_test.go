package storage

import (
	"bytes"
	"context"
	"io"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFilesystemStorageOperations(t *testing.T) {
	ctx := context.Background()
	store, err := NewFilesystemStorage(t.TempDir())
	require.NoError(t, err)

	t.Run("save, open and list files", func(t *testing.T) {
		err := store.Save(ctx, "images/logo.png", bytes.NewBufferString("logo-data"))
		require.NoError(t, err)

		reader, size, err := store.Open(ctx, "images/logo.png")
		require.NoError(t, err)
		defer reader.Close()

		contents, err := io.ReadAll(reader)
		require.NoError(t, err)
		assert.Equal(t, []byte("logo-data"), contents)
		assert.Equal(t, int64(len(contents)), size)

		err = store.Save(ctx, "images/nested/child.txt", bytes.NewBufferString("child"))
		require.NoError(t, err)

		files, err := store.List(ctx, "images")
		require.NoError(t, err)
		require.Len(t, files, 1)
		assert.Equal(t, filepath.Join("images", "logo.png"), files[0].Path)
		assert.Equal(t, int64(len("logo-data")), files[0].Size)
	})

	t.Run("delete files individually and idempotently", func(t *testing.T) {
		err := store.Save(ctx, "images/delete-me.txt", bytes.NewBufferString("temp"))
		require.NoError(t, err)

		require.NoError(t, store.Delete(ctx, "images/delete-me.txt"))
		_, _, err = store.Open(ctx, "images/delete-me.txt")
		require.Error(t, err)
		assert.True(t, IsNotExist(err))

		// Deleting a missing object should be a no-op.
		require.NoError(t, store.Delete(ctx, "images/missing.txt"))
	})

	t.Run("delete all files under a prefix", func(t *testing.T) {
		require.NoError(t, store.Save(ctx, "images/a.txt", bytes.NewBufferString("a")))
		require.NoError(t, store.Save(ctx, "images/b.txt", bytes.NewBufferString("b")))
		require.NoError(t, store.DeleteAll(ctx, "images"))

		_, _, err := store.Open(ctx, "images/a.txt")
		require.Error(t, err)
		assert.True(t, IsNotExist(err))

		_, _, err = store.Open(ctx, "images/b.txt")
		require.Error(t, err)
		assert.True(t, IsNotExist(err))
	})
}
