package common

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadEnvVarFromFile(t *testing.T) {
	t.Run("returns false when the file env var is not set", func(t *testing.T) {
		value, ok, err := LoadEnvVarFromFile("TEST_SECRET")

		require.NoError(t, err)
		assert.False(t, ok)
		assert.Empty(t, value)
	})

	t.Run("returns the content of the file as-is", func(t *testing.T) {
		fileName := writeTempFile(t, "secret.txt", "  my-secret\n")
		t.Setenv("TEST_SECRET_FILE", fileName)

		value, ok, err := LoadEnvVarFromFile("TEST_SECRET")

		require.NoError(t, err)
		assert.True(t, ok)
		assert.Equal(t, []byte("  my-secret\n"), value)
	})

	t.Run("returns an error when the file cannot be read", func(t *testing.T) {
		t.Setenv("TEST_SECRET_FILE", filepath.Join(t.TempDir(), "does-not-exist.txt"))

		_, ok, err := LoadEnvVarFromFile("TEST_SECRET")

		require.Error(t, err)
		require.ErrorContains(t, err, "TEST_SECRET_FILE")
		assert.False(t, ok)
	})
}

func TestLoadStringEnvVarFromFile(t *testing.T) {
	t.Run("returns false when the file env var is not set", func(t *testing.T) {
		value, ok, err := LoadStringEnvVarFromFile("TEST_SECRET")

		require.NoError(t, err)
		assert.False(t, ok)
		assert.Empty(t, value)
	})

	t.Run("trims leading and trailing whitespace", func(t *testing.T) {
		fileName := writeTempFile(t, "secret.txt", "\tmy-secret \r\n")
		t.Setenv("TEST_SECRET_FILE", fileName)

		value, ok, err := LoadStringEnvVarFromFile("TEST_SECRET")

		require.NoError(t, err)
		assert.True(t, ok)
		assert.Equal(t, "my-secret", value)
	})

	t.Run("returns an empty value when the file is empty", func(t *testing.T) {
		fileName := writeTempFile(t, "empty.txt", "\n")
		t.Setenv("TEST_SECRET_FILE", fileName)

		value, ok, err := LoadStringEnvVarFromFile("TEST_SECRET")

		require.NoError(t, err)
		assert.True(t, ok)
		assert.Empty(t, value)
	})

	t.Run("returns an error when the file cannot be read", func(t *testing.T) {
		t.Setenv("TEST_SECRET_FILE", filepath.Join(t.TempDir(), "does-not-exist.txt"))

		_, ok, err := LoadStringEnvVarFromFile("TEST_SECRET")

		require.Error(t, err)
		assert.False(t, ok)
	})
}

// writeTempFile writes content to a file in a temporary directory that is removed when the test ends, returning its path
func writeTempFile(t *testing.T, name string, content string) string {
	t.Helper()

	fileName := filepath.Join(t.TempDir(), name)
	err := os.WriteFile(fileName, []byte(content), 0600)
	require.NoError(t, err)

	return fileName
}
