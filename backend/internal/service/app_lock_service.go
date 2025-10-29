package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/pocket-id/pocket-id/backend/internal/model"
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
	}
}

type lockValue struct {
	ProcessID int64  `json:"process_id"`
	HostID    string `json:"host_id"`
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
	nowMillis := now.UnixMilli()
	value := lockValue{
		ProcessID: s.processID,
		HostID:    s.hostID,
		ExpiresAt: now.Add(ttl).UnixMilli(),
	}

	raw, err := value.Marshal()
	if err != nil {
		return fmt.Errorf("encode lock value: %w", err)
	}

	tx := s.db.WithContext(ctx).Begin()
	defer func() { tx.Rollback() }()

	res := tx.
		Clauses(clause.OnConflict{DoNothing: true}).
		Create(&model.KV{Key: lockKey, Value: &raw})
	if res.Error != nil {
		return fmt.Errorf("insert lock row: %w", res.Error)
	}
	if res.RowsAffected == 1 {
		return tx.Commit().Error
	}

	current, err := s.loadLockForUpdate(tx)
	if err != nil {
		return err
	} else if current == nil {
		return errors.New("lock row is missing")
	}

	needsForceAcquire := (current.ProcessID != s.processID || current.HostID != s.hostID) && current.ExpiresAt > nowMillis
	if !force && needsForceAcquire {
		return ErrLockUnavailable
	}

	if err := s.writeLock(tx, value); err != nil {
		return fmt.Errorf("update lock row: %w", err)
	}

	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("commit lock transaction: %w", err)
	}

	slog.Info("Acquired application lock",
		slog.Int64("process_id", s.processID),
		slog.String("host_id", s.hostID),
	)

	remaining := time.Until(time.UnixMilli(current.ExpiresAt))
	if needsForceAcquire && remaining > 0 {
		slog.Info("Waiting until previous application lock expires",
			slog.Int64("process_id", s.processID),
			slog.String("host_id", s.hostID),
			slog.Int64("previous_process_id", current.ProcessID),
			slog.String("previous_host_id", current.HostID),
			slog.Duration("wait_duration", remaining),
		)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(remaining):
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

	err := s.db.WithContext(opCtx).Transaction(func(tx *gorm.DB) error {
		current, err := s.loadLockForUpdate(tx)
		if err != nil {
			return err
		}
		if current == nil {
			return nil
		}
		if current.ProcessID != s.processID || current.HostID != s.hostID {
			return nil
		}
		return tx.Delete(&model.KV{}, "key = ?", lockKey).Error
	})
	if err != nil {
		return fmt.Errorf("release lock: %w", err)
	}

	slog.Info("Released application lock",
		slog.Int64("process_id", s.processID),
		slog.String("host_id", s.hostID),
	)

	return nil
}

// renew tries to renew the lock, retrying up to renewRetries times (sleeping 1s between attempts).
func (s *AppLockService) renew(ctx context.Context) error {
	var joinedErr error
	for attempt := 1; attempt <= renewRetries; attempt++ {
		now := time.Now()
		nowMillis := now.UnixMilli()
		expiresAt := now.Add(ttl).UnixMilli()

		opCtx, cancel := context.WithTimeout(ctx, 31*time.Second)

		err := s.db.WithContext(opCtx).Transaction(func(tx *gorm.DB) error {
			current, err := s.loadLockForUpdate(tx)
			if err != nil {
				return err
			}
			if current == nil {
				return ErrLockLost
			}
			if current.ProcessID != s.processID || current.HostID != s.hostID {
				return ErrLockLost
			}
			if current.ExpiresAt <= nowMillis {
				return ErrLockLost
			}

			current.ExpiresAt = expiresAt
			return s.writeLock(tx, *current)
		})

		cancel()

		if err == nil {
			return nil
		}
		if errors.Is(err, ErrLockLost) {
			return err
		}

		joinedErr = errors.Join(joinedErr, err)
		if attempt < renewRetries {
			slog.Warn("Failed to renew application lock",
				slog.Int("attempt", attempt),
				slog.Any("error", err),
			)
			time.Sleep(1 * time.Second)
			continue
		}
	}
	return fmt.Errorf("failed to renew lock after %d attempts: %w", renewRetries, joinedErr)
}

func (s *AppLockService) loadLockForUpdate(tx *gorm.DB) (*lockValue, error) {
	var row model.KV
	err := tx.
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("key = ?", lockKey).
		Take(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("load lock row: %w", err)
	}

	var value lockValue
	if err := value.Unmarshal(*row.Value); err != nil {
		return nil, fmt.Errorf("decode lock value: %w", err)
	}
	return &value, nil
}

func (s *AppLockService) writeLock(tx *gorm.DB, value lockValue) error {
	raw, err := value.Marshal()
	if err != nil {
		return fmt.Errorf("encode lock value: %w", err)
	}

	res := tx.Model(&model.KV{}).
		Where("key = ?", lockKey).
		Update("value", raw)
	if res.Error != nil {
		return fmt.Errorf("update lock row: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return errors.New("lock row is missing")
	}

	return nil
}
