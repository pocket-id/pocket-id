package storage

import (
	"bytes"
	"context"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	testingutil "github.com/pocket-id/pocket-id/backend/internal/utils/testing"
)

func TestDatabaseStorageOperations(t *testing.T) {
	ctx := context.Background()
	db := testingutil.NewDatabaseForTest(t)
	store, err := NewDatabaseStorage(db)
	require.NoError(t, err)

	t.Run("type should be database", func(t *testing.T) {
		assert.Equal(t, TypeDatabase, store.Type())
	})

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
		assert.Equal(t, "images/logo.png", files[0].Path)
		assert.Equal(t, int64(len("logo-data")), files[0].Size)
	})

	t.Run("save should update existing file", func(t *testing.T) {
		err := store.Save(ctx, "test/update.txt", bytes.NewBufferString("original"))
		require.NoError(t, err)

		err = store.Save(ctx, "test/update.txt", bytes.NewBufferString("updated"))
		require.NoError(t, err)

		reader, size, err := store.Open(ctx, "test/update.txt")
		require.NoError(t, err)
		defer reader.Close()

		contents, err := io.ReadAll(reader)
		require.NoError(t, err)
		assert.Equal(t, []byte("updated"), contents)
		assert.Equal(t, int64(len("updated")), size)
	})

	t.Run("delete files individually", func(t *testing.T) {
		err := store.Save(ctx, "images/delete-me.txt", bytes.NewBufferString("temp"))
		require.NoError(t, err)

		require.NoError(t, store.Delete(ctx, "images/delete-me.txt"))
		_, _, err = store.Open(ctx, "images/delete-me.txt")
		require.Error(t, err)
		assert.True(t, IsNotExist(err))
	})

	t.Run("delete missing file should not error", func(t *testing.T) {
		require.NoError(t, store.Delete(ctx, "images/missing.txt"))
	})

	t.Run("delete all files under a prefix", func(t *testing.T) {
		require.NoError(t, store.Save(ctx, "cleanup/a.txt", bytes.NewBufferString("a")))
		require.NoError(t, store.Save(ctx, "cleanup/b.txt", bytes.NewBufferString("b")))
		require.NoError(t, store.Save(ctx, "cleanup/nested/c.txt", bytes.NewBufferString("c")))
		require.NoError(t, store.DeleteAll(ctx, "cleanup"))

		_, _, err := store.Open(ctx, "cleanup/a.txt")
		require.Error(t, err)
		assert.True(t, IsNotExist(err))

		_, _, err = store.Open(ctx, "cleanup/b.txt")
		require.Error(t, err)
		assert.True(t, IsNotExist(err))

		_, _, err = store.Open(ctx, "cleanup/nested/c.txt")
		require.Error(t, err)
		assert.True(t, IsNotExist(err))
	})

	t.Run("walk files", func(t *testing.T) {
		require.NoError(t, store.Save(ctx, "walk/file1.txt", bytes.NewBufferString("1")))
		require.NoError(t, store.Save(ctx, "walk/file2.txt", bytes.NewBufferString("2")))
		require.NoError(t, store.Save(ctx, "walk/nested/file3.txt", bytes.NewBufferString("3")))

		var paths []string
		err := store.Walk(ctx, "walk", func(info ObjectInfo) error {
			paths = append(paths, info.Path)
			return nil
		})
		require.NoError(t, err)
		assert.Len(t, paths, 3)
		assert.Contains(t, paths, "walk/file1.txt")
		assert.Contains(t, paths, "walk/file2.txt")
		assert.Contains(t, paths, "walk/nested/file3.txt")
	})
}

func TestNewDatabaseStorage(t *testing.T) {
	t.Run("should return error with nil database", func(t *testing.T) {
		_, err := NewDatabaseStorage(nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "database connection is required")
	})

	t.Run("should create storage with valid database", func(t *testing.T) {
		db := testingutil.NewDatabaseForTest(t)
		store, err := NewDatabaseStorage(db)
		require.NoError(t, err)
		assert.NotNil(t, store)
	})
}
