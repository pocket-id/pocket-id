package onetimeaccess

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"strings"
	"time"

	"github.com/italypaleale/francis/actor"
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/appconfig"
	"github.com/pocket-id/pocket-id/backend/internal/apperror"
	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
)

// authenticationMethodOneTimePassword identifies one-time password/code authentication
// It must match the value emitted by the JWT service in the access token's "amr" claim
const authenticationMethodOneTimePassword = "otp"

// TokenStore is the minimal interface needed to persist a one-time access token in the actor state store.
// It's satisfied by both *actor.Service (used by the running application) and *local.Host (used by CLI commands, which don't run the full actor host).
type TokenStore interface {
	SetState(ctx context.Context, actorType string, actorID string, state any, opts *actor.SetStateOpts) error
}

type Service struct {
	db           *gorm.DB
	actorService *actor.Service
	userProvider UserProvider
	signer       TokenService
	auditLog     AuditLogger
	emailSender  EmailSender
}

func newService(deps Dependencies, actorService *actor.Service) *Service {
	return &Service{
		db:           deps.DB,
		actorService: actorService,
		userProvider: deps.UserProvider,
		signer:       deps.Signer,
		auditLog:     deps.AuditLog,
		emailSender:  deps.EmailSender,
	}
}

func (s *Service) RequestOneTimeAccessEmailAsAdmin(ctx context.Context, dbConfig *appconfig.AppConfigModel, userID string, ttl time.Duration) error {
	if !dbConfig.EmailOneTimeAccessAsAdminEnabled.IsTrue() {
		return apperror.OneTimeAccessDisabled()
	}

	_, err := s.requestOneTimeAccessEmailInternal(ctx, userID, "", ttl, false, dbConfig)
	return err
}

func (s *Service) RequestOneTimeAccessEmailAsUnauthenticatedUser(ctx context.Context, dbConfig *appconfig.AppConfigModel, userID, redirectPath string) (string, error) {
	if !dbConfig.EmailOneTimeAccessAsUnauthenticatedEnabled.IsTrue() {
		return "", apperror.OneTimeAccessDisabled()
	}

	var userId string
	err := s.db.Model(&model.User{}).Select("id").Where("email = ?", userID).First(&userId).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		// Do not return error if user not found to prevent email enumeration
		return "", nil
	} else if err != nil {
		return "", err
	}

	deviceToken, err := s.requestOneTimeAccessEmailInternal(ctx, userId, redirectPath, 15*time.Minute, true, dbConfig)
	if err != nil {
		return "", err
	} else if deviceToken == nil {
		return "", errors.New("device token expected but not returned")
	}

	return *deviceToken, nil
}

func (s *Service) requestOneTimeAccessEmailInternal(ctx context.Context, userID, redirectPath string, ttl time.Duration, withDeviceToken bool, dbConfig *appconfig.AppConfigModel) (*string, error) {
	// Load the user to ensure it exists and has an email address
	user, err := s.userProvider.GetUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	if user.Email == nil {
		return nil, apperror.UserEmailNotSet()
	}

	oneTimeAccessToken, deviceToken, err := StoreToken(ctx, s.actorService, user.ID, ttl, withDeviceToken)
	if err != nil {
		return nil, err
	}

	go func() {
		// This runs in background, so use a context without cancellation (or it would be stopped when the request ends)
		// We still want to have a context derived from the request's to carry over tracing info
		innerCtx := context.WithoutCancel(ctx)

		link := common.EnvConfig.AppURL + "/lc"
		linkWithCode := link + "/" + oneTimeAccessToken

		// Add redirect path to the link
		if strings.HasPrefix(redirectPath, "/") {
			encodedRedirectPath := url.QueryEscape(redirectPath)
			linkWithCode = linkWithCode + "?redirect=" + encodedRedirectPath
		}

		innerErr := s.emailSender.SendOneTimeAccessEmail(
			innerCtx,
			dbConfig,
			user.FullName(),
			*user.Email,
			oneTimeAccessToken,
			link,
			linkWithCode,
			utils.DurationToString(ttl),
		)
		if innerErr != nil {
			slog.ErrorContext(innerCtx, "Failed to send one-time access token email", slog.Any("error", innerErr), slog.String("address", *user.Email))
			return
		}
	}()

	return deviceToken, nil
}

func (s *Service) CreateToken(ctx context.Context, userID string, ttl time.Duration) (token string, err error) {
	// Load the user to ensure it exists
	_, err = s.userProvider.GetUser(ctx, userID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return "", apperror.UserNotFound()
	} else if err != nil {
		return "", err
	}

	token, _, err = StoreToken(ctx, s.actorService, userID, ttl, false)
	if err != nil {
		return "", err
	}

	return token, nil
}

