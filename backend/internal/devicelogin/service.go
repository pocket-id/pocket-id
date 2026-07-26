package devicelogin

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/italypaleale/francis/actor"

	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/dto"
	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
	cryptoutils "github.com/pocket-id/pocket-id/backend/internal/utils/crypto"
)

const (
	RequestDuration        = 15 * time.Minute
	PollingInterval        = 3
	longPollingDuration    = 25 * time.Second
	actorPollingInterval   = 250 * time.Millisecond
	codePrefix             = "P"
	codeRandomLength       = 7
	reauthenticationMaxAge = time.Minute
	// authenticationMethodOneTimePassword identifies the login-code-equivalent AMR used on the waiting device
	authenticationMethodOneTimePassword = "otp"
)

type Service struct {
	actService *actor.Service
	actorIDKey []byte
	auditLog   AuditLogger
}

type VerificationInfo struct {
	UserCode  string
	Device    string
	IPAddress string
	ExpiresAt datatype.DateTime
}

func NewService(actService *actor.Service, actorIDKey []byte, auditLog AuditLogger) *Service {
	return &Service{
		actService: actService,
		actorIDKey: actorIDKey,
		auditLog:   auditLog,
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
		actorID, actorIDErr := s.actorIDForCode(code)
		if actorIDErr != nil {
			return Request{}, "", actorIDErr
		}

		result, invokeErr := s.invoke(ctx, actorID, requestActorMethodCreate, requestActorCreateInput{
			Code:            code,
			DeviceTokenHash: deviceTokenHash,
			IPAddress:       ipAddress,
			UserAgent:       userAgent,
		})
		if invokeErr != nil {
			return Request{}, "", invokeErr
		}
		if result.Code == requestActorResultCollision {
			continue
		}
		if err = actorResultError(result.Code); err != nil {
			return Request{}, "", err
		}

		return Request{
			ID:        actorID,
			Code:      code,
			Status:    result.Status,
			ExpiresAt: datatype.DateTime(result.ExpiresAt),
		}, deviceToken, nil
	}

	return Request{}, "", errors.New("failed to generate a unique device login code")
}

func (s *Service) Inspect(ctx context.Context, code string) (VerificationInfo, error) {
	actorID, err := s.actorIDForCode(code)
	if err != nil {
		return VerificationInfo{}, err
	}

	// Read the pending actor state without taking an exclusive actor turn
	result, err := s.peek(ctx, actorID, requestActorMethodInspect, nil)
	if err != nil {
		return VerificationInfo{}, err
	}
	if err = actorResultError(result.Code); err != nil {
		return VerificationInfo{}, err
	}

	return VerificationInfo{
		UserCode:  result.UserCode,
		Device:    s.auditLog.DeviceStringFromUserAgent(result.UserAgent),
		IPAddress: result.IPAddress,
		ExpiresAt: datatype.DateTime(result.ExpiresAt),
	}, nil
}

func (s *Service) Decide(ctx context.Context, code, decision, userID, reauthenticationToken string) error {
	actorID, err := s.actorIDForCode(code)
	if err != nil {
		return err
	}

	// Let the actor serialize the decision with every competing exchange
	result, err := s.invoke(ctx, actorID, requestActorMethodDecide, requestActorDecisionInput{
		Decision:              decision,
		UserID:                userID,
		ReauthenticationToken: reauthenticationToken,
	})
	if err != nil {
		return err
	}
	return actorResultError(result.Code)
}

