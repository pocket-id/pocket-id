package oidc

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"slices"
	"strings"

	"github.com/pocket-id/fosite"
	fositejwt "github.com/pocket-id/fosite/token/jwt"
	"github.com/pocket-id/pocket-id/backend/internal/appconfig"
	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/emailverification"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	"gorm.io/gorm"
)

type TokenType string

const (
	IDTokenType     TokenType = "id-token"
	AccessTokenType TokenType = "access-token"
	UserInfoType    TokenType = "user-info"
)

const emailVerificationRequiredHint = "You must verify your email address before you can access this service."

type ClaimsService struct {
	db           *gorm.DB
	customClaims CustomClaimSource
	baseURL      string
	signer       TokenSigner
	appConfig    appconfig.AppConfigResolver
}

func newClaimsService(db *gorm.DB, customClaims CustomClaimSource, baseURL string, signer TokenSigner, appConfig appconfig.AppConfigResolver) *ClaimsService {
	return &ClaimsService{
		db:           db,
		customClaims: customClaims,
		baseURL:      baseURL,
		signer:       signer,
		appConfig:    appConfig,
	}
}

// isEmailVerificationPending reports whether the user still has to verify their email before tokens can be issued for them
func (s *ClaimsService) isEmailVerificationPending(ctx context.Context, user *model.User) (bool, error) {
	if !emailverification.CanBePending(user) {
		return false, nil
	}

	cfg, err := s.appConfig.GetConfig(ctx)
	if err != nil {
		return false, err
	}

	return emailverification.IsPending(cfg, user), nil
}

// ValidateUserAccess re-checks, at token-issuance time, that the user behind a grant is
// still allowed to obtain tokens for the client.
func (s *ClaimsService) ValidateUserAccess(ctx context.Context, userID string, client Client) error {
	// Grants without a resource owner (e.g. client_credentials) carry an empty subject
	// and have no user to validate.
	if userID == "" {
		return nil
	}

	var user model.User
	err := dbFromContext(ctx, s.db).
		Preload("UserGroups").
		First(&user, "id = ?", userID).
		Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return fosite.ErrInvalidGrant.WithHint("The user account no longer exists.")
	}
	if err != nil {
		return err
	}

	if user.Disabled {
		return fosite.ErrInvalidGrant.WithHint("The user account is disabled.")
	}

	// The authorization middleware already stops these users at the authorize step, but grants issued before the email became unverified, such as refresh tokens, are only checked here
	pending, err := s.isEmailVerificationPending(ctx, &user)
	if err != nil {
		return err
	}
	if pending {
		return fosite.ErrInvalidGrant.WithHint(emailVerificationRequiredHint)
	}

	if !IsUserGroupAllowedToAuthorize(user, client.OidcClient) {
		return fosite.ErrAccessDenied.WithHint("You are not allowed to access this service.")
	}

	return nil
}

// GetClaimMappingPolicyByClientID returns the claim mapping policy assigned to the client, or the
// default policy when the client doesn't have one assigned (claim_mapping_policy_id IS NULL).
// Both candidates are fetched in a single query: the assigned policy is ordered first so it wins
// over the default whenever it exists.
func (s *ClaimsService) GetClaimMappingPolicyByClientID(ctx context.Context, clientID string) (*model.OidcClaimMappingPolicy, error) {
	var claimMappingPolicy model.OidcClaimMappingPolicy
	err := dbFromContext(ctx, s.db).
		Model(&model.OidcClaimMappingPolicy{}).
		Where("oidc_claim_mapping_policies.id = (SELECT claim_mapping_policy_id FROM oidc_clients WHERE oidc_clients.id = ?)", clientID).
		Or("oidc_claim_mapping_policies.is_default = ?", true).
		Order("oidc_claim_mapping_policies.is_default ASC").
		First(&claimMappingPolicy).Error
	if err != nil {
		return nil, err
	}
	return &claimMappingPolicy, nil
}

