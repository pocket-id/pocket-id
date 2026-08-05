package devicelogin

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/italypaleale/francis/actor"
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/apperror"
	"github.com/pocket-id/pocket-id/backend/internal/dto"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
)

const (
	RequestDuration        = 5 * time.Minute
	PollingInterval        = 3
	longPollingDuration    = 25 * time.Second
	actorPollingInterval   = 2 * time.Second
	codePrefix             = "P"
	codeRandomLength       = 7
	reauthenticationMaxAge = time.Minute
	// authenticationMethodOneTimePassword identifies the login-code-equivalent AMR used on the waiting device
	authenticationMethodOneTimePassword = "otp"
)

type Service struct {
	actService *actor.Service
	db         *gorm.DB
	signer     TokenService
	reauth     ReauthenticationTokenConsumer
	auditLog   AuditLogger
	ipLocator  IPLocationResolver
}

type VerificationInfo struct {
	UserCode  string
	Device    string
	IPAddress string
	Country   string
	City      string
	ExpiresAt datatype.DateTime
}

func NewService(actService *actor.Service, db *gorm.DB, signer TokenService, reauth ReauthenticationTokenConsumer, auditLog AuditLogger, ipLocator IPLocationResolver) *Service {
	return &Service{
		actService: actService,
		db:         db,
		signer:     signer,
		reauth:     reauth,
		auditLog:   auditLog,
		ipLocator:  ipLocator,
	}
}

func (s *Service) Create(ctx context.Context, ipAddress, userAgent string) (Request, string, error) {
	// Bind the public request to a separate high-entropy secret that never enters the QR code
	deviceToken, err := utils.GenerateRandomAlphanumericString(32)
	if err != nil {
		return Request{}, "", err
	}
	deviceTokenHash := utils.CreateSha256Hash(deviceToken)

	// Retry code generation because of the small but non-zero chance of a live actor collision
	for range 3 {
		code, codeErr := newUserCode()
		if codeErr != nil {
			return Request{}, "", codeErr
		}

		result, err := s.invoke(ctx, code, requestActorMethodCreate, requestActorCreateInput{
			Code:            code,
			DeviceTokenHash: deviceTokenHash,
			IPAddress:       ipAddress,
			UserAgent:       userAgent,
		})
		if err != nil {
			return Request{}, "", err
		}
		if result.Code == requestActorResultCollision {
			continue
		}
		err = actorResultError(result.Code)
		if err != nil {
			return Request{}, "", err
		}

		return Request{
			ID:        code,
			Code:      code,
			Status:    result.Status,
			ExpiresAt: datatype.DateTime(result.ExpiresAt),
		}, deviceToken, nil
	}

	return Request{}, "", errors.New("failed to generate a unique device login code")
}

func (s *Service) Inspect(ctx context.Context, code string) (VerificationInfo, error) {
	actorID := normalizeUserCode(code)

	// Read the pending actor state without taking an exclusive actor turn
	result, err := s.peek(ctx, actorID, requestActorMethodInspect, nil)
	if err != nil {
		return VerificationInfo{}, err
	}

	err = actorResultError(result.Code)
	if err != nil {
		return VerificationInfo{}, err
	}

	country, city, err := s.ipLocator.GetLocationByIP(ctx, result.IPAddress)
	if err != nil {
		slog.WarnContext(ctx, "Failed to get device login request IP location", slog.String("ip", result.IPAddress), slog.Any("error", err))
	}

	return VerificationInfo{
		UserCode:  result.UserCode,
		Device:    s.auditLog.DeviceStringFromUserAgent(result.UserAgent),
		IPAddress: result.IPAddress,
		Country:   country,
		City:      city,
		ExpiresAt: datatype.DateTime(result.ExpiresAt),
	}, nil
}

func (s *Service) Decide(ctx context.Context, code, decision, userID, reauthenticationToken string) error {
	actorID := normalizeUserCode(code)

	// Consume the fresh passkey proof outside the actor before approving the request
	if decision == "approve" {
		if err := s.consumeReauthenticationProof(ctx, reauthenticationToken, userID); err != nil {
			return err
		}
	}

	// Let the actor serialize the decision with every competing exchange
	result, err := s.invoke(ctx, actorID, requestActorMethodDecide, requestActorDecisionInput{
		Decision: decision,
		UserID:   userID,
	})
	if err != nil {
		return err
	}
	return actorResultError(result.Code)
}

