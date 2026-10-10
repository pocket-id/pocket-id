package profilepicture

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestDefaultPictureCacheKeyPinsVersion pins the cache key format the cleanup job matches against
// Changing the key format silently orphans every existing cache file, so the shape is part of the contract
func TestDefaultPictureCacheKeyPinsVersion(t *testing.T) {
	assert.Equal(t, "JD--v2", DefaultPictureCacheKey("JD"))
	assert.Equal(t, "漢字--v2", DefaultPictureCacheKey("漢字"))
	assert.Equal(t, "profile-pictures/defaults/JD--v2.png", DefaultPictureCachePath("JD"))
	assert.Equal(t, "profile-pictures/defaults/漢字--v2.png", DefaultPictureCachePath("漢字"))
}

// TestIsOutdatedDefaultPictureCacheFile covers the version-aware matching the cleanup job relies on
func TestIsOutdatedDefaultPictureCacheFile(t *testing.T) {
	// The current layout's file is current
	assert.False(t, IsOutdatedDefaultPictureCacheFile("JD--v2.png", "JD"))
	assert.False(t, IsOutdatedDefaultPictureCacheFile("profile-pictures/defaults/JD--v2.png", "JD"))

	// Files without a version marker predate the versioned layout and count as outdated
	assert.True(t, IsOutdatedDefaultPictureCacheFile("JD.png", "JD"))

	// A different user's initials are outdated for this user
	assert.True(t, IsOutdatedDefaultPictureCacheFile("AB--v2.png", "JD"))

	// A previous version is outdated until the file is regenerated
	assert.True(t, IsOutdatedDefaultPictureCacheFile("JD--v1.png", "JD"))
}
