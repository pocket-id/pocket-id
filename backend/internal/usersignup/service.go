package usersignup

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/italypaleale/francis/actor"
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/appconfig"
	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/dto"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
)

// authenticationMethodOneTimePassword identifies one-time-password authentication, used for the initial admin setup token
// It must match the value emitted by the JWT service in the access token's "amr" claim
const authenticationMethodOneTimePassword = "otp"

type Service struct {
	db           *gorm.DB
	actorService *actor.Service
	userCreator  UserCreator
	signer       TokenService
	auditLog     AuditLogger
}

func newService(deps Dependencies, actorService *actor.Service) *Service {
	return &Service{
		db:           deps.DB,
		actorService: actorService,
		userCreator:  deps.UserCreator,
		signer:       deps.Signer,
		auditLog:     deps.AuditLog,
	}
}

func (s *Service) SignUp(ctx context.Context, config *appconfig.AppConfigModel, signupData signUpDto, ipAddress, userAgent string) (model.User, string, error) {
	tokenProvided := signupData.Token != ""

	if config.AllowUserSignups.String() != "open" && !tokenProvided {
		return model.User{}, "", &common.OpenSignupDisabledError{}
	}

	var userGroupIDs []string
	if tokenProvided {
		// Consume the signup token by invoking its actor: this atomically validates it and increments its usage count.
		// It must happen outside of a DB transaction, since invoking an actor while a transaction is open would deadlock on SQLite.
		res, err := s.actorService.Invoke(ctx, SignupTokenActorType, actor.SingletonActorID, signupTokenMethodConsume, signupTokenConsumeRequest{
			Token: signupData.Token,
		})
		if err != nil {
			return model.User{}, "", fmt.Errorf("error invoking signup token actor: %w", err)
		}

		var consumeRes signupTokenConsumeResponse
		err = res.Decode(&consumeRes)
		if err != nil {
			return model.User{}, "", fmt.Errorf("error decoding signup token actor response: %w", err)
		}

		if consumeRes.Status != signupTokenConsumeOK {
			return model.User{}, "", &common.TokenInvalidOrExpiredError{}
		}
		userGroupIDs = consumeRes.UserGroupIDs
	}

	userToCreate := dto.UserCreateDto{
		Username:      signupData.Username,
		Email:         signupData.Email,
		FirstName:     signupData.FirstName,
		LastName:      signupData.LastName,
		DisplayName:   strings.TrimSpace(signupData.FirstName + " " + signupData.LastName),
		UserGroupIds:  userGroupIDs,
		EmailVerified: config.EmailsVerified.IsTrue(),
	}

	// The token has now been consumed. From this point on, if we hit an error we compensate by releasing the token (best-effort).
	user, accessToken, err := s.createSignedUpUser(ctx, config, userToCreate, signupData.Token, tokenProvided, ipAddress, userAgent)
	if err != nil {
		if tokenProvided {
			s.releaseSignupToken(ctx, signupData.Token)
		}
		return model.User{}, "", err
	}

	return user, accessToken, nil
}

// createSignedUpUser creates the user and issues an access token within a single transaction.
// It performs no actor calls, so it's safe to keep the transaction open for its whole duration.
func (s *Service) createSignedUpUser(ctx context.Context, config *appconfig.AppConfigModel, userToCreate dto.UserCreateDto, token string, tokenProvided bool, ipAddress, userAgent string) (model.User, string, error) {
	tx := s.db.Begin()
	defer func() {
		tx.Rollback()
	}()

	user, err := s.userCreator.CreateUserInternal(ctx, config, userToCreate, false, tx)
	if err != nil {
		return model.User{}, "", err
	}

	accessToken, err := s.signer.GenerateAccessToken(user, "", config.SessionDuration.AsDurationMinutes())
	if err != nil {
		return model.User{}, "", err
	}

	if tokenProvided {
		s.auditLog.Create(ctx, model.AuditLogEventAccountCreated, ipAddress, userAgent, user.ID, model.AuditLogData{
			"signupToken": token,
		}, tx)
	} else {
		s.auditLog.Create(ctx, model.AuditLogEventAccountCreated, ipAddress, userAgent, user.ID, model.AuditLogData{
			"method": "open_signup",
		}, tx)
	}

	err = tx.Commit().Error
	if err != nil {
		return model.User{}, "", err
	}

	return user, accessToken, nil
}

