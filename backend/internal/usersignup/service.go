package usersignup

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"
	"uuid"

	"github.com/italypaleale/francis/actor"
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/appconfig"
	"github.com/pocket-id/pocket-id/backend/internal/apperror"
	"github.com/pocket-id/pocket-id/backend/internal/auditlogs"
	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/dto"
	"github.com/pocket-id/pocket-id/backend/internal/emailverification"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
)

// authenticationMethodOneTimePassword identifies one-time-password authentication, used for the initial admin setup token
// It must match the value emitted by the JWT service in the access token's "amr" claim
const authenticationMethodOneTimePassword = "otp"

// signupTokenPrefixLength is how many leading characters of a signup token are kept in clear text so admins can tell tokens apart
const signupTokenPrefixLength = 4

// signupVerificationPeriod is how long a self-signed-up account has to verify its email before it is deleted
const signupVerificationPeriod = emailverification.TokenLifetime

type Service struct {
	db                *gorm.DB
	actorService      *actor.Service
	userCreator       UserCreator
	sessions          SessionCreator
	auditLog          AuditLogger
	scimSync          ScimSyncScheduler
	emailVerification EmailVerificationSender
	emailSender       EmailSender
}

func newService(deps Dependencies, actorService *actor.Service) *Service {
	return &Service{
		db:                deps.DB,
		actorService:      actorService,
		userCreator:       deps.UserCreator,
		sessions:          deps.Sessions,
		auditLog:          deps.AuditLog,
		scimSync:          deps.ScimSync,
		emailVerification: deps.EmailVerification,
		emailSender:       deps.EmailSender,
	}
}

func (s *Service) SignUp(ctx context.Context, config *appconfig.AppConfigModel, signupData signUpDto, ipAddress, userAgent string) (model.User, string, error) {
	tokenProvided := signupData.Token != ""

	if config.AllowUserSignups.String() != "open" && !tokenProvided {
		return model.User{}, "", apperror.OpenSignupDisabled()
	}

	var signupTokenID string
	var userGroupIDs []string
	if tokenProvided {
		// Consume the signup token by invoking its actor: this atomically validates it and increments its usage count
		// Note: must invoke outside of a DB transaction, since invoking an actor while a transaction is open would deadlock on SQLite
		res, err := s.actorService.Invoke(ctx, SignupTokenActorType, utils.CreateSha256Hash(signupData.Token), signupTokenMethodConsume, nil)
		if err != nil {
			return model.User{}, "", fmt.Errorf("error invoking signup token actor: %w", err)
		}

		var consumeRes signupTokenConsumeResponse
		err = res.Decode(&consumeRes)
		if err != nil {
			return model.User{}, "", fmt.Errorf("error decoding signup token actor response: %w", err)
		}

		if consumeRes.Status != signupTokenConsumeOK {
			return model.User{}, "", apperror.TokenInvalidOrExpired()
		}
		signupTokenID = consumeRes.ID
		userGroupIDs = consumeRes.UserGroupIDs
	}

	userToCreate := dto.UserCreateDto{
		Username:      signupData.Username,
		Email:         signupData.Email,
		FirstName:     signupData.FirstName,
		LastName:      signupData.LastName,
		DisplayName:   strings.TrimSpace(signupData.FirstName + " " + signupData.LastName),
		Locale:        signupData.Locale,
		UserGroupIds:  userGroupIDs,
		EmailVerified: config.EmailsVerified.IsTrue(),
	}

	// The token has now been consumed
	// From this point on, if we hit an error we compensate by releasing the token (best-effort)
	user, err := s.createSignedUpUser(ctx, config, userToCreate, signupTokenID, ipAddress, userAgent)
	if err != nil {
		if tokenProvided {
			s.releaseSignupToken(ctx, signupData.Token)
		}
		return model.User{}, "", err
	}

	sessionToken, err := s.sessions.Create(ctx, user.ID, "", ipAddress, userAgent, config.SessionDuration.AsDurationMinutes(), true)
	if err != nil {
		return model.User{}, "", err
	}

	if emailverification.IsPending(config, &user) {
		err = s.emailVerification.Send(ctx, config, user.ID)
		if err != nil {
			slog.WarnContext(ctx, "Failed to send the email verification message after signup", slog.String("userID", user.ID), slog.Any("error", err))
		}
	}

	return user, sessionToken, nil
}

