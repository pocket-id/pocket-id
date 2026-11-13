package appassets

import (
	"io"
	"io/fs"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pocket-id/pocket-id/backend/internal/storage"
)

func TestInitApplicationAssetsMovesLegacyUploads(t *testing.T) {
	store, err := storage.NewFilesystemStorage(t.TempDir())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, store.Close()) })

	require.NoError(t, store.Save(t.Context(), "application-images/logoLight.png", strings.NewReader("custom")))
	require.NoError(t, store.Save(t.Context(), "application-images/.deleted/background", strings.NewReader("")))

	extensions, err := initApplicationAssets(t.Context(), store)
	require.NoError(t, err)
	assert.Equal(t, "png", extensions["logoLight"])
	assert.NotContains(t, extensions, "background", "the deleted marker must survive the move")

	reader, _, err := store.Open(t.Context(), "application-assets/logoLight.png")
	require.NoError(t, err)
	payload, err := io.ReadAll(reader)
	require.NoError(t, reader.Close())
	require.NoError(t, err)
	assert.Equal(t, "custom", string(payload))

	_, err = store.List(t.Context(), "application-images")
	require.ErrorIs(t, err, fs.ErrNotExist)
}
