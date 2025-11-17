package service

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
)

type WebAuthnService struct {
	db               *gorm.DB
	webAuthn         *webauthn.WebAuthn
	jwtService       *JwtService
	auditLogService  *AuditLogService
	appConfigService *AppConfigService
}

func NewWebAuthnService(db *gorm.DB, jwtService *JwtService, auditLogService *AuditLogService, appConfigService *AppConfigService) (*WebAuthnService, error) {
	wa, err := webauthn.New(&webauthn.Config{
		RPDisplayName: appConfigService.GetDbConfig().AppName.Value,
		RPID:          utils.GetHostnameFromURL(common.EnvConfig.AppURL),
		RPOrigins:     []string{common.EnvConfig.AppURL},
		AuthenticatorSelection: protocol.AuthenticatorSelection{
			UserVerification: protocol.VerificationRequired,
		},
		Timeouts: webauthn.TimeoutsConfig{
			Login: webauthn.TimeoutConfig{
				Enforce:    true,
				Timeout:    time.Second * 60,
				TimeoutUVD: time.Second * 60,
			},
			Registration: webauthn.TimeoutConfig{
				Enforce:    true,
				Timeout:    time.Second * 60,
				TimeoutUVD: time.Second * 60,
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to init webauthn object: %w", err)
	}

	return &WebAuthnService{
		db:               db,
		webAuthn:         wa,
		jwtService:       jwtService,
		auditLogService:  auditLogService,
		appConfigService: appConfigService,
	}, nil
}

func (s *WebAuthnService) BeginRegistration(ctx context.Context, userID string) (*model.PublicKeyCredentialCreationOptions, error) {
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
		webauthn.WithResidentKeyRequirement(protocol.ResidentKeyRequirementRequired),
		webauthn.WithExclusions(user.WebAuthnCredentialDescriptors()),
		webauthn.WithExtensions(map[string]any{"credProps": true}), // Required for Firefox Android to properly save the key in Google password manager
	)
	if err != nil {
		return nil, fmt.Errorf("failed to begin WebAuthn registration: %w", err)
	}

	sessionToStore := &model.WebauthnSession{
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

	return &model.PublicKeyCredentialCreationOptions{
		Response:  options.Response,
		SessionID: sessionToStore.ID,
		Timeout:   s.webAuthn.Config.Timeouts.Registration.Timeout,
	}, nil
}

func (s *WebAuthnService) VerifyRegistration(ctx context.Context, sessionID string, userID string, r *http.Request, ipAddress string) (model.WebauthnCredential, error) {
	tx := s.db.Begin()
	defer func() {
		tx.Rollback()
	}()

	// Load & delete the session row
	var storedSession model.WebauthnSession
	err := tx.
		WithContext(ctx).
		Clauses(clause.Returning{}).
		Delete(&storedSession, "id = ?", sessionID).
		Error
	if err != nil {
		return model.WebauthnCredential{}, fmt.Errorf("failed to load WebAuthn session: %w", err)
	}

	session := webauthn.SessionData{
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
	s.auditLogService.Create(ctx, model.AuditLogEventPasskeyAdded, ipAddress, r.UserAgent(), userID, auditLogData, tx)

	err = tx.Commit().Error
	if err != nil {
		return model.WebauthnCredential{}, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return credentialToStore, nil
}

func (s *WebAuthnService) determinePasskeyName(aaguid []byte) string {
	// First try to identify by AAGUID using a combination of builtin + MDS
	authenticatorName := utils.GetAuthenticatorName(aaguid)
	if authenticatorName != "" {
		return authenticatorName
	}

	return "New Passkey" // Default fallback
}

func (s *WebAuthnService) BeginLogin(ctx context.Context) (*model.PublicKeyCredentialRequestOptions, error) {
	options, session, err := s.webAuthn.BeginDiscoverableLogin()
	if err != nil {
		return nil, err
	}

	sessionToStore := &model.WebauthnSession{
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

	return &model.PublicKeyCredentialRequestOptions{
		Response:  options.Response,
		SessionID: sessionToStore.ID,
		Timeout:   s.webAuthn.Config.Timeouts.Registration.Timeout,
	}, nil
}

func (s *WebAuthnService) VerifyLogin(ctx context.Context, sessionID string, credentialAssertionData *protocol.ParsedCredentialAssertionData, ipAddress, userAgent string) (model.User, string, error) {
	tx := s.db.Begin()
	defer func() {
		tx.Rollback()
	}()

	// Load & delete the session row
	var storedSession model.WebauthnSession
	err := tx.
		WithContext(ctx).
		Clauses(clause.Returning{}).
		Delete(&storedSession, "id = ?", sessionID).
		Error
	if err != nil {
		return model.User{}, "", fmt.Errorf("failed to load WebAuthn session: %w", err)
	}

	session := webauthn.SessionData{
		Challenge: storedSession.Challenge,
		Expires:   storedSession.ExpiresAt.ToTime(),
	}

	var user *model.User
	_, err = s.webAuthn.ValidateDiscoverableLogin(func(_, userHandle []byte) (webauthn.User, error) {
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

	if err != nil {
		return model.User{}, "", err
	}

	if user.Disabled {
		return model.User{}, "", &common.UserDisabledError{}
	}

	token, err := s.jwtService.GenerateAccessToken(*user)
	if err != nil {
		return model.User{}, "", err
	}

	s.auditLogService.CreateNewSignInWithEmail(ctx, ipAddress, userAgent, user.ID, tx)

	err = tx.Commit().Error
	if err != nil {
		return model.User{}, "", err
	}

	return *user, token, nil
}

func (s *WebAuthnService) ListCredentials(ctx context.Context, userID string) ([]model.WebauthnCredential, error) {
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

func (s *WebAuthnService) DeleteCredential(ctx context.Context, userID string, credentialID string, ipAddress string, userAgent string) error {
	tx := s.db.Begin()
	defer func() {
		tx.Rollback()
	}()

	credential := &model.WebauthnCredential{}
	err := tx.
		WithContext(ctx).
		Clauses(clause.Returning{}).
		Delete(credential, "id = ? AND user_id = ?", credentialID, userID).
		Error
	if err != nil {
		return fmt.Errorf("failed to delete record: %w", err)
	}

	auditLogData := model.AuditLogData{"credentialID": hex.EncodeToString(credential.CredentialID), "passkeyName": credential.Name}
	s.auditLogService.Create(ctx, model.AuditLogEventPasskeyRemoved, ipAddress, userAgent, userID, auditLogData, tx)

	err = tx.Commit().Error
	if err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

func (s *WebAuthnService) UpdateCredential(ctx context.Context, userID, credentialID, name string) (model.WebauthnCredential, error) {
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
func (s *WebAuthnService) updateWebAuthnConfig() {
	s.webAuthn.Config.RPDisplayName = s.appConfigService.GetDbConfig().AppName.Value
}

func (s *WebAuthnService) CreateReauthenticationTokenWithAccessToken(ctx context.Context, accessToken string) (string, error) {
	tx := s.db.Begin()
	defer func() {
		tx.Rollback()
	}()

	token, err := s.jwtService.VerifyAccessToken(accessToken)
	if err != nil {
		return "", fmt.Errorf("invalid access token: %w", err)
	}

	userID, ok := token.Subject()
	if !ok {
		return "", errors.New("access token does not contain user ID")
	}

	// Check if token is issued less than a minute ago
	tokenExpiration, ok := token.IssuedAt()
	if !ok || time.Since(tokenExpiration) > time.Minute {
		return "", &common.ReauthenticationRequiredError{}
	}

	var user model.User
	err = tx.
		WithContext(ctx).
		First(&user, "id = ?", userID).
		Error
	if err != nil {
		return "", fmt.Errorf("failed to load user: %w", err)
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

func (s *WebAuthnService) CreateReauthenticationTokenWithWebauthn(ctx context.Context, sessionID string, credentialAssertionData *protocol.ParsedCredentialAssertionData) (string, error) {
	tx := s.db.Begin()
	defer func() {
		tx.Rollback()
	}()

	// Retrieve and delete the session
	var storedSession model.WebauthnSession
	err := tx.
		WithContext(ctx).
		Clauses(clause.Returning{}).
		Delete(&storedSession, "id = ? AND expires_at > ?", sessionID, datatype.DateTime(time.Now())).
		Error
	if err != nil {
		return "", fmt.Errorf("failed to load WebAuthn session: %w", err)
	}

	session := webauthn.SessionData{
		Challenge: storedSession.Challenge,
		Expires:   storedSession.ExpiresAt.ToTime(),
	}

	// Validate the credential assertion
	var user *model.User
	_, err = s.webAuthn.ValidateDiscoverableLogin(func(_, userHandle []byte) (webauthn.User, error) {
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

func (s *WebAuthnService) ConsumeReauthenticationToken(ctx context.Context, tx *gorm.DB, token string, userID string) error {
	hashedToken := utils.CreateSha256Hash(token)
	result := tx.WithContext(ctx).
		Clauses(clause.Returning{}).
		Delete(&model.ReauthenticationToken{}, "token = ? AND user_id = ? AND expires_at > ?", hashedToken, userID, datatype.DateTime(time.Now()))

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return &common.ReauthenticationRequiredError{}
	}
	return nil
}

func (s *WebAuthnService) createReauthenticationToken(ctx context.Context, tx *gorm.DB, userID string) (string, error) {
	token, err := utils.GenerateRandomAlphanumericString(32)
	if err != nil {
		return "", err
	}

	reauthToken := model.ReauthenticationToken{
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
