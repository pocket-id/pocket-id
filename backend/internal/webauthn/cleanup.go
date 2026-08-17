package webauthn

import (
	"context"
	"time"

	"gorm.io/gorm"

	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
)

// cleanupExpiredSessions deletes WebAuthn sessions that have expired
// It returns the number of rows removed
func cleanupExpiredSessions(ctx context.Context, db *gorm.DB) (int64, error) {
	st := db.
		WithContext(ctx).
		Delete(&WebauthnSession{}, "expires_at < ?", datatype.DateTime(time.Now()))
	return st.RowsAffected, st.Error
}

// cleanupExpiredReauthenticationTokens deletes reauthentication tokens that have expired
// It returns the number of rows removed
func cleanupExpiredReauthenticationTokens(ctx context.Context, db *gorm.DB) (int64, error) {
	st := db.
		WithContext(ctx).
		Delete(&ReauthenticationToken{}, "expires_at < ?", datatype.DateTime(time.Now()))
	return st.RowsAffected, st.Error
}