// applyIDTokenClaims applies the claims of a user to the ID token claims in the session based on the requested scopes.
func (s *ClaimsService) applyIDTokenClaims(ctx context.Context, session *Session, scopes fosite.Arguments, claimMappingPolicy model.OidcClaimMappingPolicy) error {
	userID := session.Subject
	if userID == "" {
		return nil
	}

	claims, err := s.GetUserClaims(ctx, userID, scopes, claimMappingPolicy, IDTokenType)
	if err != nil {
		return err
	}

	// Record the signing algorithm on the ID token header so fosite derives the at_hash/
	// c_hash digest from it (e.g. RS384 -> SHA-384, ES512 -> SHA-512). Without this the
	// header is empty and fosite defaults to SHA-256, producing wrong hashes whenever the
	// signing key is not a 256-bit algorithm. ToMap() strips "alg" before signing, so this
	// never overrides the real JWS header. The signer is always wired in production; it is
	// only nil in unit tests that do not assert hash correctness.
	if s.signer != nil {
		alg, err := s.signer.GetKeyAlg()
		if err != nil {
			return err
		}
		session.IDTokenHeaders().Add("alg", alg.String())
	}

	applyUserClaimsToIDToken(session, userID, claims)
	return nil
}

func applyUserClaimsToIDToken(session *Session, userID string, claims map[string]any) {
	idTokenClaims := session.IDTokenClaims()
	idTokenClaims.Subject = userID
	idTokenClaims.Extra = claims
	idTokenClaims.Extra[common.TokenTypeClaim] = string(IDTokenType)
	if session.AuthenticationMethod != "" {
		idTokenClaims.AuthenticationMethodsReferences = []string{session.AuthenticationMethod}
	}
}

// applyIDTokenClaims applies the claims of a user to the ID token claims in the session based on the requested scopes.
func (s *ClaimsService) applyAccessTokenClaims(ctx context.Context, session *Session, scopes fosite.Arguments, claimMappingPolicy model.OidcClaimMappingPolicy) error {
	userID := session.Subject
	if userID == "" {
		return nil
	}

	claims, err := s.GetUserClaims(ctx, userID, scopes, claimMappingPolicy, AccessTokenType)
	if err != nil {
		return err
	}

	applyUserClaimsToAccessToken(session, userID, claims)
	return nil
}

func applyUserClaimsToAccessToken(session *Session, userID string, claims map[string]any) {
	jwtClaims := session.GetJWTClaims().(*fositejwt.JWTClaims)
	jwtClaims.Extra = claims
}

// GetUserClaims retrieves the claims for a user based on the requested scopes. It includes standard claims
// like "sub" and "email" as well as any custom claims defined for the user or their groups.
func (s *ClaimsService) GetUserClaims(ctx context.Context, userID string, scopes []string, claimMappingPolicy model.OidcClaimMappingPolicy, tokenType TokenType) (map[string]any, error) {
	db := dbFromContext(ctx, s.db)

	var user model.User
	err := db.
		Preload("UserGroups").
		First(&user, "id = ?", userID).
		Error
	if err != nil {
		return nil, err
	}

	var customClaims []model.CustomClaim

	claims := make(map[string]any, 10)

	// filter mappings and apply them
	for _, mapping := range claimMappingPolicy.ClaimMappings {
		if ((mapping.IDToken && tokenType == IDTokenType) ||
			(mapping.AccessToken && tokenType == AccessTokenType) ||
			(mapping.UserInfo && tokenType == UserInfoType)) &&
			isScopeMatching(scopes, mapping.Scope) {
			// lazy custom claims
			if mapping.SourceType == model.MappingSourceCustomClaim && customClaims == nil {
				customClaims, err = s.customClaims.GetCustomClaimsForUserWithUserGroups(ctx, user.ID, db)
				if err != nil {
					return nil, err
				}
			}
			s.applyUserClaims(claims, user, customClaims, mapping)
		}
	}

	return claims, nil
}

