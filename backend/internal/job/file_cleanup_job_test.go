package job

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pocket-id/pocket-id/backend/internal/model"
	"github.com/pocket-id/pocket-id/backend/internal/storage"
	profilepicture "github.com/pocket-id/pocket-id/backend/internal/utils/image"
	testutils "github.com/pocket-id/pocket-id/backend/internal/utils/testing"
)

// TestClearUnusedDefaultProfilePicturesKeepsCurrentLayout pins that cache files matching a live user under the current layout survive the cleanup
func TestClearUnusedDefaultProfilePicturesKeepsCurrentLayout(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	fileStorage, err := storage.NewDatabaseStorage(db)
	require.NoError(t, err)

	user := model.User{Username: "jdoe", FirstName: "John", LastName: "Doe"}
	require.NoError(t, db.Create(&user).Error)

	current := profilepicture.DefaultPictureCachePath("JD")
	require.NoError(t, fileStorage.Save(context.Background(), current, strings.NewReader("current")))

	j := &FileCleanupJobs{db: db, fileStorage: fileStorage}
	require.NoError(t, j.clearUnusedDefaultProfilePictures(context.Background()))

	_, _, err = fileStorage.Open(context.Background(), current)
	require.NoError(t, err, "the current-layout file for live initials must survive")
}

// TestClearUnusedDefaultProfilePicturesRemovesRenamedUserFiles verifies that a renamed user's old cache is dropped
func TestClearUnusedDefaultProfilePicturesRemovesRenamedUserFiles(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	fileStorage, err := storage.NewDatabaseStorage(db)
	require.NoError(t, err)

	// Cache file for initials nobody uses anymore
	orphan := profilepicture.DefaultPictureCachePath("ZZ")
	require.NoError(t, fileStorage.Save(context.Background(), orphan, strings.NewReader("stale")))

	j := &FileCleanupJobs{db: db, fileStorage: fileStorage}
	require.NoError(t, j.clearUnusedDefaultProfilePictures(context.Background()))

	_, _, err = fileStorage.Open(context.Background(), orphan)
	require.Error(t, err, "the orphaned cache file must be removed")
}

// TestClearUnusedDefaultProfilePicturesRemovesPreVersionFiles pins the migration of the tofu-era layout
// A pre-version file has no version marker, so it counts as outdated even though the initials are still in use
func TestClearUnusedDefaultProfilePicturesRemovesPreVersionFiles(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	fileStorage, err := storage.NewDatabaseStorage(db)
	require.NoError(t, err)

	user := model.User{Username: "jdoe", FirstName: "John", LastName: "Doe"}
	require.NoError(t, db.Create(&user).Error)

	legacy := "profile-pictures/defaults/JD.png"
	require.NoError(t, fileStorage.Save(context.Background(), legacy, strings.NewReader("legacy")))

	j := &FileCleanupJobs{db: db, fileStorage: fileStorage}
	require.NoError(t, j.clearUnusedDefaultProfilePictures(context.Background()))

	_, _, err = fileStorage.Open(context.Background(), legacy)
	require.Error(t, err, "the pre-version file must be garbage-collected so it is never served again")
}
