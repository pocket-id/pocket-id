package devicelogin

import (
	"context"
	"time"

	"gorm.io/gorm"

	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
)

// CleanupExpiredRequests deletes device login requests that have expired
// It returns the number of rows removed
func CleanupExpiredRequests(ctx context.Context, db *gorm.DB) (int64, error) {
	statement := db.
		WithContext(ctx).
		Delete(&Request{}, "expires_at < ?", datatype.DateTime(time.Now()))
	return statement.RowsAffected, statement.Error
}
