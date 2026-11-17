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
	"github.com/pocket-id/pocket-id/backend/internal/authz"
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

	return s.requestOneTimeAccessEmailInternal(ctx, userID, "", ttl, nil, dbConfig, origin, true)
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
			innerErr = s.requestOneTimeAccessEmailInternal(innerCtx, userID, redirectPath, 15*time.Minute, &deviceToken, dbConfig, origin, false)
		}
		if innerErr != nil && !errors.Is(innerErr, gorm.ErrRecordNotFound) {
			slog.ErrorContext(innerCtx, "Failed to create the requested one-time access token", slog.Any("error", innerErr))
		}
	}()

	return deviceToken, nil
}

// byAdmin marks a code an admin requests for the user, as opposed to one the user requests for themselves
func (s *Service) requestOneTimeAccessEmailInternal(ctx context.Context, userID, redirectPath string, ttl time.Duration, deviceToken *string, dbConfig *appconfig.AppConfigModel, origin auditlogs.Origin, byAdmin bool) error {
	// Load the user to ensure it exists and has an email address
	user, err := s.userProvider.GetUser(ctx, userID)
	if err != nil {
		return err
	}

	if user.Email == nil {
		return apperror.UserEmailNotSet()
	}

	forbidAdmin := false
	if byAdmin {
		forbidAdmin, err = adminCodeRestriction(user, origin)
		if err != nil {
			return err
		}
	}

	oneTimeAccessToken, err := StoreToken(ctx, s.actorService, user.ID, ttl, deviceToken, forbidAdmin)
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
			user.Locale,
			oneTimeAccessToken,
			link,
			linkWithCode,
			ttl,
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
	user, err := s.userProvider.GetUser(ctx, userID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return "", apperror.UserNotFound()
	} else if err != nil {
		return "", err
	}

	forbidAdmin, err := adminCodeRestriction(user, origin)
	if err != nil {
		return "", err
	}

	token, err = StoreToken(ctx, s.actorService, userID, ttl, nil, forbidAdmin)
	if err != nil {
		return "", err
	}
	s.auditLog.Record(ctx, auditlogs.EventLoginCodeCreated, origin, userID, auditlogs.Data{"delivery": "link"}, nil)

	return token, nil
}

func (s *Service) ExchangeToken(ctx context.Context, dbConfig *appconfig.AppConfigModel, token, deviceToken, ipAddress, userAgent, browserToken string, remember bool) (model.User, model.LoginTokens, error) {
	actorID := utils.CreateSha256Hash(utils.NormalizeUnambiguousString(token))

	// Consume the token by invoking its actor: this atomically validates it and, if valid, deletes it.
	// It must happen outside of a DB transaction, since invoking an actor while a transaction is open would deadlock on SQLite.
	res, err := s.actorService.Invoke(ctx, TokenActorType, actorID, tokenMethodConsume, tokenConsumeRequest{
		DeviceToken: utils.CreateSha256Hash(deviceToken),
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
		s.restoreToken(ctx, actorID, consumeRes.State)
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

	// The code's creator couldn't have created it for an admin, so it must not sign in as one either
	if state.ForbidAdmin && user.IsAdmin {
		return model.User{}, model.LoginTokens{}, apperror.TokenInvalidOrExpired()
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
func (s *Service) restoreToken(parentCtx context.Context, actorID string, state TokenState) {
	// Use a context that is not canceled when the original request ends
	ctx, cancel := context.WithTimeout(context.WithoutCancel(parentCtx), 10*time.Second)
	defer cancel()

	_, err := s.actorService.Invoke(ctx, TokenActorType, actorID, TokenMethodRestore, state)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to restore one-time access token after a failed exchange", slog.Any("error", err))
	}
}

// StoreToken generates a new one-time access token and persists it in the actor state store, with a TTL matching its lifetime.
// A non-nil device token binds the token to the device that requested it.
// forbidAdmin makes the token stop working if the user becomes an admin before it is used, see TokenState.
func StoreToken(ctx context.Context, store TokenStore, userID string, ttl time.Duration, deviceToken *string, forbidAdmin bool) (token string, err error) {
	token, err = generateToken(ttl)
	if err != nil {
		return "", err
	}

	now := time.Now().Round(time.Second)
	state := TokenState{
		UserID:      userID,
		ExpiresAt:   now.Add(ttl),
		ForbidAdmin: forbidAdmin,
	}
	if deviceToken != nil {
		state.DeviceToken = new(utils.CreateSha256Hash(*deviceToken))
	}

	err = store.SetState(ctx, TokenActorType, utils.CreateSha256Hash(token), state, &actor.SetStateOpts{TTL: ttl})
	if err != nil {
		return "", fmt.Errorf("error saving one-time access token state: %w", err)
	}

	return token, nil
}

// adminCodeRestriction checks that the origin may create a login code for the user, since signing in as an admin hands over their access
// It reports whether the code must stop working should the user become an admin before it is used
func adminCodeRestriction(user model.User, origin auditlogs.Origin) (forbidAdmin bool, err error) {
	if user.IsAdmin {
		return false, origin.Principal.RequireAdminAccess()
	}
	return !origin.Principal.Scopes.Has(authz.AdminAccessWrite), nil
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
