package service

import (
	"context"
	"log/slog"

	"github.com/ory/fosite"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/apperror"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
	"github.com/pocket-id/pocket-id/backend/internal/oidc"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
)

// RegisterDynamicClient implements OIDC Dynamic Client Registration (RFC 7591) core
// registration: it validates the requested redirect URIs against the configured
// allowlist, creates a new dynamically-registered OidcClient, and returns the
// plaintext client secret (for confidential clients) and plaintext registration
// access token alongside the persisted client.
func (s *OidcService) RegisterDynamicClient(ctx context.Context, input fosite.ClientRegistrationRequest) (model.OidcClient, string, string, error) {
	// The allowlist is a plain accessor, matching GetCIMDURLAllowlist; it reads the
	// cached config rather than issuing a request on behalf of this one.
	//nolint:contextcheck
	allowlist := s.appConfigService.GetDynamicClientRedirectUriAllowlist()
	if err := oidc.ValidateRegistrationRedirectURIs(input.RedirectURIs, allowlist); err != nil {
		return model.OidcClient{}, "", "", err
	}

	isPublic := input.TokenEndpointAuthMethod == "none"

	client := model.OidcClient{
		Name:         input.ClientName,
		CallbackURLs: datatype.StringList(input.RedirectURIs),
		IsPublic:     isPublic,
		PkceEnabled:  isPublic,
		ClientType:   model.OidcClientTypeDynamic,
	}

	var clientSecret string
	if !isPublic {
		secret, hash, err := generateClientSecret()
		if err != nil {
			return model.OidcClient{}, "", "", err
		}
		clientSecret = secret
		client.Secret = hash
	}

	regToken, regHash, err := generateRegistrationAccessToken()
	if err != nil {
		return model.OidcClient{}, "", "", err
	}
	client.RegistrationAccessTokenHash = &regHash

	if err := s.db.WithContext(ctx).Create(&client).Error; err != nil {
		return model.OidcClient{}, "", "", err
	}

	// Storage operations must run outside of the DB transaction/create above.
	// The logo_uri is best-effort: a bad or unreachable URL (including one
	// rejected by the SSRF guard in downloadAndSaveLogoFromURL) must not fail
	// registration.
	if input.LogoURI != "" {
		if err := s.downloadAndSaveLogoFromURL(ctx, client.ID, input.LogoURI, true); err != nil {
			slog.WarnContext(ctx, "Failed to download dynamic client logo from logo_uri, continuing without a logo",
				slog.String("client_id", client.ID),
				slog.Any("error", err),
			)
		}
	}

	return client, clientSecret, regToken, nil
}

// authenticateDynamicClient loads a client by ID and verifies that it is a
// dynamically-registered client whose registration access token matches the
// provided plaintext token.
//
// This authenticates without rotating, which suits deletion: the client and its
// token are about to be removed, so issuing a replacement would be pointless. Read
// and update use authenticateAndRotate instead, because their responses are
// required to carry a registration access token.
func (s *OidcService) authenticateDynamicClient(ctx context.Context, clientID, token string) (model.OidcClient, error) {
	var client model.OidcClient
	if err := s.db.WithContext(ctx).First(&client, "id = ?", clientID).Error; err != nil {
		// Intentionally return the same auth-failure error as the other failure
		// paths below, rather than leaking whether the client exists.
		return model.OidcClient{}, apperror.InvalidRegistrationToken()
	}
	if !client.IsDynamic() || client.RegistrationAccessTokenHash == nil {
		return model.OidcClient{}, apperror.InvalidRegistrationToken()
	}
	if bcrypt.CompareHashAndPassword([]byte(*client.RegistrationAccessTokenHash), []byte(token)) != nil {
		return model.OidcClient{}, apperror.InvalidRegistrationToken()
	}
	return client, nil
}

