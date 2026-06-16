package oidc

import (
	"context"

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
	if tx, ok := ctx.Value(txContextKey{}).(*gorm.DB); ok {
		return tx.WithContext(ctx)
	}
	return fallback.WithContext(ctx)
}

// withTx runs fn inside a transaction, committing on nil error. Nested calls join the
// outer transaction.
func withTx(ctx context.Context, db *gorm.DB, fn func(ctx context.Context) error) error {
	if _, ok := ctx.Value(txContextKey{}).(*gorm.DB); ok {
		return fn(ctx)
	}

	tx := db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return tx.Error
	}
	defer tx.Rollback()

	if err := fn(contextWithTx(ctx, tx)); err != nil {
		return err
	}
	return tx.Commit().Error
}
