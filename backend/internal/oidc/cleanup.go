package oidc

import (
	"context"
	"time"

	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
	"gorm.io/gorm"
)

// CleanupExpiredOAuth2Sessions deletes OAuth2 sessions whose tokens or codes have
// expired.
//
// Invalidated-but-unexpired rows are intentionally KEPT until their original expiry: fosite relies on
// finding the inactive row to detect a reuse and revoke the affected token family.
func CleanupExpiredOAuth2Sessions(ctx context.Context, db *gorm.DB) (int64, error) {
	st := db.
		WithContext(ctx).
		Delete(&OAuth2Session{}, "expires_at < ?", datatype.DateTime(time.Now()))
	return st.RowsAffected, st.Error
}

// CleanupExpiredClientAssertionJTIs deletes expired JWT IDs used for client assertion replay protection.
func CleanupExpiredClientAssertionJTIs(ctx context.Context, db *gorm.DB) (int64, error) {
	st := db.
		WithContext(ctx).
		Delete(&clientAssertionJTI{}, "expires_at < ?", datatype.DateTime(time.Now()))
	return st.RowsAffected, st.Error
}

// CleanupAbandonedInteractionSessions removes interaction sessions that were never completed.
func CleanupAbandonedInteractionSessions(ctx context.Context, db *gorm.DB) (int64, error) {
	st := db.
		WithContext(ctx).
		Delete(&InteractionSession{}, "created_at < ?", datatype.DateTime(time.Now().Add(-interactionSessionLifetime)))
	return st.RowsAffected, st.Error
}