// releaseSignupToken reverts the usage count increment performed while consuming a token, used to compensate when the signup could not be completed.
// It's a best-effort compensation: it uses a context that is not canceled when the original request ends, but if it fails (or the process crashes before it runs) we accept that a token use was consumed unnecessarily.
func (s *Service) releaseSignupToken(ctx context.Context, token string) {
	innerCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
	defer cancel()

	_, err := s.actorService.Invoke(innerCtx, SignupTokenActorType, actor.SingletonActorID, signupTokenMethodRelease, signupTokenReleaseRequest{
		Token: token,
	})
	if err != nil {
		slog.ErrorContext(innerCtx, "Failed to release signup token after a failed signup", slog.Any("error", err))
	}
}

func (s *Service) SignUpInitialAdmin(ctx context.Context, config *appconfig.AppConfigModel, signUpData signUpDto) (model.User, string, error) {
	tx := s.db.Begin()
	defer func() {
		tx.Rollback()
	}()

	setupCompleted, err := s.isInitialAdminSetupCompleted(ctx, tx)
	if err != nil {
		return model.User{}, "", err
	}
	if setupCompleted {
		return model.User{}, "", &common.SetupNotAvailableError{}
	}

	userToCreate := dto.UserCreateDto{
		FirstName:   signUpData.FirstName,
		LastName:    signUpData.LastName,
		DisplayName: strings.TrimSpace(signUpData.FirstName + " " + signUpData.LastName),
		Username:    signUpData.Username,
		Email:       signUpData.Email,
		IsAdmin:     true,
	}

	user, err := s.userCreator.CreateUserInternal(ctx, config, userToCreate, false, tx)
	if err != nil {
		return model.User{}, "", err
	}

	token, err := s.signer.GenerateAccessToken(user, authenticationMethodOneTimePassword, config.SessionDuration.AsDurationMinutes())
	if err != nil {
		return model.User{}, "", err
	}

	err = tx.Commit().Error
	if err != nil {
		return model.User{}, "", err
	}

	return user, token, nil
}

func (s *Service) IsInitialAdminSetupCompleted(ctx context.Context) (bool, error) {
	return s.isInitialAdminSetupCompleted(ctx, s.db)
}

func (s *Service) isInitialAdminSetupCompleted(ctx context.Context, db *gorm.DB) (bool, error) {
	var userCount int64
	if err := db.WithContext(ctx).Model(&model.User{}).
		Where("id != ?", common.StaticApiKeyUserID).
		Count(&userCount).Error; err != nil {
		return false, err
	}

	return userCount != 0, nil
}

func (s *Service) ListSignupTokens(ctx context.Context, listRequestOptions utils.ListRequestOptions) ([]SignupToken, utils.PaginationResponse, error) {
	// Signup tokens are held in the singleton actor's state, so we retrieve them all via a read-only Peek and then sort and paginate in memory.
	res, err := s.actorService.Peek(ctx, SignupTokenActorType, actor.SingletonActorID, signupTokenMethodList, nil)
	if err != nil {
		return nil, utils.PaginationResponse{}, fmt.Errorf("error listing signup tokens from actor: %w", err)
	}

	var listRes signupTokenListResponse
	err = res.Decode(&listRes)
	if err != nil {
		return nil, utils.PaginationResponse{}, fmt.Errorf("error decoding signup token actor response: %w", err)
	}

	// Resolve the referenced user groups so they can be included in the response
	groupsByID, err := s.loadUserGroupsByID(ctx, listRes.Tokens)
	if err != nil {
		return nil, utils.PaginationResponse{}, err
	}

	tokens := make([]SignupToken, len(listRes.Tokens))
	for i, t := range listRes.Tokens {
		tokens[i] = signupTokenModelFromStored(t, resolveUserGroups(t.UserGroupIDs, groupsByID))
	}

	return paginateSignupTokens(tokens, listRequestOptions)
}

func (s *Service) DeleteSignupToken(ctx context.Context, tokenID string) error {
	_, err := s.actorService.Invoke(ctx, SignupTokenActorType, actor.SingletonActorID, signupTokenMethodDelete, signupTokenDeleteRequest{
		ID: tokenID,
	})
	if err != nil {
		return fmt.Errorf("error deleting signup token via actor: %w", err)
	}
	return nil
}

