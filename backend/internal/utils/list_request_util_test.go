package utils

import (
	"testing"

	"github.com/libtnb/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestPaginateFilterAndSortSortsStringsCaseInsensitively(t *testing.T) {
	type record struct {
		ID   uint
		Name string `sortable:"case-insensitive"`
	}

	db, err := gorm.Open(sqlite.Open("file:"+CreateSha256Hash(t.Name())+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Exec("CREATE TABLE records (id INTEGER PRIMARY KEY, name TEXT NOT NULL)").Error)
	require.NoError(t, db.Create(&[]record{{Name: "alpha"}, {Name: "Charlie"}, {Name: "bravo"}}).Error)

	for _, test := range []struct {
		direction string
		expected  []string
	}{
		{direction: "asc", expected: []string{"alpha", "bravo", "Charlie"}},
		{direction: "desc", expected: []string{"Charlie", "bravo", "alpha"}},
	} {
		t.Run(test.direction, func(t *testing.T) {
			options := ListRequestOptions{}
			options.Sort.Column = "name"
			options.Sort.Direction = test.direction

			var records []record
			_, err := PaginateFilterAndSort(options, db.Model(&record{}), &records)
			require.NoError(t, err)
			require.Len(t, records, 3)
			require.Equal(t, test.expected, []string{records[0].Name, records[1].Name, records[2].Name})
		})
	}
}

func TestExtractModelMetadataRequiresCaseInsensitiveSortMode(t *testing.T) {
	type record struct {
		Name  string `sortable:"case-insensitive"`
		Event string `sortable:"true"`
	}

	meta := extractModelMetadata(&[]record{})
	require.True(t, meta["Name"].IsCaseInsensitive)
	require.False(t, meta["Event"].IsCaseInsensitive)
}
