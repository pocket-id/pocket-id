package webauthn

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/go-webauthn/webauthn/protocol"
	gowebauthn "github.com/go-webauthn/webauthn/webauthn"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
)

// authenticationMethodPhishingResistant identifies phishing-resistant authentication, such as passkeys
// It must match the value emitted by the JWT service in the access token's "amr" claim
const authenticationMethodPhishingResistant = "phr"

type Service struct {
	db        *gorm.DB
	webAuthn  *gowebauthn.WebAuthn
	signer    TokenService
	auditLog  AuditLogger
	appConfig AppConfigProvider
}

func newService(deps Dependencies) (*Service, error) {
	wa, err := gowebauthn.New(&gowebauthn.Config{
		RPDisplayName: deps.AppConfig.GetDbConfig().AppName.Value,
		RPID:          utils.GetHostnameFromURL(deps.AppURL),
		RPOrigins:     []string{deps.AppURL},
		AuthenticatorSelection: protocol.AuthenticatorSelection{
			UserVerification: protocol.VerificationRequired,
		},
		Timeouts: gowebauthn.TimeoutsConfig{
			Login: gowebauthn.TimeoutConfig{
				Enforce:    true,
				Timeout:    time.Second * 60,
				TimeoutUVD: time.Second * 60,
			},
			Registration: gowebauthn.TimeoutConfig{
				Enforce:    true,
				Timeout:    time.Second * 60,
				TimeoutUVD: time.Second * 60,
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to init webauthn object: %w", err)
	}

	return &Service{
		db:        deps.DB,
		webAuthn:  wa,
		signer:    deps.Signer,
		auditLog:  deps.AuditLog,
		appConfig: deps.AppConfig,
	}, nil
}

func (s *Service) BeginRegistration(ctx context.Context, userID string) (*PublicKeyCredentialCreationOptions, error) {
	tx := s.db.Begin()
	defer func() {
		tx.Rollback()
	}()

	s.updateWebAuthnConfig()

	var user model.User
	err := tx.
		WithContext(ctx).
		Preload("Credentials").
		Find(&user, "id = ?", userID).
		Error
	if err != nil {
		return nil, fmt.Errorf("failed to load user: %w", err)
	}

	options, session, err := s.webAuthn.BeginRegistration(
		&user,
		gowebauthn.WithResidentKeyRequirement(protocol.ResidentKeyRequirementRequired),
		gowebauthn.WithExclusions(user.WebAuthnCredentialDescriptors()),
		gowebauthn.WithExtensions(map[string]any{"credProps": true}), // Required for Firefox Android to properly save the key in Google password manager
	)
	if err != nil {
		return nil, fmt.Errorf("failed to begin WebAuthn registration: %w", err)
	}

	sessionToStore := &WebauthnSession{
		ExpiresAt:        datatype.DateTime(session.Expires),
		Challenge:        session.Challenge,
		CredentialParams: session.CredParams,
		UserVerification: string(session.UserVerification),
	}

	err = tx.
		WithContext(ctx).
		Create(&sessionToStore).
		Error
	if err != nil {
		return nil, fmt.Errorf("failed to save WebAuthn session: %w", err)
	}

	err = tx.Commit().Error
	if err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return &PublicKeyCredentialCreationOptions{
		Response:  options.Response,
		SessionID: sessionToStore.ID,
		Timeout:   s.webAuthn.Config.Timeouts.Registration.Timeout,
	}, nil
}

func (s *Service) VerifyRegistration(ctx context.Context, sessionID string, userID string, r *http.Request, ipAddress string) (model.WebauthnCredential, error) {
	tx := s.db.Begin()
	defer func() {
		tx.Rollback()
	}()

	// Load & delete the session row
	var storedSession WebauthnSession
	err := tx.
		WithContext(ctx).
		Clauses(clause.Returning{}).
		Delete(&storedSession, "id = ?", sessionID).
		Error
	if err != nil {
		return model.WebauthnCredential{}, fmt.Errorf("failed to load WebAuthn session: %w", err)
	}

	session := gowebauthn.SessionData{
		Challenge:  storedSession.Challenge,
		Expires:    storedSession.ExpiresAt.ToTime(),
		CredParams: storedSession.CredentialParams,
		UserID:     []byte(userID),
	}

	var user model.User
	err = tx.
		WithContext(ctx).
		Find(&user, "id = ?", userID).
		Error
	if err != nil {
		return model.WebauthnCredential{}, fmt.Errorf("failed to load user: %w", err)
	}

	credential, err := s.webAuthn.FinishRegistration(&user, session, r)
	if err != nil {
		return model.WebauthnCredential{}, fmt.Errorf("failed to finish WebAuthn registration: %w", err)
	}

	// Determine passkey name using AAGUID and User-Agent
	passkeyName := s.determinePasskeyName(credential.Authenticator.AAGUID)

	credentialToStore := model.WebauthnCredential{
		Name:            passkeyName,
		CredentialID:    credential.ID,
		AttestationType: credential.AttestationType,
		PublicKey:       credential.PublicKey,
		Transport:       credential.Transport,
		UserID:          user.ID,
		BackupEligible:  credential.Flags.BackupEligible,
		BackupState:     credential.Flags.BackupState,
	}
	err = tx.
		WithContext(ctx).
		Create(&credentialToStore).
		Error
	if err != nil {
		return model.WebauthnCredential{}, fmt.Errorf("failed to store WebAuthn credential: %w", err)
	}

	auditLogData := model.AuditLogData{"credentialID": hex.EncodeToString(credential.ID), "passkeyName": passkeyName}
	s.auditLog.Create(ctx, model.AuditLogEventPasskeyAdded, ipAddress, r.UserAgent(), userID, auditLogData, tx)

	err = tx.Commit().Error
	if err != nil {
		return model.WebauthnCredential{}, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return credentialToStore, nil
}

func (s *Service) determinePasskeyName(aaguid []byte) string {
	// First try to identify by AAGUID using a combination of builtin + MDS
	authenticatorName := utils.GetAuthenticatorName(aaguid)
	if authenticatorName != "" {
		return authenticatorName
	}

	return "New Passkey" // Default fallback
}

func (s *Service) BeginLogin(ctx context.Context) (*PublicKeyCredentialRequestOptions, error) {
	options, session, err := s.webAuthn.BeginDiscoverableLogin()
	if err != nil {
		return nil, err
	}

	sessionToStore := &WebauthnSession{
		ExpiresAt:        datatype.DateTime(session.Expires),
		Challenge:        session.Challenge,
		UserVerification: string(session.UserVerification),
	}

	err = s.db.
		WithContext(ctx).
		Create(&sessionToStore).
		Error
	if err != nil {
		return nil, err
	}

	return &PublicKeyCredentialRequestOptions{
		Response:  options.Response,
		SessionID: sessionToStore.ID,
		Timeout:   s.webAuthn.Config.Timeouts.Registration.Timeout,
	}, nil
}

func (s *Service) VerifyLogin(ctx context.Context, sessionID string, credentialAssertionData *protocol.ParsedCredentialAssertionData, ipAddress, userAgent string) (retUser model.User, retToken string, err error) {
	var userID string
	tx := s.db.Begin()

	// With a deferred call, check if the error indicates a sign in failure that needs to be logged
	// Important: in this method, do not created "err" variables with a narrower scope (e.g. avoid `if err := ...; err {` constructs)
	defer func() {
		tx.Rollback()

		if err != nil {
			s.auditLog.CreateSignInFailure(ctx, ipAddress, userAgent, userID)
		}
	}()

	// Load & delete the session row
	var storedSession WebauthnSession
	err = tx.
		WithContext(ctx).
		Clauses(clause.Returning{}).
		Delete(&storedSession, "id = ?", sessionID).
		Error
	if err != nil {
		err = fmt.Errorf("failed to load WebAuthn session: %w", err)
		return model.User{}, "", err
	}

	session := gowebauthn.SessionData{
		Challenge: storedSession.Challenge,
		Expires:   storedSession.ExpiresAt.ToTime(),
	}

	var user *model.User
	_, err = s.webAuthn.ValidateDiscoverableLogin(
		func(_, userHandle []byte) (gowebauthn.User, error) {
			innerErr := tx.
				WithContext(ctx).
				Preload("Credentials").
				First(&user, "id = ?", string(userHandle)).
				Error
			if innerErr != nil {
				return nil, innerErr
			}
			return user, nil
		},
		session,
		credentialAssertionData,
	)
	if err != nil {
		return model.User{}, "", err
	}

	// Assign the user ID to the userID variable that can be used for logs
	userID = user.ID
	if user.Disabled {
		// Need to assign to err
		err = &common.UserDisabledError{}
		return model.User{}, "", err
	}

	token, err := s.signer.GenerateAccessToken(*user, authenticationMethodPhishingResistant)
	if err != nil {
		return model.User{}, "", err
	}

	s.auditLog.CreateNewSignInWithEmail(ctx, ipAddress, userAgent, user.ID, tx)

	err = tx.Commit().Error
	if err != nil {
		return model.User{}, "", err
	}

	return *user, token, nil
}

func (s *Service) ListCredentials(ctx context.Context, userID string) ([]model.WebauthnCredential, error) {
	var credentials []model.WebauthnCredential
	err := s.db.
		WithContext(ctx).
		Find(&credentials, "user_id = ?", userID).
		Error
	if err != nil {
		return nil, err
	}
	return credentials, nil
}

func (s *Service) DeleteCredential(ctx context.Context, userID string, credentialID string, ipAddress string, userAgent string, actorUserID string) error {
	tx := s.db.Begin()
	defer func() {
		tx.Rollback()
	}()

	credential := &model.WebauthnCredential{}
	result := tx.
		WithContext(ctx).
		Clauses(clause.Returning{}).
		Delete(credential, "id = ? AND user_id = ?", credentialID, userID)
	if result.Error != nil {
		return fmt.Errorf("failed to delete record: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	auditLogData := model.AuditLogData{"credentialID": hex.EncodeToString(credential.CredentialID), "passkeyName": credential.Name}
	if actorUserID != "" && actorUserID != userID {
		var actor model.User
		err := tx.
			WithContext(ctx).
			First(&actor, "id = ?", actorUserID).
			Error
		if err != nil {
			return fmt.Errorf("failed to load actor user: %w", err)
		}
		auditLogData["actorUserID"] = actorUserID
		auditLogData["actorUsername"] = actor.Username
	}
	s.auditLog.Create(ctx, model.AuditLogEventPasskeyRemoved, ipAddress, userAgent, userID, auditLogData, tx)

	err := tx.Commit().Error
	if err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

func (s *Service) UpdateCredential(ctx context.Context, userID, credentialID, name string) (model.WebauthnCredential, error) {
	tx := s.db.Begin()
	defer func() {
		tx.Rollback()
	}()

	var credential model.WebauthnCredential
	err := tx.
		WithContext(ctx).
		Where("id = ? AND user_id = ?", credentialID, userID).
		First(&credential).
		Error
	if err != nil {
		return credential, err
	}

	credential.Name = name

	err = tx.
		WithContext(ctx).
		Save(&credential).
		Error
	if err != nil {
		return credential, err
	}

	err = tx.Commit().Error
	if err != nil {
		return credential, err
	}

	return credential, nil
}

// updateWebAuthnConfig updates the WebAuthn configuration with the app name as it can change during runtime
func (s *Service) updateWebAuthnConfig() {
	s.webAuthn.Config.RPDisplayName = s.appConfig.GetDbConfig().AppName.Value
}

func (s *Service) CreateReauthenticationTokenWithAccessToken(ctx context.Context, accessToken string, ipAddress string, userAgent string) (retToken string, retErr error) {
	var userID string
	tx := s.db.Begin()

	// With a deferred call, check if the error indicates a sign in failure that needs to be logged
	// Important: in this method, do not created "err" variables with a narrower scope (e.g. avoid `if err := ...; err {` constructs)
	defer func() {
		tx.Rollback()
		if retErr != nil {
			s.auditLog.CreateSignInFailure(ctx, ipAddress, userAgent, userID)
		}
	}()

	token, err := s.signer.VerifyAccessToken(accessToken)
	if err != nil {
		// Need to assign to err
		err = fmt.Errorf("invalid access token: %w", err)
		return "", err
	}

	userID, ok := token.Subject()
	if !ok {
		// Need to assign to err
		err = errors.New("access token does not contain user ID")
		return "", err
	}

	authenticationMethod, err := s.signer.GetAuthenticationMethod(token)
	if err != nil {
		return "", err
	}
	if authenticationMethod != authenticationMethodPhishingResistant {
		// Need to assign to err
		err = &common.ReauthenticationRequiredError{}
		return "", err
	}

	// Check if token is issued less than a minute ago
	tokenExpiration, ok := token.IssuedAt()
	if !ok || time.Since(tokenExpiration) > time.Minute {
		// Need to assign to err
		err = &common.ReauthenticationRequiredError{}
		return "", err
	}

	var user model.User
	err = tx.
		WithContext(ctx).
		First(&user, "id = ?", userID).
		Error
	if err != nil {
		// Need to assign to err
		err = fmt.Errorf("failed to load user: %w", err)
		return "", err
	}

	reauthToken, err := s.createReauthenticationToken(ctx, tx, user.ID)
	if err != nil {
		return "", err
	}

	err = tx.Commit().Error
	if err != nil {
		return "", err
	}

	return reauthToken, nil
}

func (s *Service) CreateReauthenticationTokenWithWebauthn(ctx context.Context, sessionID string, credentialAssertionData *protocol.ParsedCredentialAssertionData) (string, error) {
	tx := s.db.Begin()
	defer func() {
		tx.Rollback()
	}()

	// Retrieve and delete the session
	var storedSession WebauthnSession
	err := tx.
		WithContext(ctx).
		Clauses(clause.Returning{}).
		Delete(&storedSession, "id = ? AND expires_at > ?", sessionID, datatype.DateTime(time.Now())).
		Error
	if err != nil {
		return "", fmt.Errorf("failed to load WebAuthn session: %w", err)
	}

	session := gowebauthn.SessionData{
		Challenge: storedSession.Challenge,
		Expires:   storedSession.ExpiresAt.ToTime(),
	}

	// Validate the credential assertion
	var user *model.User
	_, err = s.webAuthn.ValidateDiscoverableLogin(func(_, userHandle []byte) (gowebauthn.User, error) {
		innerErr := tx.
			WithContext(ctx).
			Preload("Credentials").
			First(&user, "id = ?", string(userHandle)).
			Error
		if innerErr != nil {
			return nil, innerErr
		}
		return user, nil
	}, session, credentialAssertionData)

	if err != nil || user == nil {
		return "", err
	}

	// Create reauthentication token
	token, err := s.createReauthenticationToken(ctx, tx, user.ID)
	if err != nil {
		return "", err
	}

	err = tx.Commit().Error
	if err != nil {
		return "", err
	}

	return token, nil
}

func (s *Service) ConsumeReauthenticationToken(ctx context.Context, tx *gorm.DB, token string, userID string) (time.Time, error) {
	hashedToken := utils.CreateSha256Hash(token)
	var reauthToken ReauthenticationToken
	result := tx.WithContext(ctx).
		Clauses(clause.Returning{}).
		Delete(&reauthToken, "token = ? AND user_id = ? AND expires_at > ?", hashedToken, userID, datatype.DateTime(time.Now()))

	if result.Error != nil {
		return time.Time{}, result.Error
	}
	if result.RowsAffected == 0 {
		return time.Time{}, &common.ReauthenticationRequiredError{}
	}
	return reauthToken.CreatedAt.UTC(), nil
}

func (s *Service) createReauthenticationToken(ctx context.Context, tx *gorm.DB, userID string) (string, error) {
	token, err := utils.GenerateRandomAlphanumericString(32)
	if err != nil {
		return "", err
	}

	reauthToken := ReauthenticationToken{
		Token:     utils.CreateSha256Hash(token),
		ExpiresAt: datatype.DateTime(time.Now().Add(3 * time.Minute)),
		UserID:    userID,
	}

	err = tx.WithContext(ctx).Create(&reauthToken).Error
	if err != nil {
		return "", err
	}

	return token, nil
}
