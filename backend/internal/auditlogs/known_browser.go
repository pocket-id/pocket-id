package auditlogs

import (
	"context"
	"crypto/rand"
	"time"

	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/utils"
)

const KnownBrowserLifetime = 180 * 24 * time.Hour

// knownBrowser recognizes a browser for notifications without granting authentication
// Only the hash of the cookie token is stored
type knownBrowser struct {
	UserID    string `gorm:"primaryKey"`
	TokenHash string `gorm:"primaryKey"`
	ExpiresAt int64
}

func (s *service) rememberBrowser(ctx context.Context, tx *gorm.DB, userID, token string) (bool, string, error) {
	now := time.Now()
	expiresAt := now.Add(KnownBrowserLifetime).Unix()

	// Extend only an unexpired token already associated with the authenticated user
	result := tx.WithContext(ctx).Model(&knownBrowser{}).
		Where("user_id = ? AND token_hash = ? AND expires_at > ?", userID, utils.CreateSha256Hash(token), now.Unix()).
		Update("expires_at", expiresAt)
	if result.Error != nil {
		return false, "", result.Error
	}
	if result.RowsAffected > 0 {
		return true, token, nil
	}

	// Replace unrecognized tokens with server-generated randomness to avoid accepting a caller-chosen browser identity
	token = rand.Text()
	record := knownBrowser{UserID: userID, TokenHash: utils.CreateSha256Hash(token), ExpiresAt: expiresAt}
	if err := tx.WithContext(ctx).Create(&record).Error; err != nil {
		return false, "", err
	}
	return false, token, nil
}
