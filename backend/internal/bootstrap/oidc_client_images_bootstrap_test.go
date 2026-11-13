package bootstrap

import (
	"io"
	"io/fs"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pocket-id/pocket-id/backend/internal/model"
	"github.com/pocket-id/pocket-id/backend/internal/oidc"
	"github.com/pocket-id/pocket-id/backend/internal/storage"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
	testutils "github.com/pocket-id/pocket-id/backend/internal/utils/testing"
)

func TestMoveLegacyOidcClientImages(t *testing.T) {
	const cimdClientID = "https://app.example.com/oauth/client"

	db := testutils.NewDatabaseForTest(t)
	store, err := storage.NewFilesystemStorage(t.TempDir())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, store.Close()) })

	require.NoError(t, db.Create(&[]model.OidcClient{
		{Base: model.Base{ID: "plain"}, Name: "Plain", ImageType: new("png"), DarkImageType: new("svg")},
		{Base: model.Base{ID: cimdClientID}, Name: "CIMD", ImageType: new("webp")},
		{Base: model.Base{ID: "missing"}, Name: "Missing", ImageType: new("png")},
	}).Error)
	require.NoError(t, store.Save(t.Context(), "oidc-client-images/plain.png", strings.NewReader("light")))
	require.NoError(t, store.Save(t.Context(), "oidc-client-images/plain-dark.svg", strings.NewReader("dark")))
	require.NoError(t, store.Save(t.Context(), "oidc-client-images/cimd-"+utils.CreateSha256Hash(cimdClientID)+".webp", strings.NewReader("cimd")))

	require.NoError(t, moveLegacyOidcClientImages(t.Context(), db, store))

	assertStored := func(path string, expected string) {
		t.Helper()
		reader, _, err := store.Open(t.Context(), path)
		require.NoError(t, err)
		payload, err := io.ReadAll(reader)
		require.NoError(t, reader.Close())
		require.NoError(t, err)
		assert.Equal(t, expected, string(payload))
	}
	assertStored(oidc.ClientImagePath("plain", "", "png"), "light")
	assertStored(oidc.ClientImagePath("plain", "-dark", "svg"), "dark")
	assertStored(oidc.ClientImagePath(cimdClientID, "", "webp"), "cimd")

	files, err := store.List(t.Context(), "oidc-client-images")
	require.NoError(t, err)
	assert.Len(t, files, 3, "the legacy files must be deleted")

	require.NoError(t, moveLegacyOidcClientImages(t.Context(), db, store), "a second run must be a no-op")
	_, _, err = store.Open(t.Context(), "oidc-client-images/plain.png")
	require.ErrorIs(t, err, fs.ErrNotExist)
}
