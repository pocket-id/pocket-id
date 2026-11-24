package oidc

import (
	"context"
	"errors"
	"net/url"
	"time"

	"github.com/lestrrat-go/jwx/v4/jwt"
	"github.com/pocket-id/pocket-id/backend/internal/apperror"
	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/dto"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
	"gorm.io/gorm"
)

type endSessionService struct {
	db                 *gorm.DB
	store              *Store
	signer             TokenSigner
	baseURL            string
	validateCIMDClient func(context.Context, string) error
}

func newEndSessionService(db *gorm.DB, store *Store, signer TokenSigner, baseURL string) *endSessionService {
	return &endSessionService{
		db:      db,
		store:   store,
		signer:  signer,
		baseURL: baseURL,
	}
}

// endSessionRequest is a validated RP-initiated logout
type endSessionRequest struct {
	userID     string
	clientID   string
	clientName string
	// sessionID is the Pocket ID session the ID token was issued in, which is empty for ID tokens without a sid
	sessionID   string
	idTokenJTI  string
	callbackURL string
	state       string
}

// resolve validates the id_token_hint of a logout request and returns the logout it asks for
// A signed-in caller must be the user the ID token was issued to
func (s *endSessionService) resolve(ctx context.Context, input dto.OidcLogoutDto, userID string) (endSessionRequest, error) {
	if input.IdTokenHint == "" {
		return endSessionRequest{}, apperror.TokenInvalid()
	}

	token, err := s.verifyIDTokenHint(input.IdTokenHint)
	if err != nil {
		return endSessionRequest{}, apperror.TokenInvalid()
	}

	clientIDs, ok := token.Audience()
	if !ok || len(clientIDs) == 0 {
		return endSessionRequest{}, apperror.TokenInvalid()
	}
	clientID := clientIDs[0]
	if input.ClientId != "" && clientID != input.ClientId {
		return endSessionRequest{}, apperror.OidcClientIDNotMatching()
	}

	subject, ok := token.Subject()
	if !ok || subject == "" {
		return endSessionRequest{}, apperror.TokenInvalid()
	}
	if userID != "" && subject != userID {
		return endSessionRequest{}, apperror.TokenInvalid()
	}

	idTokenJTI, ok := token.JwtID()
	if !ok {
		return endSessionRequest{}, apperror.TokenInvalid()
	}
	sessionID, _ := jwt.Get[string](token, common.SessionIDClaim)

	authorizedClient, err := s.loadAuthorizedClientSnapshot(ctx, clientID, subject)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return endSessionRequest{}, apperror.OidcMissingAuthorization()
	}
	if err != nil {
		return endSessionRequest{}, err
	}

	var callbackURL string
	if authorizedClient.Client.IsMetadataDocument() {
		validateCIMD := s.validateCIMDClient
		if validateCIMD == nil && s.store != nil {
			if resolver, ok := s.store.clientResolver.(*cimdClientResolver); ok {
				validateCIMD = resolver.policy.AllowCIMDClient
			}
		}
		cimdAllowed := !authorizedClient.Client.Disabled &&
			authorizedClient.Client.MetadataExpiresAt != nil &&
			isCIMDClientCompatible(authorizedClient.Client) &&
			(s.store == nil || !s.store.isCIMDInvalidatedForContext(ctx, clientID)) &&
			(validateCIMD == nil || validateCIMD(ctx, clientID) == nil)
		// Do not unconditionally extend expired CIMD metadata by the 24h outage allowance: resolve expired metadata first so reachable changes or mismatches take effect, and recheck surviving consent afterward
		if cimdAllowed && authorizedClient.Client.MetadataExpiresAt.ToTime().Before(time.Now()) {
			if s.store == nil || s.store.clientResolver == nil {
				cimdAllowed = false
			} else {
				_, resolveErr := s.store.resolvePersistedClient(ctx, clientID)
				// Always reload consent after discovery, including on errors such as ErrCIMDClientIDMismatch which revoke consent
				refreshedAuth, reloadErr := s.loadAuthorizedClientSnapshot(ctx, clientID, subject)
				if errors.Is(reloadErr, gorm.ErrRecordNotFound) {
					return endSessionRequest{}, apperror.OidcMissingAuthorization()
				}
				if reloadErr != nil {
					return endSessionRequest{}, reloadErr
				}
				authorizedClient = refreshedAuth
				if resolveErr != nil {
					cimdAllowed = false
				} else {
					cimdAllowed = !authorizedClient.Client.Disabled &&
						isCIMDMetadataActive(authorizedClient.Client, time.Now()) &&
						!s.store.isCIMDInvalidatedForContext(ctx, clientID) &&
						(validateCIMD == nil || validateCIMD(ctx, clientID) == nil)
				}
			}
		}
		if !cimdAllowed {
			// Suppress post-logout redirects for invalidated, disabled, expired-beyond-stale, or disallowed CIMD clients so surviving consents cannot redirect to untrusted callbacks
			callbackURL = ""
		} else {
			callbackURL, err = cimdLogoutCallbackURL(&authorizedClient.Client, input.PostLogoutRedirectUri)
			if err != nil {
				return endSessionRequest{}, err
			}
		}
	} else {
		callbackURL, err = logoutCallbackURL(&authorizedClient.Client, input.PostLogoutRedirectUri)
		if err != nil {
			return endSessionRequest{}, err
		}
	}

	return endSessionRequest{
		userID:      subject,
		clientID:    clientID,
		clientName:  authorizedClient.Client.Name,
		sessionID:   sessionID,
		idTokenJTI:  idTokenJTI,
		callbackURL: callbackURL,
		state:       input.State,
	}, nil
}

