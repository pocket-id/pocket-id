package bootstrap

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pocket-id/pocket-id/backend/internal/utils"
)

func TestGetBuiltInImageData(t *testing.T) {
	// Get the built-in image data map
	builtInImages := getBuiltInImageData()

	// Read the actual images directory from disk
	imagesDir := filepath.Join("..", "..", "resources", "images")
	actualFiles, err := os.ReadDir(imagesDir)
	require.NoError(t, err, "Failed to read images directory")

	// Create a map of actual files for comparison
	actualFilesMap := make(map[string]struct{})

	// Validate each actual file exists in the built-in data with correct hash
	for _, file := range actualFiles {
		fileName := file.Name()
		if file.IsDir() || strings.HasPrefix(fileName, ".") {
			continue
		}

		actualFilesMap[fileName] = struct{}{}

		// Check if the file exists in the built-in data
		builtInData, exists := builtInImages[fileName]
		assert.True(t, exists, "File %s exists in images directory but not in getBuiltInImageData map", fileName)

		if !exists {
			continue
		}

		filePath := filepath.Join(imagesDir, fileName)

		// Validate SHA256 hash
		actualHash, err := utils.CreateSha256FileHash(filePath)
		require.NoError(t, err, "Failed to compute hash for %s", fileName)
		assert.Equal(t, actualHash, builtInData.SHA256, "SHA256 hash mismatch for file %s", fileName)
	}

	// Ensure the built-in data doesn't have extra files that don't exist in the directory
	for fileName := range builtInImages {
		_, exists := actualFilesMap[fileName]
		assert.True(t, exists, "File %s exists in getBuiltInImageData map but not in images directory", fileName)
	}

	// Ensure we have at least some files (sanity check)
	assert.NotEmpty(t, actualFilesMap, "Images directory should contain at least one file")
	assert.Len(t, actualFilesMap, len(builtInImages), "Number of files in directory should match number in built-in data map")
}
