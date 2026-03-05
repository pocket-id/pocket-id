package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrLockUnavailable = errors.New("lock is already held by another process")
	ErrLockLost        = errors.New("lock ownership lost")
)

const (
	ttl           = 30 * time.Second
	renewInterval = 20 * time.Second
	renewRetries  = 3
	lockKey       = "application_lock"
)

type AppLockService struct {
	db        *gorm.DB
	lockID    string
	processID int64
	hostID    string
}

func NewAppLockService(db *gorm.DB) *AppLockService {
	host, err := os.Hostname()
	if err != nil || host == "" {
		host = "unknown-host"
	}

	return &AppLockService{
		db:        db,
		processID: int64(os.Getpid()),
		hostID:    host,
		lockID:    uuid.NewString(),
	}
}

type lockValue struct {
	ProcessID int64  `json:"process_id"`
	HostID    string `json:"host_id"`
	LockID    string `json:"lock_id"`
	ExpiresAt int64  `json:"expires_at"`
}

func (lv *lockValue) Marshal() (string, error) {
	data, err := json.Marshal(lv)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (lv *lockValue) Unmarshal(raw string) error {
	if raw == "" {
		return nil
	}
	return json.Unmarshal([]byte(raw), lv)
}

// Acquire obtains the lock. When force is true, the lock is stolen from any existing owner.
// If the lock is forcefully acquired, it blocks until the previous lock has expired.
func (s *AppLockService) Acquire(ctx context.Context, force bool) (waitUntil time.Time, err error) {
	tx := s.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return time.Time{}, fmt.Errorf("begin lock transaction: %w", tx.Error)
	}
	defer func() {
		tx.Rollback()
	}()

	var prevLockRaw string
	err = tx.
		WithContext(ctx).
		Model(&model.KV{}).
		Where("key = ?", lockKey).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Select("value").
		Scan(&prevLockRaw).
		Error
	if err != nil {
		return time.Time{}, fmt.Errorf("query existing lock: %w", err)
	}

	var prevLock lockValue
	if prevLockRaw != "" {
		if err := prevLock.Unmarshal(prevLockRaw); err != nil {
			return time.Time{}, fmt.Errorf("decode existing lock value: %w", err)
		}
	}

	now := time.Now()
	nowUnix := now.Unix()

	value := lockValue{
		ProcessID: s.processID,
		HostID:    s.hostID,
		LockID:    s.lockID,
		ExpiresAt: now.Add(ttl).Unix(),
	}
	raw, err := value.Marshal()
	if err != nil {
		return time.Time{}, fmt.Errorf("encode lock value: %w", err)
	}

	var query string
	switch s.db.Name() {
	case "sqlite":
		query = `
			INSERT INTO kv (key, value)
			VALUES (?, ?)
			ON CONFLICT(key) DO UPDATE SET 
				value = excluded.value
			WHERE (json_extract(kv.value, '$.expires_at') < ?) OR ?
		`
	case "postgres":
		query = `
			INSERT INTO kv (key, value)
			VALUES ($1, $2)
			ON CONFLICT(key) DO UPDATE SET 
				value = excluded.value
			WHERE ((kv.value::json->>'expires_at')::bigint < $3) OR ($4::boolean IS TRUE)
		`
	default:
		return time.Time{}, fmt.Errorf("unsupported database dialect: %s", s.db.Name())
	}

	res := tx.WithContext(ctx).Exec(query, lockKey, raw, nowUnix, force)
	if res.Error != nil {
		return time.Time{}, fmt.Errorf("lock acquisition failed: %w", res.Error)
	}

	if err := tx.Commit().Error; err != nil {
		return time.Time{}, fmt.Errorf("commit lock acquisition: %w", err)
	}

	// If there is a lock that is not expired and force is false, no rows will be affected
	if res.RowsAffected == 0 {
		return time.Time{}, ErrLockUnavailable
	}

	if force && prevLock.ExpiresAt > nowUnix && prevLock.LockID != s.lockID {
		waitUntil = time.Unix(prevLock.ExpiresAt, 0)
	}

	attrs := []any{
		slog.Int64("process_id", s.processID),
		slog.String("host_id", s.hostID),
	}
	if wait := time.Until(waitUntil); wait > 0 {
		attrs = append(attrs, slog.Duration("wait_before_proceeding", wait))
	}
	slog.Info("Acquired application lock", attrs...)

	return waitUntil, nil
}

