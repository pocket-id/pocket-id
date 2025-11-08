package storage

import (
	"bytes"
	"context"
	"errors"
	"io"
	"path/filepath"
	"testing"

	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewFileStorage_BackendValidation(t *testing.T) {
	t.Run("filesystem backend requires root", func(t *testing.T) {
		_, err := NewFileStorage(context.Background(), Config{
			Backend: "fs",
		})
		require.Error(t, err)
		assert.Equal(t, "filesystem storage requires a root path", err.Error())
	})

	t.Run("unsupported backend returns error", func(t *testing.T) {
		_, err := NewFileStorage(context.Background(), Config{
			Backend: "unsupported",
			Root:    t.TempDir(),
		})
		require.Error(t, err)
		assert.Equal(t, "unsupported file backend 'unsupported'", err.Error())
	})

	t.Run("s3 backend requires bucket and region", func(t *testing.T) {
		_, err := NewFileStorage(context.Background(), Config{
			Backend:  "s3",
			S3Region: "us-east-1",
		})
		require.Error(t, err)
		assert.Equal(t, "s3 storage requires both bucket and region", err.Error())

		_, err = NewFileStorage(context.Background(), Config{
			Backend:  "s3",
			S3Bucket: "bucket",
		})
		require.Error(t, err)
		assert.Equal(t, "s3 storage requires both bucket and region", err.Error())
	})
}

func TestFilesystemStorageOperations(t *testing.T) {
	ctx := context.Background()
	store, err := NewFileStorage(ctx, Config{Backend: "fs", Root: t.TempDir()})
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

func TestS3Helpers(t *testing.T) {
	t.Run("buildObjectKey trims and joins prefix", func(t *testing.T) {
		tests := []struct {
			name     string
			prefix   string
			path     string
			expected string
		}{
			{name: "no prefix no path", prefix: "", path: "", expected: ""},
			{name: "prefix no path", prefix: "root", path: "", expected: "root"},
			{name: "prefix with nested path", prefix: "root", path: "foo/bar/baz", expected: "root/foo/bar/baz"},
			{name: "trimmed path and prefix", prefix: "root", path: "/foo//bar/", expected: "root/foo/bar"},
			{name: "no prefix path only", prefix: "", path: "./images/logo.png", expected: "images/logo.png"},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				s := &s3Storage{
					bucket: "bucket",
					prefix: tc.prefix,
				}
				assert.Equal(t, tc.expected, s.buildObjectKey(tc.path))
			})
		}
	})

	t.Run("isS3NotFound detects expected errors", func(t *testing.T) {
		assert.True(t, isS3NotFound(&smithy.GenericAPIError{Code: "NoSuchKey"}))
		assert.True(t, isS3NotFound(&smithy.GenericAPIError{Code: "NotFound"}))
		assert.True(t, isS3NotFound(&s3types.NoSuchKey{}))
		assert.False(t, isS3NotFound(errors.New("boom")))
	})
}
