package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"strings"
	"time"

	"github.com/italypaleale/francis/actor"
	"github.com/italypaleale/francis/host/local"
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/appconfig"
	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
	"github.com/pocket-id/pocket-id/backend/internal/utils/email"
)

// OneTimeAccessTokenStore is the minimal interface needed to persist a one-time access token in the actor state store.
// It's satisfied by both *actor.Service (used by the running application) and *local.Host (used by CLI commands, which don't run the full actor host).
type OneTimeAccessTokenStore interface {
	SetState(ctx context.Context, actorType string, actorID string, state any, opts *actor.SetStateOpts) error
}

type OneTimeAccessService struct {
	db              *gorm.DB
	actorService    *actor.Service
	userService     *UserService
	jwtService      *JwtService
	auditLogService *AuditLogService
	emailService    *EmailService
}

func NewOneTimeAccessService(actors *local.Host, db *gorm.DB, userService *UserService, jwtService *JwtService, auditLogService *AuditLogService, emailService *EmailService) (*OneTimeAccessService, error) {
	err := actors.RegisterActor(OneTimeAccessTokenActorType, NewOneTimeAccessTokenActor)
	if err != nil {
		return nil, fmt.Errorf("error registering the %s actor: %w", OneTimeAccessTokenActorType, err)
	}

	return &OneTimeAccessService{
		db:              db,
		actorService:    actors.Service(),
		userService:     userService,
		jwtService:      jwtService,
		auditLogService: auditLogService,
		emailService:    emailService,
	}, nil
}

func (s *OneTimeAccessService) RequestOneTimeAccessEmailAsAdmin(ctx context.Context, dbConfig *appconfig.AppConfigModel, userID string, ttl time.Duration) error {
	if !dbConfig.EmailOneTimeAccessAsAdminEnabled.IsTrue() {
		return &common.OneTimeAccessDisabledError{}
	}

	_, err := s.requestOneTimeAccessEmailInternal(ctx, userID, "", ttl, false, dbConfig)
	return err
}