// createSignedUpUser creates the user within a single transaction.
// It performs no actor calls, so it's safe to keep the transaction open for its whole duration.
func (s *Service) createSignedUpUser(ctx context.Context, config *appconfig.AppConfigModel, userToCreate dto.UserCreateDto, signupTokenID string, ipAddress, userAgent string) (model.User, error) {
	tx := s.db.Begin()
	defer func() {
		tx.Rollback()
	}()

	user, err := s.userCreator.CreateUserInternal(ctx, config, userToCreate, false, tx)
	if err != nil {
		return model.User{}, err
	}

	// Give the account a deadline to verify its email, after which it is deleted so abandoned signups cannot squat an address
	if emailverification.IsPending(config, &user) {
		deadline := datatype.DateTime(time.Now().Add(signupVerificationPeriod))
		err = tx.WithContext(ctx).Model(&user).Update("email_verification_deadline", deadline).Error
		if err != nil {
			return model.User{}, err
		}
		user.EmailVerificationDeadline = &deadline
	}

	// The token's value is a reusable credential, so audit log readers only get its ID
	if signupTokenID != "" {
		s.auditLog.Create(ctx, auditlogs.EventAccountCreated, ipAddress, userAgent, user.ID, auditlogs.Data{
			"signupTokenID": signupTokenID,
		}, tx)
	} else {
		s.auditLog.Create(ctx, auditlogs.EventAccountCreated, ipAddress, userAgent, user.ID, auditlogs.Data{
			"method": "open_signup",
		}, tx)
	}

	err = tx.Commit().Error
	if err != nil {
		return model.User{}, err
	}
	if s.scimSync != nil {
		s.scimSync.ScheduleSync(ctx)
	}

	return user, nil
}

// releaseSignupToken reverts the usage count increment performed while consuming a token, used to compensate when the signup could not be completed.
// It's a best-effort compensation: if it fails (or the process crashes before it runs) we accept that a token use was consumed unnecessarily
func (s *Service) releaseSignupToken(parentCtx context.Context, token string) {
	// Use a context that is not canceled when the original request ends
	ctx, cancel := context.WithTimeout(context.WithoutCancel(parentCtx), 10*time.Second)
	defer cancel()

	_, err := s.actorService.Invoke(ctx, SignupTokenActorType, utils.CreateSha256Hash(token), signupTokenMethodRelease, nil)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to release signup token after a failed signup", slog.Any("error", err))
	}
}

func (s *Service) SignUpInitialAdmin(ctx context.Context, config *appconfig.AppConfigModel, signUpData signUpDto, ipAddress, userAgent string) (model.User, string, error) {
	tx := s.db.Begin()
	defer func() {
		tx.Rollback()
	}()

	// We lock the users table to prevent concurrent initial admin setups from racing to create the first user
	// This is only necessary for Postgres, since SQLite serializes all writes anyway
	if err := lockInitialAdminSetup(ctx, tx); err != nil {
		return model.User{}, "", err
	}

	// Reject setup when a committed user already exists or the first admin comes from LDAP
	setupCompleted, err := s.isInitialAdminSetupCompleted(ctx, tx, config)
	if err != nil {
		return model.User{}, "", err
	}
	if setupCompleted {
		return model.User{}, "", apperror.SetupAlreadyCompleted()
	}

	// Build the first user with administrator privileges
	userToCreate := dto.UserCreateDto{
		FirstName:   signUpData.FirstName,
		LastName:    signUpData.LastName,
		DisplayName: strings.TrimSpace(signUpData.FirstName + " " + signUpData.LastName),
		Username:    signUpData.Username,
		Email:       signUpData.Email,
		Locale:      signUpData.Locale,
		IsAdmin:     true,
	}

	user, err := s.userCreator.CreateUserInternal(ctx, config, userToCreate, false, tx)
	if err != nil {
		return model.User{}, "", err
	}
	s.auditLog.Create(ctx, auditlogs.EventAccountCreated, ipAddress, userAgent, user.ID, auditlogs.Data{"method": "initial_setup"}, tx)

	err = tx.Commit().Error
	if err != nil {
		return model.User{}, "", err
	}
	if s.scimSync != nil {
		s.scimSync.ScheduleSync(ctx)
	}

	// Start the session only after the transaction has committed, since it invokes an actor
	token, err := s.sessions.Create(ctx, user.ID, authenticationMethodOneTimePassword, ipAddress, userAgent, config.SessionDuration.AsDurationMinutes(), true)
	if err != nil {
		return model.User{}, "", err
	}

	return user, token, nil
}

