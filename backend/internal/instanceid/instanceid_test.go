package instanceid

import (
	"sync"
	"testing"
	"uuid"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/model"
	testutils "github.com/pocket-id/pocket-id/backend/internal/utils/testing"
)

// readInstanceID returns the value stored in the kv table under the "instance_id" key, together with the number of rows that match (to detect duplicates)
func readInstanceID(t *testing.T, db *gorm.DB) (value string, count int) {
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

func TestLoad(t *testing.T) {
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

	t.Run("concurrent calls without an existing instance ID converge on a single value", func(t *testing.T) {
		// This needs a database that allows concurrent access, so the goroutines race
		db := testutils.NewConcurrentDatabaseForTest(t)

		const parallel = 25

		// Each goroutine writes to its own index, so no locking is needed to collect the results
		ids := make([]string, parallel)
		loadErrs := make([]error, parallel)

		// The start channel is a barrier so all goroutines race their Load call at once
		start := make(chan struct{})

		var wg sync.WaitGroup
		wg.Add(parallel)
		for i := range parallel {
			go func() {
				defer wg.Done()
				<-start
				ids[i], loadErrs[i] = Load(t.Context(), db)
			}()
		}
		close(start)
		wg.Wait()

		// No concurrent call should have failed
		for i, err := range loadErrs {
			require.NoErrorf(t, err, "goroutine %d errored", i)
		}

		// Every goroutine must have observed the same, non-empty instance ID, meaning only one of them actually generated and persisted a value
		first := ids[0]
		require.NotEmpty(t, first)
		for _, id := range ids {
			require.Equal(t, first, id)
		}

		// Only a single row must have been persisted, and it must match the shared value
		stored, count := readInstanceID(t, db)
		require.Equal(t, 1, count)
		require.Equal(t, first, stored)
	})
}