// RunRenewal keeps renewing the lock until the context is canceled.
func (s *AppLockService) RunRenewal(ctx context.Context) error {
	ticker := time.NewTicker(renewInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			err := s.renew(ctx)
			if err != nil {
				return fmt.Errorf("renew lock: %w", err)
			}
		}
	}
}

// Release releases the lock if it is held by this process.
func (s *AppLockService) Release(ctx context.Context) error {
	db, err := s.db.DB()
	if err != nil {
		return fmt.Errorf("failed to get DB connection: %w", err)
	}

	var query string
	switch s.db.Name() {
	case "sqlite":
		query = `
DELETE FROM kv
WHERE key = ?
  AND json_extract(value, '$.lock_id') = ?
`
	case "postgres":
		query = `
DELETE FROM kv
WHERE key = $1
  AND value::json->>'lock_id' = $2
`
	default:
		return fmt.Errorf("unsupported database dialect: %s", s.db.Name())
	}

	opCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	res, err := db.ExecContext(opCtx, query, lockKey, s.lockID)
	if err != nil {
		return fmt.Errorf("release lock failed: %w", err)
	}

	count, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to count affected rows: %w", err)
	}

	if count == 0 {
		slog.Warn("Application lock not held by this process, cannot release",
			slog.Int64("process_id", s.processID),
			slog.String("host_id", s.hostID),
		)
	}

	slog.Info("Released application lock",
		slog.Int64("process_id", s.processID),
		slog.String("host_id", s.hostID),
	)
	return nil
}

// renew tries to renew the lock, retrying up to renewRetries times (sleeping 1s between attempts).
func (s *AppLockService) renew(ctx context.Context) error {
	db, err := s.db.DB()
	if err != nil {
		return fmt.Errorf("failed to get DB connection: %w", err)
	}

	var lastErr error
	for attempt := 1; attempt <= renewRetries; attempt++ {
		now := time.Now()
		nowUnix := now.Unix()
		expiresAt := now.Add(ttl).Unix()

		value := lockValue{
			LockID:    s.lockID,
			ProcessID: s.processID,
			HostID:    s.hostID,
			ExpiresAt: expiresAt,
		}
		raw, err := value.Marshal()
		if err != nil {
			return fmt.Errorf("encode lock value: %w", err)
		}

		var query string
		switch s.db.Name() {
		case "sqlite":
			query = `
UPDATE kv
SET value = ?
WHERE key = ?
  AND json_extract(value, '$.lock_id') = ?
  AND json_extract(value, '$.expires_at') > ?
`
		case "postgres":
			query = `
UPDATE kv
SET value = $1
WHERE key = $2
  AND value::json->>'lock_id' = $3
  AND ((value::json->>'expires_at')::bigint > $4)
`
		default:
			return fmt.Errorf("unsupported database dialect: %s", s.db.Name())
		}

		opCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
		res, err := db.ExecContext(opCtx, query, raw, lockKey, s.lockID, nowUnix)
		cancel()

		// Query succeeded, but may have updated 0 rows
		if err == nil {
			count, err := res.RowsAffected()
			if err != nil {
				return fmt.Errorf("failed to count affected rows: %w", err)
			}

			// If no rows were updated, we lost the lock
			if count == 0 {
				return ErrLockLost
			}

			// All good
			slog.Debug("Renewed application lock",
				slog.Int64("process_id", s.processID),
				slog.String("host_id", s.hostID),
				slog.Duration("duration", time.Since(now)),
			)
			return nil
		}

		// If we're here, we have an error that can be retried
		slog.Debug("Application lock renewal attempt failed",
			slog.Any("error", err),
			slog.Duration("duration", time.Since(now)),
		)
		lastErr = fmt.Errorf("lock renewal failed: %w", err)

		// Wait before next attempt or cancel if context is done
		if attempt < renewRetries {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(1 * time.Second):
			}
		}
	}

	return lastErr
}
