package oidc

import (
	"context"
	"net/url"
	"time"

	"github.com/lestrrat-go/jwx/v3/jwt"
	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/dto"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
	"gorm.io/gorm"
)

type endSessionService struct {
	db      *gorm.DB
	store   *Store
	signer  TokenSigner
	baseURL string
}

func newEndSessionService(db *gorm.DB, store *Store, signer TokenSigner, baseURL string) *endSessionService {
	return &endSessionService{
		db:      db,
		store:   store,
		signer:  signer,
		baseURL: baseURL,
	}
}

// endSession revokes the sessions belonging to the ID token hint and returns the
// client's post-logout callback URL (empty if none is configured).
func (s *endSessionService) endSession(ctx context.Context, input dto.OidcLogoutDto, userID string) (string, error) {
	if input.IdTokenHint == "" {
		return "", &common.TokenInvalidError{}
	}

	token, err := s.verifyIDTokenHint(input.IdTokenHint)
	if err != nil {
		return "", &common.TokenInvalidError{}
	}

	clientIDs, ok := token.Audience()
	if !ok || len(clientIDs) == 0 {
		return "", &common.TokenInvalidError{}
	}
	clientID := clientIDs[0]
	if input.ClientId != "" && clientID != input.ClientId {
		return "", &common.OidcClientIdNotMatchingError{}
	}

	subject, ok := token.Subject()
	if !ok || subject == "" {
		return "", &common.TokenInvalidError{}
	}
	if userID != "" && subject != userID {
		return "", &common.TokenInvalidError{}
	}
	userID = subject

	idTokenJTI, ok := token.JwtID()
	if !ok {
		return "", &common.TokenInvalidError{}
	}

	var callbackURL string
	err = withTx(ctx, s.db, func(ctx context.Context) error {
		var authorizedClient model.UserAuthorizedOidcClient
		err := dbFromContext(ctx, s.db).
			Preload("Client").
			First(&authorizedClient, "client_id = ? AND user_id = ?", clientID, userID).
			Error
		if err != nil {
			return &common.OidcMissingAuthorizationError{}
		}

		callbackURL, err = logoutCallbackURL(&authorizedClient.Client, input.PostLogoutRedirectUri)
		if err != nil {
			return err
		}

		return s.store.RevokeSessionsByIDTokenHint(ctx, userID, clientID, idTokenJTI)
	})
	if err != nil {
		return "", err
	}

	return callbackURL, nil
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
	var tokenType string
	if err := token.Get(common.TokenTypeClaim, &tokenType); err != nil || tokenType != idTokenType {
		return nil, &common.TokenInvalidError{}
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
		return "", &common.OidcInvalidCallbackURLError{}
	}

	return matched, nil
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