// rotateAuthenticatedToken replaces the client's registration access token, but only
// if previousHash is still the stored one.
//
// The condition is what prevents a lockout: if a concurrent request has already
// rotated, this write matches no row and the caller is told its token is no longer
// current, rather than being handed a replacement that the other request's write
// already superseded.
func (s *OidcService) rotateAuthenticatedToken(ctx context.Context, clientID, previousHash string) (string, error) {
	token, hash, err := generateRegistrationAccessToken()
	if err != nil {
		return "", err
	}
	res := s.db.WithContext(ctx).
		Model(&model.OidcClient{}).
		Where("id = ? AND registration_access_token_hash = ?", clientID, previousHash).
		Update("registration_access_token_hash", hash)
	if res.Error != nil {
		return "", res.Error
	}
	if res.RowsAffected == 0 {
		return "", apperror.InvalidRegistrationToken()
	}
	return token, nil
}

// authenticateAndRotate authenticates the caller and, in the same transaction,
// replaces the client's registration access token with a fresh one.
//
// RFC 7592 section 3 requires every client information response to carry a
// registration_access_token, but only a bcrypt hash is stored, so the token issued
// at registration cannot be reproduced on a later read. Section 2.1 and appendix
// A.1 resolve this by rotating on a read or update and returning the new value,
// which also bounds how long any single token stays valid.
//
// Two requests carrying the same valid token would otherwise both authenticate
// against the same stored hash and then both rotate. Each caller is handed a
// different new token, but only the last write survives, so one caller walks away
// holding a credential that does not work and is locked out of managing its
// registration, which RFC 7592 section 5 warns against.
//
// The write is therefore conditional on the hash that was just authenticated still
// being the stored one, which is what makes the loser fail rather than receive a
// dead token. The pair also runs in a transaction so the read and the conditional
// write see a consistent view; the condition is what provides the guarantee, and
// the transaction keeps the two statements from interleaving with an unrelated
// write to the same row.
func (s *OidcService) authenticateAndRotate(ctx context.Context, clientID, token string) (model.OidcClient, string, error) {
	var (
		client  model.OidcClient
		rotated string
	)
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var found model.OidcClient
		if err := tx.First(&found, "id = ?", clientID).Error; err != nil {
			return apperror.InvalidRegistrationToken()
		}
		if !found.IsDynamic() || found.RegistrationAccessTokenHash == nil {
			return apperror.InvalidRegistrationToken()
		}
		if bcrypt.CompareHashAndPassword([]byte(*found.RegistrationAccessTokenHash), []byte(token)) != nil {
			return apperror.InvalidRegistrationToken()
		}

		next, hash, err := generateRegistrationAccessToken()
		if err != nil {
			return err
		}
		res := tx.Model(&model.OidcClient{}).
			Where("id = ? AND registration_access_token_hash = ?", clientID, *found.RegistrationAccessTokenHash).
			Update("registration_access_token_hash", hash)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			// The hash changed between the read and the write, so another request
			// rotated first and this token is no longer current.
			return apperror.InvalidRegistrationToken()
		}

		client = found
		rotated = next
		return nil
	})
	if err != nil {
		return model.OidcClient{}, "", err
	}
	return client, rotated, nil
}

// GetDynamicClient implements RFC 7592 client configuration retrieval: it
// authenticates the caller via the registration access token and returns the
// current client configuration along with a freshly rotated registration access
// token, which the response is required to carry.
func (s *OidcService) GetDynamicClient(ctx context.Context, clientID, token string) (model.OidcClient, string, error) {
	return s.authenticateAndRotate(ctx, clientID, token)
}