func (s *Service) ExchangeToken(ctx context.Context, dbConfig *appconfig.AppConfigModel, token, deviceToken, ipAddress, userAgent string) (model.User, string, error) {
	token = utils.NormalizeUnambiguousString(token)

	// Consume the token by invoking its actor: this atomically validates it and, if valid, deletes it.
	// It must happen outside of a DB transaction, since invoking an actor while a transaction is open would deadlock on SQLite.
	res, err := s.actorService.Invoke(ctx, TokenActorType, token, tokenMethodConsume, tokenConsumeRequest{
		DeviceToken: deviceToken,
	})
	if err != nil {
		return model.User{}, "", fmt.Errorf("error invoking one-time access token actor: %w", err)
	}

	var consumeRes tokenConsumeResponse
	err = res.Decode(&consumeRes)
	if err != nil {
		return model.User{}, "", fmt.Errorf("error decoding one-time access token actor response: %w", err)
	}

	switch consumeRes.Status {
	case tokenConsumeNotFound:
		return model.User{}, "", apperror.TokenInvalidOrExpired()
	case tokenConsumeDeviceMismatch:
		return model.User{}, "", apperror.DeviceCodeInvalid()
	case tokenConsumeOK:
		// All good, continue below
	default:
		return model.User{}, "", fmt.Errorf("unexpected status from one-time access token actor: %s", consumeRes.Status)
	}

	// The token has now been consumed. From this point on, if we hit an error we compensate by restoring the token (this is best-effort).
	user, accessToken, err := s.completeTokenExchange(ctx, dbConfig, consumeRes.State, ipAddress, userAgent)
	if err != nil {
		s.restoreToken(ctx, token, consumeRes.State)
		return model.User{}, "", err
	}

	return user, accessToken, nil
}

// completeTokenExchange performs the work that follows consuming a token: loading the user, validating it, and issuing an access token.
func (s *Service) completeTokenExchange(ctx context.Context, dbConfig *appconfig.AppConfigModel, state TokenState, ipAddress, userAgent string) (model.User, string, error) {
	var user model.User
	err := s.db.
		WithContext(ctx).
		Where("id = ?", state.UserID).
		First(&user).
		Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return model.User{}, "", apperror.TokenInvalidOrExpired()
	} else if err != nil {
		return model.User{}, "", err
	}

	if user.Disabled {
		return model.User{}, "", apperror.UserDisabled()
	}

	accessToken, err := s.signer.GenerateAccessToken(
		user,
		authenticationMethodOneTimePassword,
		dbConfig.SessionDuration.AsDurationMinutes(),
	)
	if err != nil {
		return model.User{}, "", err
	}

	s.auditLog.Create(
		ctx, model.AuditLogEventOneTimeAccessTokenSignIn,
		ipAddress, userAgent,
		user.ID,
		model.AuditLogData{},
		s.db,
	)

	return user, accessToken, nil
}

// restoreToken restores a token that was consumed but whose exchange could not be completed.
// It's a best-effort compensation: if it fails (or the process crashes before it runs) we accept that the token was consumed unnecessarily.
func (s *Service) restoreToken(parentCtx context.Context, token string, state TokenState) {
	// Use a context that is not canceled when the original request ends
	ctx, cancel := context.WithTimeout(context.WithoutCancel(parentCtx), 10*time.Second)
	defer cancel()

	_, err := s.actorService.Invoke(ctx, TokenActorType, token, TokenMethodRestore, state)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to restore one-time access token after a failed exchange", slog.Any("error", err))
	}
}

// StoreToken generates a new one-time access token and persists it in the actor state store, with a TTL matching its lifetime.
// It returns the token value and, when requested, the associated device token.
func StoreToken(ctx context.Context, store TokenStore, userID string, ttl time.Duration, withDeviceToken bool) (token string, deviceToken *string, err error) {
	token, deviceToken, err = generateToken(ttl, withDeviceToken)
	if err != nil {
		return "", nil, err
	}

	now := time.Now().Round(time.Second)
	state := TokenState{
		UserID:      userID,
		DeviceToken: deviceToken,
		ExpiresAt:   now.Add(ttl),
	}

	err = store.SetState(ctx, TokenActorType, token, state, &actor.SetStateOpts{TTL: ttl})
	if err != nil {
		return "", nil, fmt.Errorf("error saving one-time access token state: %w", err)
	}

	return token, deviceToken, nil
}

// generateToken generates the random token value (and optional device token) for a one-time access token.
func generateToken(ttl time.Duration, withDeviceToken bool) (token string, deviceToken *string, err error) {
	// If expires at is less than 15 minutes, use a 6-character token instead of 16
	tokenLength := 16
	if ttl <= 15*time.Minute {
		tokenLength = 6
	}

	token, err = utils.GenerateRandomUnambiguousString(tokenLength)
	if err != nil {
		return "", nil, err
	}

	if withDeviceToken {
		dt, err := utils.GenerateRandomAlphanumericString(16)
		if err != nil {
			return "", nil, err
		}
		deviceToken = &dt
	}

	return token, deviceToken, nil
}
