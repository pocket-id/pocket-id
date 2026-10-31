package usersession

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/lestrrat-go/jwx/v4/jwt"
	"gorm.io/gorm"

	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
)

type Service struct {
	db *gorm.DB
}

func newService(db *gorm.DB) *Service {
	return &Service{db: db}
}

// Track records the session a freshly minted access token belongs to, writing through tx so it commits atomically with the sign-in
// The token is parsed without signature verification because it never left the process
func (s *Service) Track(ctx context.Context, tx *gorm.DB, accessToken string) error {
	token, err := jwt.ParseInsecure([]byte(accessToken))
	if err != nil {
		return fmt.Errorf("failed to parse access token: %w", err)
	}

	jti, ok := token.JwtID()
	if !ok || jti == "" {
		return errors.New("access token has no jti")
	}
	subject, ok := token.Subject()
	if !ok || subject == "" {
		return errors.New("access token has no subject")
	}
	expiration, ok := token.Expiration()
	if !ok {
		return errors.New("access token has no expiration")
	}

	session := UserSession{
		ExpiresAt: datatype.DateTime(expiration),
		UserID:    subject,
	}
	session.ID = jti

	return tx.WithContext(ctx).Create(&session).Error
}

// Validate checks that the session for the given jti still exists, returning gorm.ErrRecordNotFound when it doesn't
// Token expiry is left to the JWT verification
func (s *Service) Validate(ctx context.Context, jti string) error {
	return s.db.
		WithContext(ctx).
		Select("id").
		Take(&UserSession{}, "id = ?", jti).
		Error
}

// Delete ends the session for the given jti; deleting an already ended session is not an error
func (s *Service) Delete(ctx context.Context, jti string) error {
	return s.db.
		WithContext(ctx).
		Where("id = ?", jti).
		Delete(&UserSession{}).
		Error
}

// cleanupExpired deletes sessions whose tokens have expired
func (s *Service) cleanupExpired(ctx context.Context) (int64, error) {
	res := s.db.
		WithContext(ctx).
		Where("expires_at < ?", datatype.DateTime(time.Now())).
		Delete(&UserSession{})
	return res.RowsAffected, res.Error
}
