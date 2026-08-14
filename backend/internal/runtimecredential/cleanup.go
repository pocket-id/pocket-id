package runtimecredential

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/model"
	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
)

// CleanupExpiredChallenges implements FCA10 by removing expired proof state through the existing scheduled database cleanup system
func CleanupExpiredChallenges(ctx context.Context, db *gorm.DB) (int64, error) {
	result := db.WithContext(ctx).Delete(&model.RuntimeCredentialChallenge{}, "expires_at < ?", datatype.DateTime(time.Now()))
	return result.RowsAffected, result.Error
}
