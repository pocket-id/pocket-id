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
	"github.com/pocket-id/pocket-id/backend/internal/auditlogs"
	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
)

// authenticationMethodOneTimePassword identifies one-time password/code authentication
// It must match the value emitted by the JWT service in the access token's "amr" claim
const (
	authenticationMethodOneTimePassword = "otp"
	shortTokenLength                    = 6
	longTokenLength                     = 12
	deviceTokenLength                   = 16
)

// TokenStore is the minimal interface needed to persist a one-time access token in the actor state store.
// It's satisfied by both *actor.Service (used by the running application) and *local.Host (used by CLI commands, which don't run the full actor host).
type TokenStore interface {
	SetState(ctx context.Context, actorType string, actorID string, state any, opts *actor.SetStateOpts) error
}

type Service struct {
	db           *gorm.DB
	actorService *actor.Service
	userProvider UserProvider
	sessions     SessionCreator
	auditLog     AuditLogger
	emailSender  EmailSender
}

func newService(deps Dependencies, actorService *actor.Service) *Service {
	return &Service{
		db:           deps.DB,
		actorService: actorService,
		userProvider: deps.UserProvider,
		sessions:     deps.Sessions,
		auditLog:     deps.AuditLog,
		emailSender:  deps.EmailSender,
	}
}

func (s *Service) RequestOneTimeAccessEmailAsAdmin(ctx context.Context, dbConfig *appconfig.AppConfigModel, userID string, ttl time.Duration, origin auditlogs.Origin) error {
	if !dbConfig.EmailOneTimeAccessAsAdminEnabled.IsTrue() {
		return apperror.OneTimeAccessDisabled()
	}

	return s.requestOneTimeAccessEmailInternal(ctx, userID, "", ttl, nil, dbConfig, origin)
}

func (s *Service) RequestOneTimeAccessEmailAsUnauthenticatedUser(ctx context.Context, dbConfig *appconfig.AppConfigModel, email, redirectPath string, origin auditlogs.Origin) (string, error) {
	if !dbConfig.EmailOneTimeAccessAsUnauthenticatedEnabled.IsTrue() {
		return "", apperror.OneTimeAccessDisabled()
	}

	deviceToken, err := utils.GenerateRandomAlphanumericString(deviceTokenLength)
	if err != nil {
		return "", err
	}

	// The work for a registered email runs in the background, so neither the response time nor its outcome reveals whether the email is registered
	go func() {
		innerCtx := context.WithoutCancel(ctx)

		var userID string
		innerErr := s.db.WithContext(innerCtx).Model(&model.User{}).Select("id").Where("email = ?", email).First(&userID).Error
		if innerErr == nil {
			innerErr = s.requestOneTimeAccessEmailInternal(innerCtx, userID, redirectPath, 15*time.Minute, &deviceToken, dbConfig, origin)
		}
		if innerErr != nil && !errors.Is(innerErr, gorm.ErrRecordNotFound) {
			slog.ErrorContext(innerCtx, "Failed to create the requested one-time access token", slog.Any("error", innerErr))
		}
	}()

	return deviceToken, nil
}

func (s *Service) requestOneTimeAccessEmailInternal(ctx context.Context, userID, redirectPath string, ttl time.Duration, deviceToken *string, dbConfig *appconfig.AppConfigModel, origin auditlogs.Origin) error {
	// Load the user to ensure it exists and has an email address
	user, err := s.userProvider.GetUser(ctx, userID)
	if err != nil {
		return err
	}

	if user.Email == nil {
		return apperror.UserEmailNotSet()
	}

	oneTimeAccessToken, err := StoreToken(ctx, s.actorService, user.ID, ttl, deviceToken)
	if err != nil {
		return err
	}
	s.auditLog.Record(ctx, auditlogs.EventLoginCodeCreated, origin, user.ID, auditlogs.Data{"delivery": "email"}, nil)

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
			durationToString(ttl),
		)
		if innerErr != nil {
			slog.ErrorContext(innerCtx, "Failed to send one-time access token email", slog.Any("error", innerErr), slog.String("address", *user.Email))
			return
		}
	}()

	return nil
}

func (s *Service) CreateToken(ctx context.Context, userID string, ttl time.Duration, origin auditlogs.Origin) (token string, err error) {
	// Load the user to ensure it exists
	_, err = s.userProvider.GetUser(ctx, userID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return "", apperror.UserNotFound()
	} else if err != nil {
		return "", err
	}

	token, err = StoreToken(ctx, s.actorService, userID, ttl, nil)
	if err != nil {
		return "", err
	}
	s.auditLog.Record(ctx, auditlogs.EventLoginCodeCreated, origin, userID, auditlogs.Data{"delivery": "link"}, nil)

	return token, nil
}

