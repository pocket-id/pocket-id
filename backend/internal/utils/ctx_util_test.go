package utils

import (
	"context"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// newContextTestDB creates a minimal in-memory SQLite database for testing
func newContextTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{
		SkipDefaultTransaction: true, // Ensure we control transactions explicitly
	})
	require.NoError(t, err, "Failed to create test database")
	return db
}

func TestContextWithTransaction(t *testing.T) {
	db := newContextTestDB(t)

	t.Run("should store transaction in context", func(t *testing.T) {
		tx := db.Begin()
		require.NoError(t, tx.Error)
		defer tx.Rollback()

		ctxWithTx := ContextWithTransaction(t.Context(), tx)

		// Verify the transaction was stored
		assert.NotNil(t, ctxWithTx)
		storedValue := ctxWithTx.Value(transactionContextKey{})
		assert.NotNil(t, storedValue)
		assert.IsType(t, &gorm.DB{}, storedValue)
	})

	t.Run("storing nil transaction is a no-op", func(t *testing.T) {
		ctx := t.Context()
		ctxWithTx := ContextWithTransaction(ctx, nil)

		assert.NotNil(t, ctxWithTx)
		storedValue := ctxWithTx.Value(transactionContextKey{})
		assert.Nil(t, storedValue)
	})

	t.Run("should preserve existing context values", func(t *testing.T) {
		tx := db.Begin()
		require.NoError(t, tx.Error)
		defer tx.Rollback()

		type testKey struct{}
		ctx := context.WithValue(t.Context(), testKey{}, "test-value")
		ctxWithTx := ContextWithTransaction(ctx, tx)

		// Verify original context value is preserved
		assert.Equal(t, "test-value", ctxWithTx.Value(testKey{}))
		// Verify transaction was added
		assert.NotNil(t, ctxWithTx.Value(transactionContextKey{}))
	})
}

func TestTransactionFromContext(t *testing.T) {
	db := newContextTestDB(t)

	// Create a simple test table
	err := db.Exec("CREATE TABLE IF NOT EXISTS test_table (id INTEGER PRIMARY KEY, name TEXT)").Error
	require.NoError(t, err)

	t.Run("should retrieve active transaction from context", func(t *testing.T) {
		tx := db.Begin()
		require.NoError(t, tx.Error)
		defer tx.Rollback()

		ctx := ContextWithTransaction(t.Context(), tx)
		retrievedTx := TransactionFromContext(ctx)

		assert.NotNil(t, retrievedTx)
		// Verify it's the same transaction by checking the underlying connection
		assert.Same(t, tx, retrievedTx)
	})

	t.Run("should return nil when context has no transaction", func(t *testing.T) {
		ctx := t.Context()
		retrievedTx := TransactionFromContext(ctx)

		assert.Nil(t, retrievedTx)
	})

	t.Run("should return nil when transaction value is nil", func(t *testing.T) {
		ctx := ContextWithTransaction(t.Context(), nil)
		retrievedTx := TransactionFromContext(ctx)

		assert.Nil(t, retrievedTx)
	})

	t.Run("should return nil when context value is not a transaction", func(t *testing.T) {
		// Manually create a context with wrong type
		ctx := context.WithValue(t.Context(), transactionContextKey{}, "not-a-transaction")
		retrievedTx := TransactionFromContext(ctx)

		assert.Nil(t, retrievedTx)
	})

	t.Run("should return regular db connection if transaction not started", func(t *testing.T) {
		// Store a regular DB connection (not a transaction) in the context
		ctx := ContextWithTransaction(t.Context(), db)
		retrievedTx := TransactionFromContext(ctx)

		assert.NotNil(t, retrievedTx)
		// Verify it's the same database by checking the underlying connection
		assert.Same(t, db, retrievedTx)
	})
}
