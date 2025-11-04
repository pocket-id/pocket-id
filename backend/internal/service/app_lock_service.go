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
	"gorm.io/gorm"
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
func (s *AppLockService) Acquire(ctx context.Context, force bool) error {
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
		return fmt.Errorf("encode lock value: %w", err)
	}

	var query string
	switch s.db.Name() {
	case "sqlite":
		query = `
			INSERT INTO kv (key, value)
			VALUES (?, ?)
			ON CONFLICT(key) DO UPDATE SET 
				value = excluded.value
			WHERE 
				(json_extract(kv.value, '$.expires_at') < ?) OR ?
			RETURNING json_extract(kv.value, '$.expires_at') AS prev_expires_at
		`
	case "postgres":
		query = `
			INSERT INTO kv (key, value)
			VALUES ($1, $2)
			ON CONFLICT(key) DO UPDATE SET 
				value = excluded.value
			WHERE 
				((kv.value::json->>'expires_at')::bigint < $3) OR $4
			RETURNING (kv.value::json->>'expires_at')::bigint AS prev_expires_at
		`
	default:
		return fmt.Errorf("unsupported database dialect: %s", s.db.Name())
	}

	var prevExpires int64
	res := s.db.WithContext(ctx).Raw(query, lockKey, raw, nowUnix, force).Scan(&prevExpires)
	if res.Error != nil {
		return fmt.Errorf("lock acquisition failed: %w", res.Error)
	}

	// If there is a lock that is not expired and force is false, no rows will be affected
	if res.RowsAffected == 0 {
		return ErrLockUnavailable
	}

	wait := time.Duration(0)
	if force && prevExpires > nowUnix {
		wait = time.Until(time.Unix(prevExpires, 0))
	}

	slog.Info("Acquired application lock",
		slog.Int64("process_id", s.processID),
		slog.String("host_id", s.hostID),
		slog.Duration("wait_before_proceeding", wait),
	)

	// If we forcefully acquired the lock, wait until the previous lock has expired
	if wait > 0 {
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(wait):
		}
	}

	return nil
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
			if err := s.renew(ctx); err != nil {
				return fmt.Errorf("renew lock: %w", err)
			}
		}
	}
}

// Release releases the lock if it is held by this process.
func (s *AppLockService) Release(ctx context.Context) error {
	opCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

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

	res := s.db.WithContext(opCtx).Exec(query, lockKey, s.lockID)
	if res.Error != nil {
		return fmt.Errorf("release lock failed: %w", res.Error)
	}

	if res.RowsAffected > 0 {
		slog.Info("Released application lock",
			slog.Int64("process_id", s.processID),
			slog.String("host_id", s.hostID),
		)
	}
	return nil
}

// renew tries to renew the lock, retrying up to renewRetries times (sleeping 1s between attempts).
func (s *AppLockService) renew(ctx context.Context) error {
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

		res := s.db.WithContext(ctx).Exec(query, raw, lockKey, s.lockID, nowUnix)
		switch {
		case res.Error != nil:
			lastErr = fmt.Errorf("lock renewal failed: %w", res.Error)
		case res.RowsAffected == 0:
			// If there isn't a lock with the same lock ID and not expired, we lost the lock
			lastErr = ErrLockLost
		default:
			slog.Debug("Renewed application lock",
				slog.Int64("process_id", s.processID),
				slog.String("host_id", s.hostID),
			)
			return nil
		}

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