func (s *Service) Exchange(ctx context.Context, requestID, deviceToken, ipAddress, userAgent string, sessionDuration time.Duration) (dto.UserDto, string, RequestStatus, error) {
	if requestID == "" || deviceToken == "" || sessionDuration <= 0 {
		return dto.UserDto{}, "", "", apperror.DeviceLoginRequestInvalidOrExpired()
	}

	deviceTokenHash := utils.CreateSha256Hash(deviceToken)
	timeout := time.NewTimer(longPollingDuration)
	defer timeout.Stop()
	ticker := time.NewTicker(actorPollingInterval)
	defer ticker.Stop()

	for {
		// Poll the actor's activation cache so the long-lived HTTP request does not repeatedly query the database
		result, err := s.peek(ctx, requestID, requestActorMethodPoll, requestActorPollInput{DeviceTokenHash: deviceTokenHash})
		if err != nil {
			return dto.UserDto{}, "", "", err
		}

		err = actorResultError(result.Code)
		if err != nil {
			return dto.UserDto{}, "", result.Status, err
		}

		switch result.Status {
		case RequestStatusApproved:
			// Validate the approved user before consuming so lookup failures leave the request untouched
			user, userDTO, err := s.loadExchangeUser(ctx, result.UserID)
			if err != nil {
				return dto.UserDto{}, "", result.Status, err
			}

			// Consume inside the actor so only one concurrent exchange can mint a token
			consume, err := s.invoke(ctx, requestID, requestActorMethodConsume, requestActorConsumeInput{
				DeviceTokenHash: deviceTokenHash,
			})
			if err != nil {
				return dto.UserDto{}, "", "", err
			}

			err = actorResultError(consume.Code)
			if err != nil {
				return dto.UserDto{}, "", consume.Status, err
			}

			// Mint the session with login-code semantics because the waiting device did not perform WebAuthn
			accessToken, err := s.signer.GenerateAccessToken(user, authenticationMethodOneTimePassword, sessionDuration)
			if err != nil {
				return dto.UserDto{}, "", consume.Status, err
			}

			// Record the successful remote sign-in after the request has been consumed
			_, created := s.auditLog.Create(ctx, model.AuditLogEventRemoteSignIn, ipAddress, userAgent, user.ID, model.AuditLogData{}, s.db)
			if !created {
				return dto.UserDto{}, "", consume.Status, errors.New("failed to create device login audit log")
			}

			return userDTO, accessToken, consume.Status, nil
		case RequestStatusPending:
			// no-op
		case RequestStatusDenied:
			return dto.UserDto{}, "", result.Status, apperror.DeviceLoginDenied()
		default:
			return dto.UserDto{}, "", "", apperror.DeviceLoginRequestInvalidOrExpired()
		}

		select {
		case <-ticker.C:
			// no-op
		case <-timeout.C:
			return dto.UserDto{}, "", RequestStatusPending, nil
		case <-ctx.Done():
			return dto.UserDto{}, "", "", ctx.Err()
		}
	}
}

func (s *Service) consumeReauthenticationProof(ctx context.Context, token, userID string) error {
	if token == "" {
		return apperror.ReauthenticationRequired()
	}

	tx := s.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return tx.Error
	}
	defer tx.Rollback()

	reauthenticatedAt, err := s.reauth.ConsumeReauthenticationToken(ctx, tx, token, userID)
	if err != nil {
		if apperror.IsCode(err, apperror.CodeReauthenticationRequired) {
			return apperror.ReauthenticationRequired()
		}
		return err
	}
	if time.Since(reauthenticatedAt) > reauthenticationMaxAge {
		return apperror.ReauthenticationRequired()
	}

	if err = tx.Commit().Error; err != nil {
		return fmt.Errorf("error committing database transaction: %w", err)
	}
	return nil
}

func (s *Service) loadExchangeUser(ctx context.Context, userID string) (model.User, dto.UserDto, error) {
	var user model.User
	err := s.db.WithContext(ctx).First(&user, "id = ?", userID).Error
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		return model.User{}, dto.UserDto{}, apperror.DeviceLoginRequestInvalidOrExpired()
	case err != nil:
		return model.User{}, dto.UserDto{}, err
	case user.Disabled:
		return model.User{}, dto.UserDto{}, apperror.UserDisabled()
	}

	var userDTO dto.UserDto
	if err = dto.MapStruct(user, &userDTO); err != nil {
		return model.User{}, dto.UserDto{}, fmt.Errorf("failed to map exchanged device login user: %w", err)
	}

	return user, userDTO, nil
}

func normalizeUserCode(code string) string {
	code = strings.ToUpper(strings.TrimSpace(code))
	return utils.NormalizeUnambiguousString(code)
}

func (s *Service) invoke(ctx context.Context, actorID, method string, input any) (requestActorResult, error) {
	envelope, err := s.actService.Invoke(ctx, requestActorType, actorID, method, input)
	if err != nil {
		return requestActorResult{}, err
	}

	return decodeActorResult(envelope)
}

func (s *Service) peek(ctx context.Context, actorID, method string, input any) (requestActorResult, error) {
	envelope, err := s.actService.Peek(ctx, requestActorType, actorID, method, input)
	if err != nil {
		return requestActorResult{}, err
	}

	return decodeActorResult(envelope)
}

func decodeActorResult(envelope actor.Envelope) (requestActorResult, error) {
	if envelope == nil {
		return requestActorResult{}, errors.New("device login actor returned an empty response")
	}

	var result requestActorResult
	err := envelope.Decode(&result)
	if err != nil {
		return requestActorResult{}, fmt.Errorf("failed to decode device login actor response: %w", err)
	}
	return result, nil
}

func actorResultError(code requestActorResultCode) error {
	switch code {
	case requestActorResultNone:
		return nil
	case requestActorResultCollision:
		return errors.New("unexpected live device login actor collision")
	case requestActorResultInvalid:
		return apperror.DeviceLoginRequestInvalidOrExpired()
	case requestActorResultDenied:
		return apperror.DeviceLoginDenied()
	default:
		return fmt.Errorf("unsupported device login actor result %q", code)
	}
}

func newUserCode() (string, error) {
	randomCode, err := utils.GenerateRandomUppercaseUnambiguousString(codeRandomLength)
	if err != nil {
		return "", err
	}

	return codePrefix + randomCode, nil
}
