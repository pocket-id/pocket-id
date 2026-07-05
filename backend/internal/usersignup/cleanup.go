package usersignup

import (
	"context"
	"time"

	"gorm.io/gorm"

	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
)

// CleanupExpiredSignupTokens deletes signup tokens that have expired
// It returns the number of rows removed
func CleanupExpiredSignupTokens(ctx context.Context, db *gorm.DB) (int64, error) {
	st := db.
		WithContext(ctx).
		Delete(&SignupToken{}, "expires_at < ?", datatype.DateTime(time.Now()))
	return st.RowsAffected, st.Error
}