// loadAuthorizedClientSnapshot reads user consent and the client's callback metadata in a single joined query so a concurrent CIMD refresh cannot pair revoked consent with newly written callbacks even inside a READ COMMITTED caller transaction
func (s *endSessionService) loadAuthorizedClientSnapshot(ctx context.Context, clientID, userID string) (model.UserAuthorizedOidcClient, error) {
	var authorizedClient model.UserAuthorizedOidcClient
	err := dbFromContext(ctx, s.db).
		InnerJoins("Client").
		First(&authorizedClient, "user_authorized_oidc_clients.client_id = ? AND user_authorized_oidc_clients.user_id = ?", clientID, userID).
		Error
	return authorizedClient, err
}

// revokeClientGrant revokes the grant the ID token was issued with, unless it requested offline_access
func (s *endSessionService) revokeClientGrant(ctx context.Context, req endSessionRequest) error {
	return s.store.RevokeSessionsByIDTokenHint(ctx, req.userID, req.clientID, req.idTokenJTI)
}

func (s *endSessionService) verifyIDTokenHint(tokenString string) (jwt.Token, error) {
	alg, err := s.signer.GetKeyAlg()
	if err != nil {
		return nil, err
	}

	token, err := jwt.ParseString(
		tokenString,
		jwt.WithValidate(true),
		jwt.WithKey(alg, s.signer.GetPrivateKey()),
		jwt.WithAcceptableSkew(time.Minute),
		jwt.WithResetValidators(true),
		jwt.WithIssuer(s.baseURL),
		jwt.WithValidator(jwt.IsIssuedAtValid()),
		jwt.WithValidator(jwt.IsNbfValid()),
	)
	if err != nil {
		return nil, err
	}

	// id_token_hint must be an ID token, never an access token (both are signed with the same
	// key). An expired ID token is still accepted here, as required by OIDC RP-Initiated Logout.
	tokenType, err := jwt.Get[string](token, common.TokenTypeClaim)
	if err != nil || tokenType != string(IDTokenType) {
		return nil, apperror.TokenInvalid()
	}

	return token, nil
}

func logoutCallbackURL(client *model.OidcClient, inputLogoutCallbackURL string) (string, error) {
	if len(client.LogoutCallbackURLs) == 0 {
		return "", nil
	}
	if inputLogoutCallbackURL == "" {
		return client.LogoutCallbackURLs[0], nil
	}

	matched, err := utils.GetCallbackURLFromList(client.LogoutCallbackURLs, inputLogoutCallbackURL)
	if err != nil || matched == "" {
		return "", apperror.OidcInvalidCallbackURL()
	}

	return matched, nil
}

// cimdLogoutCallbackURL enforces exact string matching per OIDC RP-Initiated Logout §3 (no wildcards, fragments, or unregistered loopback ports)
func cimdLogoutCallbackURL(client *model.OidcClient, inputLogoutCallbackURL string) (string, error) {
	if len(client.LogoutCallbackURLs) == 0 {
		return "", nil
	}
	if inputLogoutCallbackURL == "" {
		matched := matchExactRedirectURI([]string{client.LogoutCallbackURLs[0]}, "")
		if matched == nil {
			return "", nil
		}
		return matched.String(), nil
	}

	matched := matchExactRedirectURI([]string(client.LogoutCallbackURLs), inputLogoutCallbackURL)
	if matched == nil {
		return "", apperror.OidcInvalidCallbackURL()
	}
	return matched.String(), nil
}

func appendStateToURL(callbackURL string, state string) string {
	parsed, err := url.Parse(callbackURL)
	if err != nil {
		return callbackURL
	}
	if state != "" {
		q := parsed.Query()
		q.Set("state", state)
		parsed.RawQuery = q.Encode()
	}
	return parsed.String()
}