func (s *Service) ExchangeToken(ctx context.Context, dbConfig *appconfig.AppConfigModel, token, deviceToken, ipAddress, userAgent, browserToken string, remember bool) (model.User, model.LoginTokens, error) {
	token = utils.NormalizeUnambiguousString(token)

	// Consume the token by invoking its actor: this atomically validates it and, if valid, deletes it.
	// It must happen outside of a DB transaction, since invoking an actor while a transaction is open would deadlock on SQLite.
	res, err := s.actorService.Invoke(ctx, TokenActorType, token, tokenMethodConsume, tokenConsumeRequest{
		DeviceToken: deviceToken,
	})
	if err != nil {
		return model.User{}, model.LoginTokens{}, fmt.Errorf("error invoking one-time access token actor: %w", err)
	}

	var consumeRes tokenConsumeResponse
	err = res.Decode(&consumeRes)
	if err != nil {
		return model.User{}, model.LoginTokens{}, fmt.Errorf("error decoding one-time access token actor response: %w", err)
	}

	switch consumeRes.Status {
	case tokenConsumeNotFound:
		return model.User{}, model.LoginTokens{}, apperror.TokenInvalidOrExpired()
	case tokenConsumeOK:
		// All good, continue below
	default:
		return model.User{}, model.LoginTokens{}, fmt.Errorf("unexpected status from one-time access token actor: %s", consumeRes.Status)
	}

	// The token has now been consumed. From this point on, if we hit an error we compensate by restoring the token (this is best-effort).
	user, tokens, err := s.completeTokenExchange(ctx, dbConfig, consumeRes.State, ipAddress, userAgent, browserToken, remember)
	if err != nil {
		s.restoreToken(ctx, token, consumeRes.State)
		return model.User{}, model.LoginTokens{}, err
	}

	return user, tokens, nil
}

// completeTokenExchange performs the work that follows consuming a token: loading the user, validating it, and starting a session.
func (s *Service) completeTokenExchange(ctx context.Context, dbConfig *appconfig.AppConfigModel, state TokenState, ipAddress, userAgent, browserToken string, remember bool) (model.User, model.LoginTokens, error) {
	var user model.User
	err := s.db.
		WithContext(ctx).
		Where("id = ?", state.UserID).
		First(&user).
		Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return model.User{}, model.LoginTokens{}, apperror.TokenInvalidOrExpired()
	} else if err != nil {
		return model.User{}, model.LoginTokens{}, err
	}

	if user.Disabled {
		return model.User{}, model.LoginTokens{}, apperror.UserDisabled()
	}

	sessionToken, err := s.sessions.Create(ctx, user.ID, authenticationMethodOneTimePassword, ipAddress, userAgent, dbConfig.SessionDuration.AsDurationMinutes(), remember)
	if err != nil {
		return model.User{}, model.LoginTokens{}, err
	}

	// Recognize the receiving browser after the login code has been consumed and the user validated
	signIn := s.auditLog.CreateSignIn(ctx, auditlogs.EventOneTimeAccessTokenSignIn, ipAddress, userAgent, user.ID, browserToken, s.db, dbConfig.EmailLoginNotificationMode)
	s.auditLog.SendSignInNotification(ctx, signIn)

	return user, model.LoginTokens{SessionToken: sessionToken, KnownBrowserToken: signIn.KnownBrowserToken}, nil
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
// A non-nil device token binds the token to the device that requested it.
func StoreToken(ctx context.Context, store TokenStore, userID string, ttl time.Duration, deviceToken *string) (token string, err error) {
	token, err = generateToken(ttl)
	if err != nil {
		return "", err
	}

	now := time.Now().Round(time.Second)
	state := TokenState{
		UserID:      userID,
		DeviceToken: deviceToken,
		ExpiresAt:   now.Add(ttl),
	}

	err = store.SetState(ctx, TokenActorType, token, state, &actor.SetStateOpts{TTL: ttl})
	if err != nil {
		return "", fmt.Errorf("error saving one-time access token state: %w", err)
	}

	return token, nil
}

// generateToken generates the random token value for a one-time access token.
func generateToken(ttl time.Duration) (string, error) {
	// Use the shorter format only for codes that expire within 15 minutes
	tokenLength := longTokenLength
	if ttl <= 15*time.Minute {
		tokenLength = shortTokenLength
	}

	return utils.GenerateRandomUnambiguousString(tokenLength)
}

// durationToString converts a duration to a human-readable string for the one-time access email
// It shows minutes and hours below a day, and days and hours from a day on
func durationToString(duration time.Duration) string {
	// For a duration less than a day
	if duration < 24*time.Hour {
		hours := int(duration.Hours())
		mins := int(duration.Minutes()) % 60

		switch hours {
		case 0:
			return fmt.Sprintf("%d minutes", mins)
		case 1:
			if mins == 0 {
				return "1 hour"
			}
			return fmt.Sprintf("1 hour and %d minutes", mins)
		default:
			if mins == 0 {
				return fmt.Sprintf("%d hours", hours)
			}
			return fmt.Sprintf("%d hours and %d minutes", hours, mins)
		}
	} else {
		// For durations of a day or more
		days := int(duration.Hours() / 24)
		hours := int(duration.Hours()) % 24

		switch hours {
		case 0:
			if days == 1 {
				return "1 day"
			}
			return fmt.Sprintf("%d days", days)
		case 1:
			if days == 1 {
				return "1 day and 1 hour"
			}
			return fmt.Sprintf("%d days and 1 hour", days)
		default:
			if days == 1 {
				return fmt.Sprintf("1 day and %d hours", hours)
			}
			return fmt.Sprintf("%d days and %d hours", days, hours)
		}
	}
}