// parseCustomClaimValue decodes a custom claim value that is a JSON document and returns any other value as a plain string
// Integers are kept as int64 because float64 drops digits past 2^53 and is written in exponent form in signed tokens
func parseCustomClaimValue(value string) any {
	decoder := json.NewDecoder(strings.NewReader(value))
	decoder.UseNumber()

	var parsed any
	if err := decoder.Decode(&parsed); err != nil || decoder.Decode(new(any)) != io.EOF {
		return value
	}

	normalized, err := normalizeJSONNumbers(parsed)
	if err != nil {
		return value
	}
	return normalized
}

// normalizeJSONNumbers replaces every json.Number with an int64 when it is an integer and with a float64 otherwise
func normalizeJSONNumbers(value any) (any, error) {
	switch v := value.(type) {
	case json.Number:
		if i, err := v.Int64(); err == nil {
			return i, nil
		}
		return v.Float64()
	case map[string]any:
		for key, item := range v {
			normalized, err := normalizeJSONNumbers(item)
			if err != nil {
				return nil, err
			}
			v[key] = normalized
		}
	case []any:
		for i, item := range v {
			normalized, err := normalizeJSONNumbers(item)
			if err != nil {
				return nil, err
			}
			v[i] = normalized
		}
	}
	return value, nil
}

func isScopeMatching(requestedScopes []string, claimScopes []string) bool {
	for _, claimScope := range claimScopes {
		if slices.Contains(requestedScopes, claimScope) {
			return true
		}
	}
	return false
}

func (s *ClaimsService) applyUserClaims(claims map[string]any, user model.User, customClaims []model.CustomClaim, mapping model.OidcClaimMapping) {
	switch mapping.SourceType {
	case model.MappingSourceUserField:
		// Map from user field
		switch model.OidcUserField(mapping.SourceValue) {
		case model.UserFieldID:
			claims[mapping.ClaimName] = user.ID
		case model.UserFieldEmail:
			if user.Email != nil {
				claims[mapping.ClaimName] = *user.Email
			}
		case model.UserFieldEmailVerified:
			claims[mapping.ClaimName] = user.EmailVerified
		case model.UserFieldFirstName:
			claims[mapping.ClaimName] = user.FirstName
		case model.UserFieldLastName:
			claims[mapping.ClaimName] = user.LastName
		case model.UserFieldFullName:
			claims[mapping.ClaimName] = user.FullName()
		case model.UserFieldDisplayName:
			claims[mapping.ClaimName] = user.DisplayName
		case model.UserFieldUsername:
			claims[mapping.ClaimName] = user.Username
		case model.UserFieldLocale:
			if user.Locale != nil {
				claims[mapping.ClaimName] = user.Locale
			}
		case model.UserFieldPicture:
			claims[mapping.ClaimName] = s.baseURL + "/api/users/" + user.ID + "/profile-picture.png"
		case model.UserFieldGroups:
			userGroups := make([]string, len(user.UserGroups))
			for i, group := range user.UserGroups {
				userGroups[i] = group.Name
			}
			claims[mapping.ClaimName] = userGroups
		}

	case model.MappingSourceCustomClaim:
		// Map from custom claim
		for _, customClaim := range customClaims {
			if mapping.SourceValue == "*" || customClaim.Key == mapping.SourceValue {
				claimName := mapping.ClaimName
				if mapping.ClaimName == "*" {
					claimName = customClaim.Key
				}
				claims[claimName] = parseCustomClaimValue(customClaim.Value)
				if mapping.SourceValue != "*" {
					break
				}
			}
		}

	case model.MappingSourceStatic:
		// Try to parse as JSON, fall back to string
		var jsonValue any
		err := json.Unmarshal([]byte(mapping.SourceValue), &jsonValue)
		if err == nil {
			claims[mapping.ClaimName] = jsonValue
		} else {
			claims[mapping.ClaimName] = mapping.SourceValue
		}
	}
}