func lockInitialAdminSetup(ctx context.Context, tx *gorm.DB) error {
	if tx.Name() != "postgres" {
		return nil
	}

	if err := tx.WithContext(ctx).Exec("LOCK TABLE users IN SHARE ROW EXCLUSIVE MODE").Error; err != nil {
		return fmt.Errorf("failed to lock users table for initial admin setup: %w", err)
	}

	return nil
}

func (s *Service) isInitialAdminSetupCompleted(ctx context.Context, db *gorm.DB, config *appconfig.AppConfigModel) (bool, error) {
	// An LDAP sync that deletes the users missing from the directory can empty the users table, which must not reopen the setup to whoever visits it first
	// An instance with LDAP gets its first admin from the directory instead
	if config.LdapEnabled.IsTrue() {
		return true, nil
	}

	var exists bool
	err := db.WithContext(ctx).Raw("SELECT EXISTS (SELECT 1 FROM users)").Scan(&exists).Error
	return exists, err
}

func (s *Service) ListSignupTokens(ctx context.Context, listRequestOptions utils.ListRequestOptions) ([]SignupToken, utils.PaginationResponse, error) {
	// Each signup token is its own actor, so we enumerate the stored states (expired ones are filtered out by the state store), then sort and paginate in memory
	entries, err := s.listSignupTokenStates(ctx)
	if err != nil {
		return nil, utils.PaginationResponse{}, err
	}

	// Resolve the referenced user groups so they can be included in the response
	groupsByID, err := s.loadUserGroupsByID(ctx, entries)
	if err != nil {
		return nil, utils.PaginationResponse{}, err
	}

	tokens := make([]SignupToken, len(entries))
	for i, e := range entries {
		tokens[i] = signupTokenModelFromState(e.State, resolveUserGroups(e.State.UserGroupIDs, groupsByID))
	}

	return paginateSignupTokens(tokens, listRequestOptions)
}

func (s *Service) DeleteSignupToken(ctx context.Context, tokenID string) error {
	// Tokens are addressed by their hash (the actor ID), while the API deletes them by ID, so we look up the matching token first
	entries, err := s.listSignupTokenStates(ctx)
	if err != nil {
		return err
	}

	for _, e := range entries {
		if e.State.ID != tokenID {
			continue
		}

		_, err = s.actorService.Invoke(ctx, SignupTokenActorType, e.ActorID, SignupTokenMethodDelete, nil)
		if err != nil {
			return fmt.Errorf("error deleting signup token via actor: %w", err)
		}
		return nil
	}

	// The token doesn't exist (or has expired): deleting it already reaches the desired end state
	return nil
}

// signupTokenEntry pairs a signup token's actor ID with its stored state
type signupTokenEntry struct {
	ActorID string
	State   SignupTokenState
}