// UpdateDynamicClient implements RFC 7592 client configuration update: it
// authenticates the caller, re-validates the requested redirect URIs against
// the configured allowlist, and persists the updated client metadata. It
// returns the plaintext client secret when a new one was (re)issued as part
// of a public-to-confidential transition (or "" otherwise), along with a freshly
// rotated registration access token, which the response is required to carry.
func (s *OidcService) UpdateDynamicClient(ctx context.Context, clientID, token string, input fosite.ClientRegistrationRequest) (model.OidcClient, string, string, error) {
	// Authenticated without rotating, so that a request rejected below leaves the
	// caller's token intact: burning a registration access token on a validation
	// failure would let a client lock itself out by sending a bad payload.
	client, err := s.authenticateDynamicClient(ctx, clientID, token)
	if err != nil {
		return model.OidcClient{}, "", "", err
	}
	//nolint:contextcheck
	allowlist := s.appConfigService.GetDynamicClientRedirectUriAllowlist()
	if err := oidc.ValidateRegistrationRedirectURIs(input.RedirectURIs, allowlist); err != nil {
		return model.OidcClient{}, "", "", err
	}
	client.Name = input.ClientName
	client.CallbackURLs = datatype.StringList(input.RedirectURIs)

	isPublic := input.TokenEndpointAuthMethod == "none"
	client.IsPublic = isPublic
	client.PkceEnabled = isPublic

	var clientSecret string
	switch {
	case !isPublic && client.Secret == "":
		// Public-to-confidential transition: this client never had a secret, so
		// one must be (re)issued now or it could never authenticate.
		secret, hash, err := generateClientSecret()
		if err != nil {
			return model.OidcClient{}, "", "", err
		}
		clientSecret = secret
		client.Secret = hash
	case isPublic:
		// Confidential-to-public transition: clear any stale secret hash.
		client.Secret = ""
	}

	// Persist only the columns RFC 7592 lets the client manage. A full Save() would
	// write back every field loaded at the start of this call, silently reverting a
	// concurrent admin change to security-relevant columns such as is_group_restricted
	// or client_type. A map is used so that zero values (e.g. clearing the secret on a
	// confidential-to-public transition) are still written.
	if err := s.db.WithContext(ctx).
		Model(&model.OidcClient{}).
		Where("id = ?", client.ID).
		Updates(map[string]any{
			"name":          client.Name,
			"callback_urls": client.CallbackURLs,
			"is_public":     client.IsPublic,
			"pkce_enabled":  client.PkceEnabled,
			"secret":        client.Secret,
		}).Error; err != nil {
		return model.OidcClient{}, "", "", err
	}

	// Storage operations must run outside of the DB transaction/save above.
	// The logo_uri is best-effort: a bad or unreachable URL (including one
	// rejected by the SSRF guard in downloadAndSaveLogoFromURL) must not fail
	// the update.
	if input.LogoURI != "" {
		if err := s.downloadAndSaveLogoFromURL(ctx, client.ID, input.LogoURI, true); err != nil {
			slog.WarnContext(ctx, "Failed to download dynamic client logo from logo_uri, continuing without a logo",
				slog.String("client_id", client.ID),
				slog.Any("error", err),
			)
		}
	}

	// Rotated only now that the update has been accepted and applied. The compare
	// against the authenticated hash still makes a concurrent rotation lose here.
	rotated, err := s.rotateAuthenticatedToken(ctx, client.ID, *client.RegistrationAccessTokenHash)
	if err != nil {
		return model.OidcClient{}, "", "", err
	}

	return client, clientSecret, rotated, nil
}

// DeleteDynamicClient implements RFC 7592 client configuration deletion: it
// authenticates the caller and removes the client.
func (s *OidcService) DeleteDynamicClient(ctx context.Context, clientID, token string) error {
	client, err := s.authenticateDynamicClient(ctx, clientID, token)
	if err != nil {
		return err
	}
	return s.db.WithContext(ctx).Delete(&client).Error
}

// generateClientSecret creates a new random client secret and its bcrypt hash
// for storage. It returns the plaintext secret (to be shown to the caller once)
// and the hash (to be persisted).
func generateClientSecret() (plaintext, hash string, err error) {
	secret, err := utils.GenerateRandomAlphanumericString(32)
	if err != nil {
		return "", "", err
	}
	hashed, err := bcrypt.GenerateFromPassword([]byte(secret), bcrypt.DefaultCost)
	if err != nil {
		return "", "", err
	}
	return secret, string(hashed), nil
}

// generateRegistrationAccessToken creates a new random registration access token
// (RFC 7591 section 3.2.1) and its bcrypt hash for storage.
func generateRegistrationAccessToken() (string, string, error) {
	token, err := utils.GenerateRandomAlphanumericString(48)
	if err != nil {
		return "", "", err
	}
	hashed, err := bcrypt.GenerateFromPassword([]byte(token), bcrypt.DefaultCost)
	if err != nil {
		return "", "", err
	}
	return token, string(hashed), nil
}
