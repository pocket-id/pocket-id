package oidc

import (
	"context"
	"encoding/json"
	"errors"
	"slices"

	"github.com/ory/fosite"
	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	"gorm.io/gorm"
)

const (
	idTokenType = "id-token"
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

	if !IsUserGroupAllowedToAuthorize(user, client.OidcClient) {
		return fosite.ErrAccessDenied.WithHint("You are not allowed to access this service.")
	}

	return nil
}

// applyIDTokenClaims applies the claims of a user to the ID token claims in the session based on the requested scopes.
func (s *ClaimsService) applyIDTokenClaims(ctx context.Context, session *Session, scopes fosite.Arguments) error {
	userID := session.Subject
	if userID == "" {
		return nil
	}

	claims, err := s.GetUserClaims(ctx, userID, scopes)
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
	idTokenClaims.Extra[common.TokenTypeClaim] = idTokenType
	if session.AuthenticationMethod != "" {
		idTokenClaims.AuthenticationMethodsReferences = []string{session.AuthenticationMethod}
	}
}

// GetUserClaims retrieves the claims for a user based on the requested scopes. It includes standard claims
// like "sub" and "email" as well as any custom claims defined for the user or their groups.
func (s *ClaimsService) GetUserClaims(ctx context.Context, userID string, scopes []string) (map[string]any, error) {
	db := dbFromContext(ctx, s.db)

	var user model.User
	err := db.
		Preload("UserGroups").
		First(&user, "id = ?", userID).
		Error
	if err != nil {
		return nil, err
	}

	claims := make(map[string]any, 10)

	if slices.Contains(scopes, "profile") {
		customClaims, err := s.customClaims.GetCustomClaimsForUserWithUserGroups(ctx, user.ID, db)
		if err != nil {
			return nil, err
		}

		for _, customClaim := range customClaims {
			// A custom claim value can be a JSON document or a plain string
			var jsonValue any
			if err := json.Unmarshal([]byte(customClaim.Value), &jsonValue); err == nil {
				claims[customClaim.Key] = jsonValue
			} else {
				claims[customClaim.Key] = customClaim.Value
			}
		}

		claims["given_name"] = user.FirstName
		claims["family_name"] = user.LastName
		claims["name"] = user.FullName()
		claims["display_name"] = user.DisplayName
		claims["preferred_username"] = user.Username
		claims["picture"] = s.baseURL + "/api/users/" + user.ID + "/profile-picture.png"
	}

	claims["sub"] = user.ID

	// Only release the email claims when the user actually has an email. Emitting
	// email_verified alongside a null/absent email (OIDC Core §5.1) is malformed and can
	// mislead relying parties that key trust decisions on email_verified.
	if slices.Contains(scopes, "email") && user.Email != nil && *user.Email != "" {
		claims["email"] = *user.Email
		claims["email_verified"] = user.EmailVerified
	}

	if slices.Contains(scopes, "groups") {
		userGroups := make([]string, len(user.UserGroups))
		for i, group := range user.UserGroups {
			userGroups[i] = group.Name
		}
		claims["groups"] = userGroups
	}

	return claims, nil
}