// listSignupTokenStates returns every signup token currently stored in the actor state store.
// Expired tokens are not returned, since the state store filters out states whose TTL has passed.
func (s *Service) listSignupTokenStates(ctx context.Context) ([]signupTokenEntry, error) {
	var (
		entries []signupTokenEntry
		after   string
	)
	for {
		res, err := s.actorService.ListStates(ctx, SignupTokenActorType, &actor.ListStatesOpts{
			IncludeData: true,
			After:       after,
		})
		if err != nil {
			return nil, fmt.Errorf("error listing signup token states: %w", err)
		}

		for _, st := range res.States {
			if st.Data == nil {
				continue
			}

			var state SignupTokenState
			err = st.Data.Decode(&state)
			if err != nil {
				return nil, fmt.Errorf("error decoding state of signup token actor '%s': %w", st.ActorID, err)
			}

			entries = append(entries, signupTokenEntry{
				ActorID: st.ActorID,
				State:   state,
			})
		}

		// An empty cursor means we've just read the last page
		after = res.AfterID()
		if after == "" {
			break
		}
	}

	return entries, nil
}

func (s *Service) CreateSignupToken(ctx context.Context, ttl time.Duration, usageLimit int, userGroupIDs []string, sentTo string) (SignupToken, error) {
	// Load the referenced user groups to validate them and to include them in the response
	var userGroups []model.UserGroup
	if len(userGroupIDs) > 0 {
		err := s.db.WithContext(ctx).
			Where("id IN ?", userGroupIDs).
			Find(&userGroups).
			Error
		if err != nil {
			return SignupToken{}, err
		}
	}

	validGroupIDs := make([]string, len(userGroups))
	for i, g := range userGroups {
		validGroupIDs[i] = g.ID
	}

	// Generate a random token
	token, err := utils.GenerateRandomAlphanumericString(16)
	if err != nil {
		return SignupToken{}, err
	}

	now := time.Now().Round(time.Second)
	state := SignupTokenState{
		ID:           uuid.NewV4().String(),
		Prefix:       token[:signupTokenPrefixLength],
		ExpiresAt:    now.Add(ttl),
		UsageLimit:   usageLimit,
		UsageCount:   0,
		UserGroupIDs: validGroupIDs,
		SentTo:       sentTo,
		CreatedAt:    now,
	}

	_, err = s.actorService.Invoke(ctx, SignupTokenActorType, utils.CreateSha256Hash(token), SignupTokenMethodCreate, state)
	if err != nil {
		return SignupToken{}, fmt.Errorf("error creating signup token via actor: %w", err)
	}

	// The response is the only place the token appears, since only its hash is stored
	signupToken := signupTokenModelFromState(state, userGroups)
	signupToken.Token = token
	return signupToken, nil
}

// SendSignupInvitation emails the signup link of a token to the given address
// It sends in the background like an admin-sent login code, so a failure is only logged
func (s *Service) SendSignupInvitation(ctx context.Context, dbConfig *appconfig.AppConfigModel, signupToken SignupToken, recipientEmail string, ttl time.Duration) {
	go func() {
		// Keep the request's tracing info but not its cancellation, which would stop the delivery when the request ends
		innerCtx := context.WithoutCancel(ctx)

		signupLink := common.EnvConfig.AppURL + "/st/" + signupToken.Token
		err := s.emailSender.SendSignupInvitation(innerCtx, dbConfig, recipientEmail, signupLink, ttl)
		if err != nil {
			slog.ErrorContext(innerCtx, "Failed to send signup invitation email", slog.Any("error", err), slog.String("address", recipientEmail))
		}
	}()
}

// loadUserGroupsByID loads every user group referenced by the given tokens, keyed by ID.
func (s *Service) loadUserGroupsByID(ctx context.Context, entries []signupTokenEntry) (map[string]model.UserGroup, error) {
	idSet := make(map[string]struct{})
	for _, e := range entries {
		for _, id := range e.State.UserGroupIDs {
			idSet[id] = struct{}{}
		}
	}
	if len(idSet) == 0 {
		return map[string]model.UserGroup{}, nil
	}

	ids := make([]string, 0, len(idSet))
	for id := range idSet {
		ids = append(ids, id)
	}

	var groups []model.UserGroup
	err := s.db.WithContext(ctx).
		Where("id IN ?", ids).
		Find(&groups).
		Error
	if err != nil {
		return nil, err
	}

	byID := make(map[string]model.UserGroup, len(groups))
	for _, g := range groups {
		byID[g.ID] = g
	}

	return byID, nil
}

