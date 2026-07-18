package devicelogin

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
)

const (
	RequestDuration        = 15 * time.Minute
	PollingInterval        = 3
	codePrefix             = "P"
	codeRandomLength       = 7
	reauthenticationMaxAge = time.Minute
	// authenticationMethodOneTimePassword identifies the login-code-equivalent AMR used on the waiting device
	authenticationMethodOneTimePassword = "otp"
)

type Service struct {
	db       *gorm.DB
	signer   TokenService
	auditLog AuditLogger
	reauth   ReauthenticationTokenConsumer
}

type VerificationInfo struct {
	UserCode  string
	Device    string
	IPAddress string
	ExpiresAt datatype.DateTime
}

func newService(deps Dependencies) *Service {
	return &Service{
		db:       deps.DB,
		signer:   deps.Signer,
		auditLog: deps.AuditLog,
		reauth:   deps.Reauth,
	}
}

func (s *Service) Create(ctx context.Context, ipAddress, userAgent string) (Request, string, error) {
	// Bind the public request to a separate high-entropy secret that never enters the QR code
	deviceToken, err := utils.GenerateRandomAlphanumericString(32)
	if err != nil {
		return Request{}, "", err
	}

	now := time.Now().Round(time.Second)
	request := Request{
		DeviceTokenHash: utils.CreateSha256Hash(deviceToken),
		Status:          RequestStatusPending,
		ExpiresAt:       datatype.DateTime(now.Add(RequestDuration)),
		UserAgent:       userAgent,
		IpAddress:       ipAddress,
	}

	// Retry code generation because of the small but non-zero chance of a collision with an existing code
	for range 3 {
		request.Code, err = newUserCode()
		if err != nil {
			return Request{}, "", err
		}

		err = s.db.WithContext(ctx).Create(&request).Error
		if err == nil {
			return request, deviceToken, nil
		}
		if !errors.Is(err, gorm.ErrDuplicatedKey) {
			return Request{}, "", err
		}
	}

	return Request{}, "", errors.New("failed to generate a unique device login code")
}

func (s *Service) Inspect(ctx context.Context, code string) (VerificationInfo, error) {
	code = strings.ToUpper(strings.TrimSpace(code))

	// Return requester metadata only while the request can still be decided
	var request Request
	err := s.db.
		WithContext(ctx).
		Where("code = ? AND status = ? AND expires_at > ?", code, RequestStatusPending, datatype.DateTime(time.Now())).
		First(&request).
		Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return VerificationInfo{}, &common.DeviceLoginRequestInvalidOrExpiredError{}
		}
		return VerificationInfo{}, err
	}

	return VerificationInfo{
		UserCode:  request.Code,
		Device:    s.auditLog.DeviceStringFromUserAgent(request.UserAgent),
		IPAddress: request.IpAddress,
		ExpiresAt: request.ExpiresAt,
	}, nil
}

func (s *Service) Decide(ctx context.Context, code, decision, userID, reauthenticationToken string) error {
	code = strings.ToUpper(strings.TrimSpace(code))

	tx := s.db.Begin()
	defer tx.Rollback()

	var request Request
	err := tx.
		WithContext(ctx).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("code = ? AND status = ? AND expires_at > ?", code, RequestStatusPending, datatype.DateTime(time.Now())).
		First(&request).
		Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &common.DeviceLoginRequestInvalidOrExpiredError{}
		}
		return err
	}

	switch decision {
	case "approve":
		// Consume the fresh passkey proof in the same transaction as the approval
		if reauthenticationToken == "" {
			return &common.ReauthenticationRequiredError{}
		}
		reauthenticatedAt, consumeErr := s.reauth.ConsumeReauthenticationToken(ctx, tx, reauthenticationToken, userID)
		if consumeErr != nil {
			return consumeErr
		}
		if time.Since(reauthenticatedAt) > reauthenticationMaxAge {
			return &common.ReauthenticationRequiredError{}
		}
		request.Status = RequestStatusApproved
		request.UserID = &userID
	case "deny":
		request.Status = RequestStatusDenied
	default:
		return fmt.Errorf("unsupported device login decision %q", decision)
	}

	result := tx.
		WithContext(ctx).
		Model(&Request{}).
		Where("id = ? AND status = ? AND expires_at > ?", request.ID, RequestStatusPending, datatype.DateTime(time.Now())).
		Updates(map[string]any{"status": request.Status, "user_id": request.UserID})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return &common.DeviceLoginRequestInvalidOrExpiredError{}
	}

	return tx.Commit().Error
}

func (s *Service) Exchange(ctx context.Context, requestID, deviceToken, ipAddress, userAgent string) (model.User, string, RequestStatus, error) {
	if deviceToken == "" {
		return model.User{}, "", "", &common.DeviceLoginRequestInvalidOrExpiredError{}
	}

	tx := s.db.Begin()
	defer tx.Rollback()

	var request Request
	err := tx.
		WithContext(ctx).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Preload("User").
		Where("id = ? AND device_token_hash = ? AND expires_at > ?", requestID, utils.CreateSha256Hash(deviceToken), datatype.DateTime(time.Now())).
		First(&request).
		Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return model.User{}, "", "", &common.DeviceLoginRequestInvalidOrExpiredError{}
		}
		return model.User{}, "", "", err
	}

	switch request.Status {
	case RequestStatusPending:
		// Leave pending requests untouched so the waiting device can continue polling
		return model.User{}, "", request.Status, nil
	case RequestStatusDenied:
		return model.User{}, "", request.Status, &common.DeviceLoginDeniedError{}
	case RequestStatusApproved:
	default:
		return model.User{}, "", "", &common.DeviceLoginRequestInvalidOrExpiredError{}
	}

	if request.UserID == nil || request.User.ID == "" {
		return model.User{}, "", "", &common.DeviceLoginRequestInvalidOrExpiredError{}
	}
	if request.User.Disabled {
		return model.User{}, "", "", &common.UserDisabledError{}
	}

	// Mint the session with login-code semantics because the waiting device did not perform WebAuthn
	accessToken, err := s.signer.GenerateAccessToken(request.User, authenticationMethodOneTimePassword)
	if err != nil {
		return model.User{}, "", "", err
	}

	result := tx.
		WithContext(ctx).
		Where("id = ? AND status = ?", request.ID, RequestStatusApproved).
		Delete(&Request{})
	if result.Error != nil {
		return model.User{}, "", "", result.Error
	}
	if result.RowsAffected != 1 {
		return model.User{}, "", "", &common.DeviceLoginRequestInvalidOrExpiredError{}
	}

	s.auditLog.Create(ctx, model.AuditLogEventRemoteSignIn, ipAddress, userAgent, request.User.ID, model.AuditLogData{}, tx)

	err = tx.Commit().Error
	if err != nil {
		return model.User{}, "", "", err
	}

	return request.User, accessToken, request.Status, nil
}

func newUserCode() (string, error) {
	randomCode, err := utils.GenerateRandomUppercaseUnambiguousString(codeRandomLength)
	if err != nil {
		return "", err
	}
	return codePrefix + randomCode, nil
}
