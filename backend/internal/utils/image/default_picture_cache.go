package profilepicture

import (
	"path"
	"strings"
)

// DefaultPictureCacheVersion identifies the current default-profile-picture layout
// A layout change that alters how pictures look, e.g. the font fallback for CJK initials, must bump this version
// Bumping the version makes every existing cache entry miss and the cleanup job remove the stale files
const DefaultPictureCacheVersion = "v2"

// DefaultPictureCacheDir is the directory the generated default pictures are cached in
const DefaultPictureCacheDir = "profile-pictures/defaults"

// defaultPictureVersionSeparator joins the initials and the version in a cache file name
const defaultPictureVersionSeparator = "--"

// DefaultPictureCacheKey builds the cache key for the given initials under the current layout
// The key is the file name without the .png extension and is the unit the cleanup job matches against
func DefaultPictureCacheKey(initials string) string {
	return initials + defaultPictureVersionSeparator + DefaultPictureCacheVersion
}

// DefaultPictureCachePath builds the full storage path of the cached picture for the given initials
func DefaultPictureCachePath(initials string) string {
	return path.Join(DefaultPictureCacheDir, DefaultPictureCacheKey(initials)+".png")
}

// IsOutdatedDefaultPictureCacheFile reports whether a file name from the cache directory belongs to the current layout
// File names without a version separator predate the versioned layout and always count as outdated
func IsOutdatedDefaultPictureCacheFile(filename, initials string) bool {
	_, name := path.Split(filename)
	name = strings.TrimSuffix(name, ".png")
	return name != DefaultPictureCacheKey(initials)
}