// resolveUserGroups maps the given group IDs to the corresponding UserGroup objects, preserving order and skipping any that no longer exist.
func resolveUserGroups(ids []string, byID map[string]model.UserGroup) []model.UserGroup {
	if len(ids) == 0 {
		return nil
	}

	groups := make([]model.UserGroup, 0, len(ids))
	for _, id := range ids {
		g, ok := byID[id]
		if ok {
			groups = append(groups, g)
		}
	}

	return groups
}

// signupTokenModelFromState builds the API/model representation of a signup token from its stored state.
func signupTokenModelFromState(state SignupTokenState, groups []model.UserGroup) SignupToken {
	return SignupToken{
		Base: model.Base{
			ID:        state.ID,
			CreatedAt: datatype.DateTime(state.CreatedAt),
		},
		Prefix:     state.Prefix,
		ExpiresAt:  datatype.DateTime(state.ExpiresAt),
		UsageLimit: state.UsageLimit,
		UsageCount: state.UsageCount,
		UserGroups: groups,
		SentTo:     state.SentTo,
	}
}

// paginateSignupTokens sorts and paginates the in-memory list of signup tokens, mirroring the behavior of the DB-backed pagination utility.
func paginateSignupTokens(tokens []SignupToken, params utils.ListRequestOptions) ([]SignupToken, utils.PaginationResponse, error) {
	sortSignupTokens(tokens, params.Sort.Column, params.Sort.Direction)

	page := max(params.Pagination.Page, 1)
	pageSize := params.Pagination.Limit
	switch {
	case pageSize < 1:
		pageSize = 20
	case pageSize > 100:
		pageSize = 100
	}

	totalItems := int64(len(tokens))
	totalPages := (totalItems + int64(pageSize) - 1) / int64(pageSize)
	if totalItems == 0 {
		totalPages = 1
	}
	if int64(page) > totalPages {
		page = int(totalPages)
	}

	start := min((page-1)*pageSize, len(tokens))
	end := min(start+pageSize, len(tokens))

	return tokens[start:end], utils.PaginationResponse{
		TotalPages:   totalPages,
		TotalItems:   totalItems,
		CurrentPage:  page,
		ItemsPerPage: pageSize,
	}, nil
}

// sortSignupTokens sorts the tokens by the given column and direction.
// It defaults to sorting by creation date ascending, matching the DB-backed listing.
func sortSignupTokens(tokens []SignupToken, column, direction string) {
	desc := utils.NormalizeSortDirection(direction) == "desc"

	less := func(i, j int) bool {
		caI := time.Time(tokens[i].CreatedAt)
		caJ := time.Time(tokens[j].CreatedAt)
		return caI.Before(caJ)
	}
	switch column {
	case "expiresAt":
		less = func(i, j int) bool {
			eaI := time.Time(tokens[i].ExpiresAt)
			eaJ := time.Time(tokens[j].ExpiresAt)
			return eaI.Before(eaJ)
		}
	case "usageLimit":
		less = func(i, j int) bool { return tokens[i].UsageLimit < tokens[j].UsageLimit }
	case "usageCount":
		less = func(i, j int) bool {
			return tokens[i].UsageCount < tokens[j].UsageCount
		}
	case "createdAt", "":
		// Use the default comparator (creation date)
	default:
		// Unknown or non-sortable column: keep the default (creation date) ordering
	}

	sort.SliceStable(tokens, func(i, j int) bool {
		if desc {
			return less(j, i)
		}
		return less(i, j)
	})
}
