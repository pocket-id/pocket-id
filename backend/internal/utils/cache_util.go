package utils

import (
	"context"
	"sync/atomic"
	"time"

	"golang.org/x/sync/singleflight"
)

type CacheEntry[T any] struct {
	Value     T
	FetchedAt time.Time
}

type ErrStale struct {
	Err error
}

func (e *ErrStale) Error() string { return "returned stale cache: " + e.Err.Error() }
func (e *ErrStale) Unwrap() error { return e.Err }

type Cache[T any] struct {
	ttl   time.Duration
	entry atomic.Pointer[CacheEntry[T]]
	sf    singleflight.Group
}

func New[T any](ttl time.Duration) *Cache[T] {
	return &Cache[T]{ttl: ttl}
}

// Get returns the cached value if it's still fresh.
func (c *Cache[T]) Get() (T, bool) {
	entry := c.entry.Load()
	if entry == nil {
		var zero T
		return zero, false
	}
	if time.Since(entry.FetchedAt) < c.ttl {
		return entry.Value, true
	}
	var zero T
	return zero, false
}

// GetOrFetch returns the cached value if it's still fresh, otherwise calls fetch to get a new value.
func (c *Cache[T]) GetOrFetch(ctx context.Context, fetch func(context.Context) (T, error)) (T, error) {
	// If fresh, serve immediately
	if v, ok := c.Get(); ok {
		return v, nil
	}

	// Fetch with singleflight to prevent multiple concurrent fetches
	vAny, err, _ := c.sf.Do("singleton", func() (any, error) {
		if v2, ok := c.Get(); ok {
			return v2, nil
		}
		val, fetchErr := fetch(ctx)
		if fetchErr != nil {
			return nil, fetchErr
		}
		c.entry.Store(&CacheEntry[T]{Value: val, FetchedAt: time.Now()})
		return val, nil
	})

	if err == nil {
		return vAny.(T), nil
	}

	// Fetch failed. Return stale if possible.
	if e := c.entry.Load(); e != nil {
		return e.Value, &ErrStale{Err: err}
	}

	var zero T
	return zero, err
}
