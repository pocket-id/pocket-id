package oidc

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"

	"gorm.io/gorm"
)

// Transactions are carried in the context, every database access goes through dbFromContext.
// Although storing transactions in the context is not preferred, fosite handlers don't allow passing a transaction
// so we have to do it this way.

type txContextKey struct{}
type txHooksKey struct{}

type postTxHook struct {
	onCommit bool
	ran      bool
	fn       func(context.Context)
}

type txHooks struct {
	mu           sync.Mutex
	done         bool
	committed    bool
	rolledBack   bool
	onFinish     []func(committed bool)
	postTx       []*postTxHook
	postTxByKey  map[string]*postTxHook
	repairedCIMD map[string]uint64
}

func txHooksFromContext(ctx context.Context) *txHooks {
	hooks, _ := ctx.Value(txHooksKey{}).(*txHooks)
	return hooks
}

func hasTxHooks(ctx context.Context) bool {
	return txHooksFromContext(ctx) != nil
}

func markTxCIMDRepaired(ctx context.Context, clientID string, gen uint64) {
	if gen == 0 {
		return
	}
	hooks, ok := ctx.Value(txHooksKey{}).(*txHooks)
	if !ok || hooks == nil {
		return
	}
	hooks.mu.Lock()
	if hooks.repairedCIMD == nil {
		hooks.repairedCIMD = make(map[string]uint64)
	}
	hooks.repairedCIMD[clientID] = gen
	hooks.mu.Unlock()
}

func txRepairedCIMD(ctx context.Context, clientID string, gen uint64) bool {
	if gen == 0 {
		return false
	}
	hooks, ok := ctx.Value(txHooksKey{}).(*txHooks)
	if !ok || hooks == nil {
		return false
	}
	hooks.mu.Lock()
	repaired := hooks.repairedCIMD[clientID] == gen
	hooks.mu.Unlock()
	return repaired
}

// contextWithoutTx detaches cancellation and clears any active transaction value while preserving the parent context chain.
func contextWithoutTx(ctx context.Context) context.Context {
	return context.WithValue(context.WithoutCancel(ctx), txContextKey{}, nil)
}

func txFromContext(ctx context.Context) (*gorm.DB, bool) {
	tx, ok := ctx.Value(txContextKey{}).(*gorm.DB)
	return tx, ok && tx != nil
}

// afterTxFinish registers a non-blocking finalizer that runs immediately after the enclosing withTx commits or rolls back,
// before any post-tx callbacks that may acquire database locks.
func afterTxFinish(ctx context.Context, fn func(committed bool)) bool {
	hooks := txHooksFromContext(ctx)
	if hooks == nil {
		return false
	}
	hooks.mu.Lock()
	if hooks.done {
		committed := hooks.committed
		hooks.mu.Unlock()
		fn(committed)
		return true
	}
	hooks.onFinish = append(hooks.onFinish, fn)
	hooks.mu.Unlock()
	return true
}

func registerPostTxOnce(ctx context.Context, key string, onCommit bool, fn func(context.Context)) bool {
	hooks := txHooksFromContext(ctx)
	if hooks == nil {
		return false
	}
	hooks.mu.Lock()
	if hooks.postTxByKey == nil {
		hooks.postTxByKey = make(map[string]*postTxHook)
	}
	if existing, exists := hooks.postTxByKey[key]; exists {
		if onCommit {
			existing.onCommit = true
		}
		if !existing.ran {
			existing.fn = fn
		}
		var runFn func(context.Context)
		if hooks.done && !existing.ran && (hooks.rolledBack || (existing.onCommit && hooks.committed)) {
			existing.ran = true
			runFn = existing.fn
		}
		hooks.mu.Unlock()
		if runFn != nil {
			runFn(contextWithoutTx(ctx))
		}
		return true
	}
	entry := &postTxHook{onCommit: onCommit, fn: fn}
	hooks.postTxByKey[key] = entry
	if hooks.done {
		shouldRun := hooks.rolledBack || (onCommit && hooks.committed)
		entry.ran = shouldRun
		hooks.mu.Unlock()
		if shouldRun {
			fn(contextWithoutTx(ctx))
		}
		return true
	}
	hooks.postTx = append(hooks.postTx, entry)
	hooks.mu.Unlock()
	return true
}