func (s *OneTimeAccessService) RequestOneTimeAccessEmailAsUnauthenticatedUser(ctx context.Context, dbConfig *appconfig.AppConfigModel, userID, redirectPath string) (string, error) {
	if !dbConfig.EmailOneTimeAccessAsUnauthenticatedEnabled.IsTrue() {
		return "", &common.OneTimeAccessDisabledError{}
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

func (s *OneTimeAccessService) requestOneTimeAccessEmailInternal(ctx context.Context, userID, redirectPath string, ttl time.Duration, withDeviceToken bool, dbConfig *appconfig.AppConfigModel) (*string, error) {
	// Load the user to ensure it exists and has an email address
	user, err := s.userService.GetUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	if user.Email == nil {
		return nil, &common.UserEmailNotSetError{}
	}

	oneTimeAccessToken, deviceToken, err := StoreOneTimeAccessToken(ctx, s.actorService, user.ID, ttl, withDeviceToken)
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

		errInternal := SendEmail(innerCtx, s.emailService, dbConfig, email.Address{
			Name:  user.FullName(),
			Email: *user.Email,
		}, OneTimeAccessTemplate, &OneTimeAccessTemplateData{
			Code:              oneTimeAccessToken,
			LoginLink:         link,
			LoginLinkWithCode: linkWithCode,
			ExpirationString:  utils.DurationToString(ttl),
		})
		if errInternal != nil {
			slog.ErrorContext(innerCtx, "Failed to send one-time access token email", slog.Any("error", errInternal), slog.String("address", *user.Email))
			return
		}
	}()

	return deviceToken, nil
}

func (s *OneTimeAccessService) CreateOneTimeAccessToken(ctx context.Context, userID string, ttl time.Duration) (token string, err error) {
	// Load the user to ensure it exists
	_, err = s.userService.GetUser(ctx, userID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return "", &common.UserNotFoundError{}
	} else if err != nil {
		return "", err
	}

	token, _, err = StoreOneTimeAccessToken(ctx, s.actorService, userID, ttl, false)
	if err != nil {
		return "", err
	}

	return token, nil
}

func (s *OneTimeAccessService) ExchangeOneTimeAccessToken(ctx context.Context, dbConfig *appconfig.AppConfigModel, token, deviceToken, ipAddress, userAgent string) (model.User, string, error) {
	// Consume the token by invoking its actor: this atomically validates it and, if valid, deletes it.
	// It must happen outside of a DB transaction, since invoking an actor while a transaction is open would deadlock on SQLite.
	res, err := s.actorService.Invoke(ctx, OneTimeAccessTokenActorType, token, oneTimeAccessTokenMethodConsume, oneTimeAccessConsumeRequest{
		DeviceToken: deviceToken,
	})
	if err != nil {
		return model.User{}, "", fmt.Errorf("error invoking one-time access token actor: %w", err)
	}

	var consumeRes oneTimeAccessConsumeResponse
	err = res.Decode(&consumeRes)
	if err != nil {
		return model.User{}, "", fmt.Errorf("error decoding one-time access token actor response: %w", err)
	}

	switch consumeRes.Status {
	case oneTimeAccessConsumeNotFound:
		return model.User{}, "", &common.TokenInvalidOrExpiredError{}
	case oneTimeAccessConsumeDeviceMismatch:
		return model.User{}, "", &common.DeviceCodeInvalid{}
	case oneTimeAccessConsumeOK:
		// All good, continue below
	default:
		return model.User{}, "", fmt.Errorf("unexpected status from one-time access token actor: %s", consumeRes.Status)
	}

	// The token has now been consumed. From this point on, if we hit an error we compensate by restoring the token (this is best-effort).
	user, accessToken, err := s.completeOneTimeAccessTokenExchange(ctx, dbConfig, consumeRes.State, ipAddress, userAgent)
	if err != nil {
		s.restoreOneTimeAccessToken(ctx, token, consumeRes.State)
		return model.User{}, "", err
	}

	return user, accessToken, nil
}

// completeOneTimeAccessTokenExchange performs the work that follows consuming a token: loading the user, validating it, and issuing an access token.
func (s *OneTimeAccessService) completeOneTimeAccessTokenExchange(ctx context.Context, dbConfig *appconfig.AppConfigModel, state oneTimeAccessTokenState, ipAddress, userAgent string) (model.User, string, error) {
	var user model.User
	err := s.db.
		WithContext(ctx).
		Where("id = ?", state.UserID).
		First(&user).
		Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return model.User{}, "", &common.TokenInvalidOrExpiredError{}
	} else if err != nil {
		return model.User{}, "", err
	}

	if user.Disabled {
		return model.User{}, "", &common.UserDisabledError{}
	}

	accessToken, err := s.jwtService.GenerateAccessToken(
		user,
		AuthenticationMethodOneTimePassword,
		dbConfig.SessionDuration.AsDurationMinutes(),
	)
	if err != nil {
		return model.User{}, "", err
	}

	s.auditLogService.Create(
		ctx, model.AuditLogEventOneTimeAccessTokenSignIn,
		ipAddress, userAgent,
		user.ID,
		model.AuditLogData{},
		s.db,
	)

	return user, accessToken, nil
}

// restoreOneTimeAccessToken restores a token that was consumed but whose exchange could not be completed.
// It's a best-effort compensation: if it fails (or the process crashes before it runs) we accept that the token was consumed unnecessarily.
func (s *OneTimeAccessService) restoreOneTimeAccessToken(parentCtx context.Context, token string, state oneTimeAccessTokenState) {
	// Use a context that is not canceled when the original request ends
	ctx, cancel := context.WithTimeout(context.WithoutCancel(parentCtx), 10*time.Second)
	defer cancel()

	_, err := s.actorService.Invoke(ctx, OneTimeAccessTokenActorType, token, oneTimeAccessTokenMethodRestore, state)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to restore one-time access token after a failed exchange", slog.Any("error", err))
	}
}

// StoreOneTimeAccessToken generates a new one-time access token and persists it in the actor state store, with a TTL matching its lifetime.
// It returns the token value and, when requested, the associated device token.
func StoreOneTimeAccessToken(ctx context.Context, store OneTimeAccessTokenStore, userID string, ttl time.Duration, withDeviceToken bool) (token string, deviceToken *string, err error) {
	token, deviceToken, err = generateOneTimeAccessToken(ttl, withDeviceToken)
	if err != nil {
		return "", nil, err
	}

	now := time.Now().Round(time.Second)
	state := oneTimeAccessTokenState{
		UserID:      userID,
		DeviceToken: deviceToken,
		ExpiresAt:   now.Add(ttl),
	}

	err = store.SetState(ctx, OneTimeAccessTokenActorType, token, state, &actor.SetStateOpts{TTL: ttl})
	if err != nil {
		return "", nil, fmt.Errorf("error saving one-time access token state: %w", err)
	}

	return token, deviceToken, nil
}

// generateOneTimeAccessToken generates the random token value (and optional device token) for a one-time access token.
func generateOneTimeAccessToken(ttl time.Duration, withDeviceToken bool) (token string, deviceToken *string, err error) {
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