func (s *Service) Exchange(ctx context.Context, requestID, deviceToken, ipAddress, userAgent string, sessionDuration time.Duration) (dto.UserDto, string, RequestStatus, error) {
	if deviceToken == "" || !validRequestActorID(requestID) {
		return dto.UserDto{}, "", "", &common.DeviceLoginRequestInvalidOrExpiredError{}
	}

	deviceTokenHash := utils.CreateSha256Hash(deviceToken)
	preflightStatus, err := s.preflightExchange(ctx, requestID, deviceTokenHash)
	if err != nil {
		return dto.UserDto{}, "", preflightStatus, err
	}
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
		if resultErr := actorResultError(result.Code); resultErr != nil {
			return dto.UserDto{}, "", result.Status, resultErr
		}

		switch result.Status {
		case RequestStatusApproved:
			exchange, invokeErr := s.invoke(ctx, requestID, requestActorMethodExchange, requestActorExchangeInput{
				DeviceTokenHash: deviceTokenHash,
				IPAddress:       ipAddress,
				UserAgent:       userAgent,
				SessionDuration: sessionDuration,
			})
			if invokeErr != nil {
				return dto.UserDto{}, "", "", invokeErr
			}
			if resultErr := actorResultError(exchange.Code); resultErr != nil {
				return dto.UserDto{}, "", exchange.Status, resultErr
			}
			return exchange.User, exchange.AccessToken, exchange.Status, nil
		case RequestStatusPending:
		case RequestStatusDenied:
			return dto.UserDto{}, "", result.Status, &common.DeviceLoginDeniedError{}
		default:
			return dto.UserDto{}, "", "", &common.DeviceLoginRequestInvalidOrExpiredError{}
		}

		select {
		case <-ticker.C:
		case <-timeout.C:
			return dto.UserDto{}, "", RequestStatusPending, nil
		case <-ctx.Done():
			return dto.UserDto{}, "", "", ctx.Err()
		}
	}
}

// preflightExchange checks the request state before starting the long-polling exchange.
func (s *Service) preflightExchange(ctx context.Context, requestID, deviceTokenHash string) (RequestStatus, error) {
	// Read durable state once before activation so random public IDs cannot allocate actors
	var state requestActorState
	err := s.actService.GetState(ctx, requestActorType, requestID, &state)
	if errors.Is(err, actor.ErrStateNotFound) {
		return "", &common.DeviceLoginRequestInvalidOrExpiredError{}
	}
	if err != nil {
		return "", fmt.Errorf("failed to preflight device login actor state: %w", err)
	}
	if state.Code == "" || !constantTimeStringEqual(state.DeviceTokenHash, deviceTokenHash) {
		return "", &common.DeviceLoginRequestInvalidOrExpiredError{}
	}

	switch state.Status {
	case RequestStatusPending, RequestStatusApproved:
		return state.Status, nil
	case RequestStatusDenied:
		return state.Status, &common.DeviceLoginDeniedError{}
	default:
		return "", &common.DeviceLoginRequestInvalidOrExpiredError{}
	}
}

func (s *Service) actorIDForCode(code string) (string, error) {
	code = strings.ToUpper(strings.TrimSpace(code))
	derived, err := cryptoutils.DeriveKey(s.actorIDKey, "pocketid/device-login-request/"+code)
	if err != nil {
		return "", fmt.Errorf("failed to derive device login actor ID: %w", err)
	}
	return hex.EncodeToString(derived), nil
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
	if err := envelope.Decode(&result); err != nil {
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
		return &common.DeviceLoginRequestInvalidOrExpiredError{}
	case requestActorResultDenied:
		return &common.DeviceLoginDeniedError{}
	case requestActorResultReauthenticationRequired:
		return &common.ReauthenticationRequiredError{}
	case requestActorResultUserDisabled:
		return &common.UserDisabledError{}
	default:
		return fmt.Errorf("unsupported device login actor result %q", code)
	}
}

func validRequestActorID(actorID string) bool {
	if len(actorID) != 64 {
		return false
	}
	_, err := hex.DecodeString(actorID)
	return err == nil
}

func newUserCode() (string, error) {
	randomCode, err := utils.GenerateRandomUppercaseUnambiguousString(codeRandomLength)
	if err != nil {
		return "", err
	}
	return codePrefix + randomCode, nil
}