// cancelPostTxOnCommit disarms commit-time execution of a keyed post-tx callback (while preserving rollback execution)
// when a subsequent operation in the same transaction supersedes the failed step.
func cancelPostTxOnCommit(ctx context.Context, key string) {
	hooks := txHooksFromContext(ctx)
	if hooks == nil {
		return
	}
	hooks.mu.Lock()
	if existing, exists := hooks.postTxByKey[key]; exists && !hooks.done {
		existing.onCommit = false
	}
	hooks.mu.Unlock()
}

// afterTxRollbackOnce registers fn to run at most once per key after the enclosing withTx rolls back,
// receiving a detached non-transactional context derived from ctx.
// If the transaction has already rolled back and key has not yet run, fn runs immediately after releasing the hook lock.
// It returns false when ctx is not inside a withTx-managed transaction.
func afterTxRollbackOnce(ctx context.Context, key string, fn func(context.Context)) bool {
	return registerPostTxOnce(ctx, key, false, fn)
}

// afterTxSettledOnce registers fn to run at most once per key after the enclosing withTx finishes
// (whether it commits or rolls back), receiving a detached non-transactional context derived from ctx.
// It returns false when ctx is not inside a withTx-managed transaction.
func afterTxSettledOnce(ctx context.Context, key string, fn func(context.Context)) bool {
	return registerPostTxOnce(ctx, key, true, fn)
}

func contextWithTx(ctx context.Context, tx *gorm.DB) context.Context {
	return context.WithValue(ctx, txContextKey{}, tx)
}

func dbFromContext(ctx context.Context, fallback *gorm.DB) *gorm.DB {
	if tx, ok := txFromContext(ctx); ok {
		return tx.WithContext(ctx)
	}

	return fallback.WithContext(ctx)
}

// withTx runs fn inside a transaction, committing on nil error. Nested calls join the
// outer transaction.
func withTx(ctx context.Context, db *gorm.DB, fn func(ctx context.Context) error) error {
	if _, ok := txFromContext(ctx); ok {
		return fn(ctx)
	}

	tx := db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return fmt.Errorf("error starting transaction: %w", tx.Error)
	}

	hooks := &txHooks{}
	var committed bool
	defer func() {
		if committed {
			hooks.mu.Lock()
			hooks.done = true
			hooks.committed = true
			onFinish := hooks.onFinish
			var toRun []func(context.Context)
			for _, entry := range hooks.postTx {
				if entry.onCommit && !entry.ran {
					entry.ran = true
					toRun = append(toRun, entry.fn)
				}
			}
			hooks.onFinish = nil
			hooks.postTx = nil
			hooks.mu.Unlock()
			for _, fn := range onFinish {
				fn(true)
			}
			if len(toRun) > 0 {
				settledCtx := context.WithoutCancel(ctx)
				for _, fn := range toRun {
					fn(settledCtx)
				}
			}
			return
		}
		rErr := tx.Rollback().Error
		if rErr != nil && !errors.Is(rErr, gorm.ErrInvalidTransaction) {
			slog.ErrorContext(ctx, "Failed to rollback transaction", "error", rErr)
		}
		hooks.mu.Lock()
		hooks.done = true
		hooks.rolledBack = true
		onFinish := hooks.onFinish
		var toRun []func(context.Context)
		for _, entry := range hooks.postTx {
			if !entry.ran {
				entry.ran = true
				toRun = append(toRun, entry.fn)
			}
		}
		hooks.onFinish = nil
		hooks.postTx = nil
		hooks.mu.Unlock()
		for _, fn := range onFinish {
			fn(false)
		}
		if len(toRun) > 0 {
			rollbackCtx := context.WithoutCancel(ctx)
			for _, fn := range toRun {
				fn(rollbackCtx)
			}
		}
	}()

	err := fn(context.WithValue(contextWithTx(ctx, tx), txHooksKey{}, hooks))
	if err != nil {
		return err
	}

	err = tx.Commit().Error
	if err != nil {
		return fmt.Errorf("error committing transaction: %w", err)
	}
	committed = true

	return nil
}
