package instanceid

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/model"
	testutils "github.com/pocket-id/pocket-id/backend/internal/utils/testing"
)

func TestLoad(t *testing.T) {
	// readInstanceID returns the value stored in the kv table under the "instance_id" key, together with the number of rows that match (to detect duplicates)
	readInstanceID := func(t *testing.T, db *gorm.DB) (value string, count int) {
		t.Helper()
		var rows []model.KV
		err := db.Where("key = ?", "instance_id").Find(&rows).Error
		require.NoError(t, err)
		if len(rows) == 0 {
			return "", 0
		}
		require.NotNil(t, rows[0].Value, "instance_id value should not be NULL")
		return *rows[0].Value, len(rows)
	}

	t.Run("generates and persists a new instance ID when none exists", func(t *testing.T) {
		db := testutils.NewDatabaseForTest(t)

		id, err := Load(t.Context(), db)
		require.NoError(t, err)
		require.NotEmpty(t, id)

		// The generated ID should be a valid UUID
		_, err = uuid.Parse(id)
		require.NoError(t, err, "generated instance ID should be a valid UUID")

		// The value should have been persisted in the kv table
		stored, count := readInstanceID(t, db)
		require.Equal(t, 1, count)
		require.Equal(t, id, stored)
	})

	t.Run("returns the existing instance ID on subsequent calls", func(t *testing.T) {
		db := testutils.NewDatabaseForTest(t)

		first, err := Load(t.Context(), db)
		require.NoError(t, err)
		require.NotEmpty(t, first)

		// A second call must return the same ID and must not create a duplicate row
		second, err := Load(t.Context(), db)
		require.NoError(t, err)
		require.Equal(t, first, second)

		stored, count := readInstanceID(t, db)
		require.Equal(t, 1, count)
		require.Equal(t, first, stored)
	})

	t.Run("returns an instance ID that was already stored", func(t *testing.T) {
		db := testutils.NewDatabaseForTest(t)

		// Seed an instance ID directly in the database, e.g. as written by the migration that moved it out of the app config table
		existing := "existing-instance-id"
		err := db.Create(&model.KV{Key: "instance_id", Value: &existing}).Error
		require.NoError(t, err)

		id, err := Load(t.Context(), db)
		require.NoError(t, err)
		require.Equal(t, existing, id)

		stored, count := readInstanceID(t, db)
		require.Equal(t, 1, count)
		require.Equal(t, existing, stored)
	})
}
