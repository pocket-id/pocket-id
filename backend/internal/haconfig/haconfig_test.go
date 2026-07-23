package haconfig

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/model"
	testutils "github.com/pocket-id/pocket-id/backend/internal/utils/testing"
)

// readHAEnabled returns the value stored in the kv table under the "ha_enabled" key, together with the number of rows that match (to detect duplicates)
func readHAEnabled(t *testing.T, db *gorm.DB) (value string, count int) {
	t.Helper()
	var rows []model.KV
	err := db.Where("key = ?", KVKey).Find(&rows).Error
	require.NoError(t, err)
	if len(rows) == 0 {
		return "", 0
	}
	require.NotNil(t, rows[0].Value, "ha_enabled value should not be NULL")
	return *rows[0].Value, len(rows)
}

func TestCheck(t *testing.T) {
	t.Run("records the value on first startup", func(t *testing.T) {
		for _, haEnabled := range []bool{false, true} {
			t.Run("HAEnabled="+boolStr(haEnabled), func(t *testing.T) {
				db := testutils.NewDatabaseForTest(t)

				err := Check(t.Context(), db, haEnabled)
				require.NoError(t, err)

				stored, count := readHAEnabled(t, db)
				require.Equal(t, 1, count)
				require.Equal(t, boolStr(haEnabled), stored)
			})
		}
	})

	t.Run("succeeds when the value is unchanged on a later startup", func(t *testing.T) {
		db := testutils.NewDatabaseForTest(t)

		require.NoError(t, Check(t.Context(), db, true))

		// A second startup with the same value must succeed and not create a duplicate row
		require.NoError(t, Check(t.Context(), db, true))

		stored, count := readHAEnabled(t, db)
		require.Equal(t, 1, count)
		require.Equal(t, "true", stored)
	})

	t.Run("fails when the value changed, without overwriting the stored value", func(t *testing.T) {
		db := testutils.NewDatabaseForTest(t)

		// The cluster was created with HA disabled
		require.NoError(t, Check(t.Context(), db, false))

		// Starting up with HA enabled must be rejected
		err := Check(t.Context(), db, true)
		require.Error(t, err)
		require.Contains(t, err.Error(), "cannot be changed")

		// The stored value must not have been changed by the rejected startup
		stored, count := readHAEnabled(t, db)
		require.Equal(t, 1, count)
		require.Equal(t, "false", stored)

		// Starting up again with the original value still works
		require.NoError(t, Check(t.Context(), db, false))
	})

	t.Run("concurrent first startups converge on a single value", func(t *testing.T) {
		db := testutils.NewConcurrentDatabaseForTest(t)

		const parallel = 25
		errs := make([]error, parallel)

		start := make(chan struct{})
		var wg sync.WaitGroup
		wg.Add(parallel)
		for i := range parallel {
			go func() {
				defer wg.Done()
				<-start
				errs[i] = Check(t.Context(), db, true)
			}()
		}
		close(start)
		wg.Wait()

		for i, err := range errs {
			require.NoErrorf(t, err, "goroutine %d errored", i)
		}

		// Only a single row must have been persisted
		stored, count := readHAEnabled(t, db)
		require.Equal(t, 1, count)
		require.Equal(t, "true", stored)
	})
}

func boolStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}
