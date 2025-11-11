package utils

import (
	"context"

	"gorm.io/gorm"
)

type transactionContextKey struct{}

// ContextWithTransaction stores a database transaction inside a context.
func ContextWithTransaction(ctx context.Context, tx *gorm.DB) context.Context {
	if tx == nil {
		return ctx
	}
	return context.WithValue(ctx, transactionContextKey{}, tx)
}

// TransactionFromContext retrieves a database transaction from a context.
func TransactionFromContext(ctx context.Context) *gorm.DB {
	tx, ok := (ctx.Value(transactionContextKey{})).(*gorm.DB)
	if !ok || tx == nil {
		return nil
	}

	return tx
}
