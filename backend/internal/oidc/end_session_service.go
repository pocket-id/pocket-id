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

	var authorizedClient model.UserAuthorizedOidcClient
	err = s.db.
		WithContext(ctx).
		Preload("Client").
		First(&authorizedClient, "client_id = ? AND user_id = ?", clientID, subject).
		Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return endSessionRequest{}, apperror.OidcMissingAuthorization()
	}
	if err != nil {
		return endSessionRequest{}, err
	}

	callbackURL, err := logoutCallbackURL(&authorizedClient.Client, input.PostLogoutRedirectUri)
	if err != nil {
		return endSessionRequest{}, err
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
