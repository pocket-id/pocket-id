package oidc

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"gorm.io/gorm"
)

// Transactions are carried in the context, every database access goes through dbFromContext.
// Although storing transactions in the context is not preferred, fosite handlers don't allow passing a transaction
// so we have to do it this way.

type txContextKey struct{}

func contextWithTx(ctx context.Context, tx *gorm.DB) context.Context {
	return context.WithValue(ctx, txContextKey{}, tx)
}

func dbFromContext(ctx context.Context, fallback *gorm.DB) *gorm.DB {
	tx, ok := ctx.Value(txContextKey{}).(*gorm.DB)
	if ok {
		return tx.WithContext(ctx)
	}

	return fallback.WithContext(ctx)
}

// withTx runs fn inside a transaction, committing on nil error. Nested calls join the
// outer transaction.
func withTx(ctx context.Context, db *gorm.DB, fn func(ctx context.Context) error) error {
	_, ok := ctx.Value(txContextKey{}).(*gorm.DB)
	if ok {
		return fn(ctx)
	}

	tx := db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return fmt.Errorf("error starting transaction: %w", tx.Error)
	}

	var committed bool
	defer func() {
		if committed {
			return
		}
		rErr := tx.Rollback().Error
		if rErr != nil && !errors.Is(rErr, gorm.ErrInvalidTransaction) {
			slog.ErrorContext(ctx, "Failed to rollback transaction", "error", rErr)
		}
	}()

	err := fn(contextWithTx(ctx, tx))
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
