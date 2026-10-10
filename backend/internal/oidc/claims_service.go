package oidc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"

	"github.com/ory/fosite"
	fositejwt "github.com/ory/fosite/token/jwt"
	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	"gorm.io/gorm"
)

type TokenType string

const (
	IDTokenType     TokenType = "id-token"
	AccessTokenType TokenType = "access-token"
	UserInfoType    TokenType = "user-info"
)

type ClaimsService struct {
	db           *gorm.DB
	customClaims CustomClaimSource
	baseURL      string
	signer       TokenSigner
}

func newClaimsService(db *gorm.DB, customClaims CustomClaimSource, baseURL string, signer TokenSigner) *ClaimsService {
	return &ClaimsService{
		db:           db,
		customClaims: customClaims,
		baseURL:      baseURL,
		signer:       signer,
	}
}

// errClaimsUserNotFound is returned when the user whose claims are requested no longer exists
// Callers map it to the error their endpoint expects, e.g. invalid_grant on the token endpoint
var errClaimsUserNotFound = errors.New("user not found")

// userClaimsSource holds everything needed to build the claims of a user for a client
// It is loaded once per request so every token built from it sees the same data and no query is repeated
type userClaimsSource struct {
	user         model.User
	policy       model.OidcClaimMappingPolicy
	customClaims []model.CustomClaim
}

// loadUserClaimsSource reads the user with its groups, the claim mapping policy of the client and, when the policy needs them, the custom claims of the user
func (s *ClaimsService) loadUserClaimsSource(ctx context.Context, userID string, clientID string) (*userClaimsSource, error) {
	var src userClaimsSource
	err := withTx(ctx, s.db, func(ctx context.Context) error {
		db := dbFromContext(ctx, s.db)

		// Load the user with its groups, which back the groups claim, the group restriction and the group custom claims
		err := db.
			Preload("UserGroups").
			First(&src.user, "id = ?", userID).
			Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errClaimsUserNotFound
		}
		if err != nil {
			return err
		}

		// Load the policy assigned to the client, falling back to the default one
		policy, err := s.GetClaimMappingPolicyByClientID(ctx, clientID)
		if err != nil {
			return fmt.Errorf("failed to load claim mapping policy: %w", err)
		}
		src.policy = *policy

		// Custom claims cost an extra query, so they are only read when the policy can release one
		// A service wired without a custom claim source simply has none
		if s.customClaims == nil || !policyHasCustomClaimMapping(src.policy) {
			return nil
		}
		src.customClaims, err = s.customClaims.GetCustomClaimsForUserWithUserGroups(ctx, src.user.ID, db)
		return err
	})
	if err != nil {
		return nil, err
	}

	return &src, nil
}

func policyHasCustomClaimMapping(policy model.OidcClaimMappingPolicy) bool {
	return slices.ContainsFunc(policy.ClaimMappings, func(mapping model.OidcClaimMapping) bool {
		return mapping.SourceType == model.MappingSourceCustomClaim
	})
}

// loadGrantClaimsSource loads the claims source of the user behind a grant and re-checks, at token-issuance time, that the user is still allowed to obtain tokens for the client
// Grants without a resource owner (e.g. client_credentials) carry an empty subject, so they have no user to validate and get a nil source
func (s *ClaimsService) loadGrantClaimsSource(ctx context.Context, userID string, client Client) (*userClaimsSource, error) {
	if userID == "" {
		return nil, nil
	}

	src, err := s.loadUserClaimsSource(ctx, userID, client.GetID())
	if errors.Is(err, errClaimsUserNotFound) {
		return nil, fosite.ErrInvalidGrant.WithHint("The user account no longer exists.")
	}
	if err != nil {
		return nil, err
	}

	if src.user.Disabled {
		return nil, fosite.ErrInvalidGrant.WithHint("The user account is disabled.")
	}

	if !IsUserGroupAllowedToAuthorize(src.user, client.OidcClient) {
		return nil, fosite.ErrAccessDenied.WithHint("You are not allowed to access this service.")
	}

	return src, nil
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

// applyTokenClaims applies the claims of a user to both the ID token and the access token of the session based on the granted scopes
func (s *ClaimsService) applyTokenClaims(session *Session, scopes fosite.Arguments, src *userClaimsSource) error {
	// Record the signing algorithm on the ID token header so fosite derives the at_hash/c_hash digest from it (e.g. RS384 -> SHA-384, ES512 -> SHA-512)
	// Without this the header is empty and fosite defaults to SHA-256, producing wrong hashes whenever the signing key is not a 256-bit algorithm
	// ToMap() strips "alg" before signing, so this never overrides the real JWS header
	// The signer is always wired in production; it is only nil in unit tests that do not assert hash correctness
	if s.signer != nil {
		alg, err := s.signer.GetKeyAlg()
		if err != nil {
			return err
		}
		session.IDTokenHeaders().Add("alg", alg.String())
	}

	// Both tokens are built from the same source, so they always agree on the user data
	applyUserClaimsToIDToken(session, src.user.ID, s.buildClaims(src, scopes, IDTokenType))
	applyUserClaimsToAccessToken(session, src.user.ID, s.buildClaims(src, scopes, AccessTokenType))
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

func applyUserClaimsToAccessToken(session *Session, userID string, claims map[string]any) {
	jwtClaims := session.GetJWTClaims().(*fositejwt.JWTClaims)
	jwtClaims.Extra = claims
}

// buildClaims applies the mappings of the policy that target the token type and match the scopes
// It works on already loaded data, so it never touches the database and cannot fail
func (s *ClaimsService) buildClaims(src *userClaimsSource, scopes []string, tokenType TokenType) map[string]any {
	claims := make(map[string]any, len(src.policy.ClaimMappings))
	for _, mapping := range src.policy.ClaimMappings {
		if ((mapping.IDToken && tokenType == IDTokenType) ||
			(mapping.AccessToken && tokenType == AccessTokenType) ||
			(mapping.UserInfo && tokenType == UserInfoType)) &&
			isScopeMatching(scopes, mapping.Scope) {
			s.applyUserClaims(claims, src.user, src.customClaims, mapping)
		}
	}

	return claims
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
				// A custom claim value can be a JSON document or a plain string
				var jsonValue any
				if err := json.Unmarshal([]byte(customClaim.Value), &jsonValue); err == nil {
					claims[claimName] = jsonValue
				} else {
					claims[claimName] = customClaim.Value
				}
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