func (s *Service) CreateSignupToken(ctx context.Context, ttl time.Duration, usageLimit int, userGroupIDs []string) (SignupToken, error) {
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
	randomString, err := utils.GenerateRandomAlphanumericString(16)
	if err != nil {
		return SignupToken{}, err
	}

	now := time.Now().Round(time.Second)
	stored := storedSignupToken{
		ID:           uuid.NewString(),
		Token:        randomString,
		ExpiresAt:    now.Add(ttl),
		UsageLimit:   usageLimit,
		UsageCount:   0,
		UserGroupIDs: validGroupIDs,
		CreatedAt:    now,
	}

	_, err = s.actorService.Invoke(ctx, SignupTokenActorType, actor.SingletonActorID, signupTokenMethodCreate, stored)
	if err != nil {
		return SignupToken{}, fmt.Errorf("error creating signup token via actor: %w", err)
	}

	return signupTokenModelFromStored(stored, userGroups), nil
}

// loadUserGroupsByID loads every user group referenced by the given tokens, keyed by ID.
func (s *Service) loadUserGroupsByID(ctx context.Context, tokens []storedSignupToken) (map[string]model.UserGroup, error) {
	idSet := make(map[string]struct{})
	for _, t := range tokens {
		for _, id := range t.UserGroupIDs {
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
		if g, ok := byID[id]; ok {
			groups = append(groups, g)
		}
	}
	return groups
}

// signupTokenModelFromStored builds the API/model representation of a signup token from its stored form.
func signupTokenModelFromStored(t storedSignupToken, groups []model.UserGroup) SignupToken {
	return SignupToken{
		Base: model.Base{
			ID:        t.ID,
			CreatedAt: datatype.DateTime(t.CreatedAt),
		},
		Token:      t.Token,
		ExpiresAt:  datatype.DateTime(t.ExpiresAt),
		UsageLimit: t.UsageLimit,
		UsageCount: t.UsageCount,
		UserGroups: groups,
	}
}

// paginateSignupTokens sorts and paginates the in-memory list of signup tokens, mirroring the behavior of the DB-backed pagination utility.
func paginateSignupTokens(tokens []SignupToken, params utils.ListRequestOptions) ([]SignupToken, utils.PaginationResponse, error) {
	sortSignupTokens(tokens, params.Sort.Column, params.Sort.Direction)

	page := params.Pagination.Page
	if page < 1 {
		page = 1
	}
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

	start := (page - 1) * pageSize
	if start > len(tokens) {
		start = len(tokens)
	}
	end := start + pageSize
	if end > len(tokens) {
		end = len(tokens)
	}

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

	less := func(i, j int) bool { return time.Time(tokens[i].CreatedAt).Before(time.Time(tokens[j].CreatedAt)) }
	switch column {
	case "expiresAt":
		less = func(i, j int) bool { return time.Time(tokens[i].ExpiresAt).Before(time.Time(tokens[j].ExpiresAt)) }
	case "usageLimit":
		less = func(i, j int) bool { return tokens[i].UsageLimit < tokens[j].UsageLimit }
	case "usageCount":
		less = func(i, j int) bool { return tokens[i].UsageCount < tokens[j].UsageCount }
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

// loadExistingSignupTokens loads all signup tokens currently stored in the database, so they can seed the actor's state on first startup.
func loadExistingSignupTokens(ctx context.Context, db *gorm.DB) ([]storedSignupToken, error) {
	var tokens []SignupToken
	err := db.WithContext(ctx).
		Preload("UserGroups").
		Find(&tokens).
		Error
	if err != nil {
		return nil, fmt.Errorf("failed to load existing signup tokens: %w", err)
	}

	result := make([]storedSignupToken, len(tokens))
	for i, t := range tokens {
		groupIDs := make([]string, len(t.UserGroups))
		for j, g := range t.UserGroups {
			groupIDs[j] = g.ID
		}
		result[i] = storedSignupToken{
			ID:           t.ID,
			Token:        t.Token,
			ExpiresAt:    time.Time(t.ExpiresAt),
			UsageLimit:   t.UsageLimit,
			UsageCount:   t.UsageCount,
			UserGroupIDs: groupIDs,
			CreatedAt:    time.Time(t.CreatedAt),
		}
	}

	return result, nil
}
