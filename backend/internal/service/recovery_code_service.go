package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
)

// Number of recovery codes issued in one batch.
const recoveryCodeBatchSize = 10

// Length of each raw recovery code, excluding separators.
const recoveryCodeRawLength = 16

type RecoveryCodeService struct {
	db               *gorm.DB
	appConfigService *AppConfigService
	jwtService       *JwtService
	auditLogService  *AuditLogService
}

func NewRecoveryCodeService(db *gorm.DB, appConfigService *AppConfigService, jwtService *JwtService, auditLogService *AuditLogService) *RecoveryCodeService {
	return &RecoveryCodeService{
		db:               db,
		appConfigService: appConfigService,
		jwtService:       jwtService,
		auditLogService:  auditLogService,
	}
}

// GenerateForUser revokes any existing codes for the user and issues a fresh batch.
// The raw codes are returned only once.
func (s *RecoveryCodeService) GenerateForUser(ctx context.Context, userID, ipAddress, userAgent string) ([]string, error) {
	if !s.appConfigService.GetDbConfig().AllowRecoveryCodes.IsTrue() {
		return nil, &common.RecoveryCodesDisabledError{}
	}

	tx := s.db.Begin()
	defer func() {
		tx.Rollback()
	}()

	// Drop any previous codes so the user only ever has one active batch.
	err := tx.
		WithContext(ctx).
		Where("user_id = ?", userID).
		Delete(&model.RecoveryCode{}).
		Error
	if err != nil {
		return nil, err
	}

	rawCodes := make([]string, 0, recoveryCodeBatchSize)
	records := make([]model.RecoveryCode, 0, recoveryCodeBatchSize)

	for i := 0; i < recoveryCodeBatchSize; i++ {
		raw, err := utils.GenerateRandomUnambiguousString(recoveryCodeRawLength)
		if err != nil {
			return nil, err
		}

		rawCodes = append(rawCodes, formatRecoveryCode(raw))
		// Hash the normalised form (lower-case, no separators) so a user
		// who types "ABCD efgh" during recovery still matches regardless
		// of case or spacing.
		records = append(records, model.RecoveryCode{
			UserID:   userID,
			CodeHash: utils.CreateSha256Hash(normalizeRecoveryCode(raw)),
		})
	}

	err = tx.WithContext(ctx).Create(&records).Error
	if err != nil {
		return nil, err
	}

	s.auditLogService.Create(ctx, model.AuditLogEventRecoveryCodesGenerated, ipAddress, userAgent, userID, model.AuditLogData{}, tx)

	err = tx.Commit().Error
	if err != nil {
		return nil, err
	}

	return rawCodes, nil
}

// ListForUser returns the code metadata (without the code itself) for the given user,
// ordered by creation time.
func (s *RecoveryCodeService) ListForUser(ctx context.Context, userID string) ([]model.RecoveryCode, error) {
	var codes []model.RecoveryCode
	err := s.db.
		WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at ASC").
		Find(&codes).
		Error
	if err != nil {
		return nil, err
	}
	return codes, nil
}

// StatusForUser returns the total count and the number of unused codes for the user.
// Returns a RecoveryCodesDisabledError if the feature has been disabled globally.
func (s *RecoveryCodeService) StatusForUser(ctx context.Context, userID string) (total, unused int, err error) {
	if !s.appConfigService.GetDbConfig().AllowRecoveryCodes.IsTrue() {
		return 0, 0, &common.RecoveryCodesDisabledError{}
	}
	var totalCount, unusedCount int64
	err = s.db.
		WithContext(ctx).
		Model(&model.RecoveryCode{}).
		Where("user_id = ?", userID).
		Count(&totalCount).
		Error
	if err != nil {
		return 0, 0, err
	}
	err = s.db.
		WithContext(ctx).
		Model(&model.RecoveryCode{}).
		Where("user_id = ? AND used_at IS NULL", userID).
		Count(&unusedCount).
		Error
	if err != nil {
		return 0, 0, err
	}
	return int(totalCount), int(unusedCount), nil
}

// RevokeAllForUser removes every recovery code of the given user.
// Returns a RecoveryCodesDisabledError if the feature has been disabled globally
// (in that case every code has already been purged as part of the config change).
func (s *RecoveryCodeService) RevokeAllForUser(ctx context.Context, userID, ipAddress, userAgent string) error {
	if !s.appConfigService.GetDbConfig().AllowRecoveryCodes.IsTrue() {
		return &common.RecoveryCodesDisabledError{}
	}
	tx := s.db.Begin()
	defer func() {
		tx.Rollback()
	}()

	err := tx.
		WithContext(ctx).
		Where("user_id = ?", userID).
		Delete(&model.RecoveryCode{}).
		Error
	if err != nil {
		return err
	}

	s.auditLogService.Create(ctx, model.AuditLogEventRecoveryCodesRevoked, ipAddress, userAgent, userID, model.AuditLogData{}, tx)

	return tx.Commit().Error
}

// Redeem validates an entered recovery code. On success it marks the code as used,
// returns the user and a signed access token suitable for a session cookie.
func (s *RecoveryCodeService) Redeem(ctx context.Context, rawCode, ipAddress, userAgent string) (model.User, string, error) {
	if !s.appConfigService.GetDbConfig().AllowRecoveryCodes.IsTrue() {
		return model.User{}, "", &common.RecoveryCodesDisabledError{}
	}

	normalized := normalizeRecoveryCode(rawCode)
	if len(normalized) != recoveryCodeRawLength {
		return model.User{}, "", &common.RecoveryCodeInvalidError{}
	}

	hash := utils.CreateSha256Hash(normalized)

	tx := s.db.Begin()
	defer func() {
		tx.Rollback()
	}()

	var record model.RecoveryCode
	err := tx.
		WithContext(ctx).
		Where("code_hash = ? AND used_at IS NULL", hash).
		Preload("User").
		Clauses(clause.Locking{Strength: "UPDATE"}).
		First(&record).
		Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return model.User{}, "", &common.RecoveryCodeInvalidError{}
		}
		return model.User{}, "", err
	}

	now := datatype.DateTime(time.Now())
	record.UsedAt = &now

	err = tx.WithContext(ctx).Save(&record).Error
	if err != nil {
		return model.User{}, "", err
	}

	accessToken, err := s.jwtService.GenerateAccessToken(record.User, AuthenticationMethodRecoveryCode)
	if err != nil {
		return model.User{}, "", err
	}

	s.auditLogService.Create(ctx, model.AuditLogEventRecoveryCodeSignIn, ipAddress, userAgent, record.User.ID, model.AuditLogData{}, tx)

	err = tx.Commit().Error
	if err != nil {
		return model.User{}, "", err
	}

	return record.User, accessToken, nil
}

// normalizeRecoveryCode strips spaces and dashes and lower-cases the string.
// Recovery codes are issued from an unambiguous alphabet that mixes both cases,
// but we accept whatever the user typed so long as the underlying characters
// match. Callers must normalise on BOTH write and compare so that hashing stays
// consistent across flows.
func normalizeRecoveryCode(raw string) string {
	cleaned := strings.ReplaceAll(raw, "-", "")
	cleaned = strings.ReplaceAll(cleaned, " ", "")
	return strings.ToLower(cleaned)
}

// formatRecoveryCode inserts dashes every 4 characters for readability
// (e.g. "abcd-efgh-ijkl-mnop").
func formatRecoveryCode(raw string) string {
	var b strings.Builder
	for i, r := range raw {
		if i > 0 && i%4 == 0 {
			b.WriteByte('-')
		}
		b.WriteRune(r)
	}
	return b.String()
}
