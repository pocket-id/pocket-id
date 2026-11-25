package users

import (
	"path"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pocket-id/pocket-id/backend/internal/model"
	"github.com/pocket-id/pocket-id/backend/internal/storage"
	testutils "github.com/pocket-id/pocket-id/backend/internal/utils/testing"
)

func TestClearUnusedDefaultProfilePictures(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	cacheStorage, err := storage.NewFilesystemStorage(t.TempDir())
	require.NoError(t, err)
	// Windows can't remove the temporary folder while the storage holds it open
	t.Cleanup(func() {
		require.NoError(t, cacheStorage.Close())
	})
	service := newService(Dependencies{DB: db, CacheStorage: cacheStorage})

	require.NoError(t, db.Create(&model.User{Username: "ada", FirstName: "Ada", LastName: "Lovelace"}).Error)

	// A cache folder that holds no default pictures yet has nothing to clean
	require.NoError(t, service.clearUnusedDefaultProfilePictures(t.Context()))

	// Cache the picture the user needs, one nobody needs, and one an earlier version rendered for the same initials
	for _, file := range []string{"AL-v3.png", "ZZ-v3.png", "AL.png"} {
		require.NoError(t, cacheStorage.Save(t.Context(), path.Join(defaultPicturesDir, file), strings.NewReader("img")))
	}

	require.NoError(t, service.clearUnusedDefaultProfilePictures(t.Context()))

	files, err := cacheStorage.List(t.Context(), defaultPicturesDir)
	require.NoError(t, err)
	require.Len(t, files, 1)
	assert.Equal(t, path.Join(defaultPicturesDir, "AL-v3.png"), files[0].Path)
}
