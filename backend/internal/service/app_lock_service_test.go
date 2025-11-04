package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/model"
	testutils "github.com/pocket-id/pocket-id/backend/internal/utils/testing"
)

func newTestAppLockService(t *testing.T, db *gorm.DB) *AppLockService {
	t.Helper()

	return &AppLockService{
		db:        db,
		processID: 1,
		hostID:    "test-host",
		lockID:    "a13c7673-c7ae-49f1-9112-2cd2d0d4b0c1",
	}
}

func insertLock(t *testing.T, db *gorm.DB, value lockValue) {
	t.Helper()

	raw, err := value.Marshal()
	require.NoError(t, err)

	err = db.Create(&model.KV{Key: lockKey, Value: &raw}).Error
	require.NoError(t, err)
}

func readLockValue(t *testing.T, db *gorm.DB) lockValue {
	t.Helper()

	var row model.KV
	err := db.Take(&row, "key = ?", lockKey).Error
	require.NoError(t, err)

	require.NotNil(t, row.Value)

	var value lockValue
	err = value.Unmarshal(*row.Value)
	require.NoError(t, err)

	return value
}

func TestAppLockServiceAcquire(t *testing.T) {
	t.Run("creates new lock when none exists", func(t *testing.T) {
		db := testutils.NewDatabaseForTest(t)
		service := newTestAppLockService(t, db)

		err := service.Acquire(context.Background(), false)
		require.NoError(t, err)

		stored := readLockValue(t, db)
		require.Equal(t, service.processID, stored.ProcessID)
		require.Equal(t, service.hostID, stored.HostID)
		require.Greater(t, stored.ExpiresAt, time.Now().Unix())
	})

	t.Run("returns ErrLockUnavailable when lock held by another process", func(t *testing.T) {
		db := testutils.NewDatabaseForTest(t)
		service := newTestAppLockService(t, db)

		existing := lockValue{
			ProcessID: 99,
			HostID:    "other-host",
			ExpiresAt: time.Now().Add(ttl).Unix(),
		}
		insertLock(t, db, existing)

		err := service.Acquire(context.Background(), false)
		require.ErrorIs(t, err, ErrLockUnavailable)

		current := readLockValue(t, db)
		require.Equal(t, existing, current)
	})

	t.Run("force acquisition steals lock", func(t *testing.T) {
		db := testutils.NewDatabaseForTest(t)
		service := newTestAppLockService(t, db)

		insertLock(t, db, lockValue{
			ProcessID: 99,
			HostID:    "other-host",
			ExpiresAt: time.Now().Unix(),
		})

		opCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		err := service.Acquire(opCtx, true)
		require.NoError(t, err)

		stored := readLockValue(t, db)
		require.Equal(t, service.processID, stored.ProcessID)
		require.Equal(t, service.hostID, stored.HostID)
		require.Greater(t, stored.ExpiresAt, time.Now().Unix())
	})
}

func TestAppLockServiceRelease(t *testing.T) {
	t.Run("removes owned lock", func(t *testing.T) {
		db := testutils.NewDatabaseForTest(t)
		service := newTestAppLockService(t, db)

		err := service.Acquire(context.Background(), false)
		require.NoError(t, err)

		err = service.Release(context.Background())
		require.NoError(t, err)

		var row model.KV
		err = db.Take(&row, "key = ?", lockKey).Error
		require.ErrorIs(t, err, gorm.ErrRecordNotFound)
	})

	t.Run("ignores lock held by another owner", func(t *testing.T) {
		db := testutils.NewDatabaseForTest(t)
		service := newTestAppLockService(t, db)

		existing := lockValue{
			ProcessID: 2,
			HostID:    "other-host",
			ExpiresAt: time.Now().Add(ttl).Unix(),
		}
		insertLock(t, db, existing)

		err := service.Release(context.Background())
		require.NoError(t, err)

		stored := readLockValue(t, db)
		require.Equal(t, existing, stored)
	})
}

func TestAppLockServiceRenew(t *testing.T) {
	t.Run("extends expiration when lock is still owned", func(t *testing.T) {
		db := testutils.NewDatabaseForTest(t)
		service := newTestAppLockService(t, db)

		err := service.Acquire(context.Background(), false)
		require.NoError(t, err)

		before := readLockValue(t, db)

		err = service.renew(context.Background())
		require.NoError(t, err)

		after := readLockValue(t, db)
		require.Equal(t, service.processID, after.ProcessID)
		require.Equal(t, service.hostID, after.HostID)
		require.GreaterOrEqual(t, after.ExpiresAt, before.ExpiresAt)
	})

	t.Run("returns ErrLockLost when lock is missing", func(t *testing.T) {
		db := testutils.NewDatabaseForTest(t)
		service := newTestAppLockService(t, db)

		err := service.renew(context.Background())
		require.ErrorIs(t, err, ErrLockLost)
	})

	t.Run("returns ErrLockLost when ownership changed", func(t *testing.T) {
		db := testutils.NewDatabaseForTest(t)
		service := newTestAppLockService(t, db)

		err := service.Acquire(context.Background(), false)
		require.NoError(t, err)

		// Simulate a different process taking the lock.
		newOwner := lockValue{
			ProcessID: 9,
			HostID:    "stolen-host",
			ExpiresAt: time.Now().Add(ttl).Unix(),
		}
		raw, marshalErr := newOwner.Marshal()
		require.NoError(t, marshalErr)
		updateErr := db.Model(&model.KV{}).
			Where("key = ?", lockKey).
			Update("value", raw).Error
		require.NoError(t, updateErr)

		err = service.renew(context.Background())
		require.ErrorIs(t, err, ErrLockLost)
	})
}
