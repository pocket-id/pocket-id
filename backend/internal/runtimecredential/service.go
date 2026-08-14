package runtimecredential

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/pocket-id/pocket-id/backend/internal/appconfig"
	"github.com/pocket-id/pocket-id/backend/internal/apperror"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
)

const (
	authenticationMethodProofOfPossession = "pop"
	challengeLifetime                     = time.Minute
	challengeRandomLength                 = 32
	operationRegister                     = "register"
	operationLogin                        = "login"
	operationReauthenticate               = "reauthenticate"
)

type Service struct {
	db        *gorm.DB
	signer    TokenService
	auditLog  AuditLogger
	bootstrap BootstrapTokenConsumer
	reauth    ReauthenticationTokenIssuer
}

func newService(deps Dependencies) *Service {
	return &Service{db: deps.DB, signer: deps.Signer, auditLog: deps.AuditLog, bootstrap: deps.Bootstrap, reauth: deps.Reauth}
}

// BeginRegistration implements FCA06 by binding one-time bootstrap authority to a locally generated public key before durable registration
func (s *Service) BeginRegistration(ctx context.Context, input registrationStartDto, deviceToken string) (challengeDto, error) {
	token := utils.NormalizeUnambiguousString(input.Token)
	state, err := s.bootstrap.ConsumeToken(ctx, token, deviceToken)
	if err != nil {
		return challengeDto{}, err
	}

	challenge, err := s.beginRegistrationAfterConsume(ctx, state.UserID, input)
	if err != nil {
		s.bootstrap.RestoreToken(ctx, token, state)
		return challengeDto{}, err
	}
	return challenge, nil
}

func (s *Service) beginRegistrationAfterConsume(ctx context.Context, userID string, input registrationStartDto) (challengeDto, error) {
	publicKey, err := decodePublicKey(input.Algorithm, input.PublicKey)
	if err != nil {
		return challengeDto{}, err
	}

	tx := s.db.Begin()
	defer tx.Rollback()

	user, err := loadRuntimePathUser(ctx, tx, userID)
	if err != nil {
		return challengeDto{}, err
	}
	if err := ensureRegistrationAvailable(ctx, tx, user); err != nil {
		return challengeDto{}, err
	}

	name := strings.TrimSpace(input.Name)
	if name == "" {
		return challengeDto{}, apperror.InvalidField("name", "required", "is required")
	}
	algorithm := model.RuntimeCredentialAlgorithmEd25519
	stored, response, err := newChallenge(operationRegister, user.ID)
	if err != nil {
		return challengeDto{}, err
	}
	stored.CredentialName = &name
	stored.Algorithm = &algorithm
	stored.PublicKey = publicKey

	if err := tx.WithContext(ctx).Create(&stored).Error; err != nil {
		return challengeDto{}, fmt.Errorf("failed to save runtime registration challenge: %w", err)
	}
	if err := tx.Commit().Error; err != nil {
		return challengeDto{}, fmt.Errorf("failed to commit runtime registration challenge: %w", err)
	}
	return response, nil
}

func (s *Service) FinishRegistration(ctx context.Context, cfg *appconfig.AppConfigModel, input proofFinishDto, ipAddress, userAgent string) (model.User, model.RuntimeCredential, string, error) {
	tx := s.db.Begin()
	defer tx.Rollback()

	challenge, err := consumeChallenge(ctx, tx, input.SessionID, operationRegister)
	if err != nil {
		return model.User{}, model.RuntimeCredential{}, "", err
	}
	user, err := loadRuntimePathUser(ctx, tx, challenge.UserID)
	if err != nil {
		return model.User{}, model.RuntimeCredential{}, "", err
	}
	if err := ensureRegistrationAvailable(ctx, tx, user); err != nil {
		return model.User{}, model.RuntimeCredential{}, "", err
	}
	if challenge.CredentialName == nil || challenge.Algorithm == nil {
		return model.User{}, model.RuntimeCredential{}, "", apperror.RuntimeCredentialInvalid()
	}
	if err := verifyProof(challenge.PublicKey, challenge.Challenge, input.Signature); err != nil {
		return model.User{}, model.RuntimeCredential{}, "", err
	}

	credential := model.RuntimeCredential{
		Name:      *challenge.CredentialName,
		Algorithm: *challenge.Algorithm,
		PublicKey: challenge.PublicKey,
		UserID:    user.ID,
	}
	if err := tx.WithContext(ctx).Create(&credential).Error; err != nil {
		return model.User{}, model.RuntimeCredential{}, "", fmt.Errorf("failed to store runtime credential: %w", err)
	}

	s.auditLog.Create(ctx, model.AuditLogEventRuntimeCredentialRegistered, ipAddress, userAgent, user.ID, credentialAuditData(credential), tx)
	s.auditLog.CreateNewSignInWithEmail(ctx, ipAddress, userAgent, user.ID, tx, cfg.EmailLoginNotificationEnabled.IsTrue())
	accessToken, err := s.signer.GenerateAccessToken(user, authenticationMethodProofOfPossession, cfg.SessionDuration.AsDurationMinutes())
	if err != nil {
		return model.User{}, model.RuntimeCredential{}, "", err
	}
	if err := tx.Commit().Error; err != nil {
		return model.User{}, model.RuntimeCredential{}, "", err
	}
	return user, credential, accessToken, nil
}

// BeginLogin implements FCA07 by authenticating an existing username and credential through proof of possession before normal session issuance
func (s *Service) BeginLogin(ctx context.Context, input loginStartDto) (challengeDto, error) {
	return s.beginCredentialProof(ctx, input.Username, input.CredentialID, "", operationLogin)
}

// BeginReauthentication implements FCA08 by binding fresh runtime proof to the already authenticated user before issuing the ordinary reauthentication token
func (s *Service) BeginReauthentication(ctx context.Context, userID, credentialID string) (challengeDto, error) {
	return s.beginCredentialProof(ctx, "", credentialID, userID, operationReauthenticate)
}

func (s *Service) beginCredentialProof(ctx context.Context, username, credentialID, userID, operation string) (challengeDto, error) {
	tx := s.db.Begin()
	defer tx.Rollback()

	query := tx.WithContext(ctx).
		Joins("User").
		Where("runtime_credentials.id = ? AND runtime_credentials.revoked_at IS NULL", credentialID)
	if userID != "" {
		query = query.Where("runtime_credentials.user_id = ?", userID)
	} else {
		query = query.Where(`"User".username = ?`, username)
	}

	var credential model.RuntimeCredential
	if err := query.First(&credential).Error; err != nil {
		return challengeDto{}, apperror.RuntimeCredentialInvalid()
	}
	if credential.User.Disabled || !credential.User.IsAgent || credentialExpired(credential, time.Now()) {
		return challengeDto{}, apperror.RuntimeCredentialInvalid()
	}

	stored, response, err := newChallenge(operation, credential.UserID)
	if err != nil {
		return challengeDto{}, err
	}
	stored.RuntimeCredentialID = &credential.ID
	if err := tx.WithContext(ctx).Create(&stored).Error; err != nil {
		return challengeDto{}, fmt.Errorf("failed to save runtime proof challenge: %w", err)
	}
	if err := tx.Commit().Error; err != nil {
		return challengeDto{}, err
	}
	return response, nil
}

func (s *Service) FinishLogin(ctx context.Context, cfg *appconfig.AppConfigModel, input proofFinishDto, ipAddress, userAgent string) (model.User, string, error) {
	user, _, tx, err := s.finishCredentialProof(ctx, input, operationLogin, ipAddress, userAgent)
	if err != nil {
		return model.User{}, "", err
	}
	defer tx.Rollback()

	s.auditLog.CreateNewSignInWithEmail(ctx, ipAddress, userAgent, user.ID, tx, cfg.EmailLoginNotificationEnabled.IsTrue())
	accessToken, err := s.signer.GenerateAccessToken(user, authenticationMethodProofOfPossession, cfg.SessionDuration.AsDurationMinutes())
	if err != nil {
		return model.User{}, "", err
	}
	if err := tx.Commit().Error; err != nil {
		return model.User{}, "", err
	}
	return user, accessToken, nil
}

func (s *Service) FinishReauthentication(ctx context.Context, userID string, input proofFinishDto, ipAddress, userAgent string) (string, error) {
	user, _, tx, err := s.finishCredentialProof(ctx, input, operationReauthenticate, ipAddress, userAgent)
	if err != nil {
		return "", err
	}
	defer tx.Rollback()
	if user.ID != userID {
		return "", apperror.RuntimeCredentialInvalid()
	}
	token, err := s.reauth.CreateReauthenticationToken(ctx, tx, user.ID)
	if err != nil {
		return "", err
	}
	if err := tx.Commit().Error; err != nil {
		return "", err
	}
	return token, nil
}

func (s *Service) finishCredentialProof(ctx context.Context, input proofFinishDto, operation, ipAddress, userAgent string) (model.User, model.RuntimeCredential, *gorm.DB, error) {
	tx := s.db.Begin()
	challenge, err := consumeChallenge(ctx, tx, input.SessionID, operation)
	if err != nil {
		tx.Rollback()
		return model.User{}, model.RuntimeCredential{}, nil, err
	}
	if challenge.RuntimeCredentialID == nil {
		tx.Rollback()
		return model.User{}, model.RuntimeCredential{}, nil, apperror.RuntimeCredentialInvalid()
	}

	var credential model.RuntimeCredential
	err = tx.WithContext(ctx).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Preload("User").
		Where("id = ? AND user_id = ? AND revoked_at IS NULL", *challenge.RuntimeCredentialID, challenge.UserID).
		First(&credential).
		Error
	if err != nil || credential.User.Disabled || !credential.User.IsAgent || credentialExpired(credential, time.Now()) {
		tx.Rollback()
		return model.User{}, model.RuntimeCredential{}, nil, apperror.RuntimeCredentialInvalid()
	}
	if err := verifyProof(credential.PublicKey, challenge.Challenge, input.Signature); err != nil {
		tx.Rollback()
		return model.User{}, model.RuntimeCredential{}, nil, err
	}

	now := datatype.DateTime(time.Now())
	credential.LastUsedAt = &now
	if err := tx.WithContext(ctx).Model(&credential).Update("last_used_at", now).Error; err != nil {
		tx.Rollback()
		return model.User{}, model.RuntimeCredential{}, nil, err
	}
	s.auditLog.Create(ctx, model.AuditLogEventRuntimeCredentialAuthenticated, ipAddress, userAgent, credential.UserID, credentialAuditData(credential), tx)
	return credential.User, credential, tx, nil
}

// ListCredentials implements the FCA09 metadata boundary while rename and revocation remain scoped to the owning user or an administrator
func (s *Service) ListCredentials(ctx context.Context, userID string) ([]model.RuntimeCredential, error) {
	var userCount int64
	if err := s.db.WithContext(ctx).Model(&model.User{}).Where("id = ?", userID).Count(&userCount).Error; err != nil {
		return nil, err
	}
	if userCount == 0 {
		return nil, apperror.UserNotFound()
	}

	var credentials []model.RuntimeCredential
	err := s.db.WithContext(ctx).Where("user_id = ?", userID).Order("created_at DESC").Find(&credentials).Error
	return credentials, err
}

func (s *Service) UpdateCredential(ctx context.Context, userID, credentialID, name string) (model.RuntimeCredential, error) {
	name = strings.TrimSpace(name)
	if name == "" || len(name) > 50 {
		return model.RuntimeCredential{}, apperror.InvalidField("name", "invalid_length", "must contain between 1 and 50 characters")
	}

	var credential model.RuntimeCredential
	err := s.db.WithContext(ctx).Where("id = ? AND user_id = ?", credentialID, userID).First(&credential).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return model.RuntimeCredential{}, apperror.NotFound("Runtime credential")
	}
	if err != nil {
		return model.RuntimeCredential{}, err
	}
	credential.Name = name
	if err := s.db.WithContext(ctx).Save(&credential).Error; err != nil {
		return model.RuntimeCredential{}, err
	}
	return credential, nil
}

func (s *Service) RevokeCredential(ctx context.Context, userID, credentialID, ipAddress, userAgent, actorUserID string) error {
	tx := s.db.Begin()
	defer tx.Rollback()

	var credential model.RuntimeCredential
	err := tx.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND user_id = ?", credentialID, userID).First(&credential).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return apperror.NotFound("Runtime credential")
	}
	if err != nil {
		return err
	}
	if credential.RevokedAt != nil {
		return nil
	}

	now := datatype.DateTime(time.Now())
	credential.RevokedAt = &now
	if err := tx.WithContext(ctx).Model(&credential).Update("revoked_at", now).Error; err != nil {
		return err
	}
	data := credentialAuditData(credential)
	if actorUserID != "" && actorUserID != userID {
		var actor model.User
		if err := tx.WithContext(ctx).First(&actor, "id = ?", actorUserID).Error; err != nil {
			return err
		}
		data["actorUserID"] = actor.ID
		data["actorUsername"] = actor.Username
	}
	s.auditLog.Create(ctx, model.AuditLogEventRuntimeCredentialRevoked, ipAddress, userAgent, userID, data, tx)
	return tx.Commit().Error
}

func newChallenge(operation, userID string) (model.RuntimeCredentialChallenge, challengeDto, error) {
	randomBytes := make([]byte, challengeRandomLength)
	if _, err := rand.Read(randomBytes); err != nil {
		return model.RuntimeCredentialChallenge{}, challengeDto{}, err
	}
	message := append([]byte("pocket-id-runtime-credential/v1/"+operation+"\n"), randomBytes...)
	expiresAt := datatype.DateTime(time.Now().Add(challengeLifetime))
	stored := model.RuntimeCredentialChallenge{Operation: operation, Challenge: message, ExpiresAt: expiresAt, UserID: userID}
	stored.ID = uuid.NewString()
	stored.CreatedAt = datatype.DateTime(time.Now())
	response := challengeDto{SessionID: stored.ID, Challenge: base64.RawURLEncoding.EncodeToString(message), ExpiresAt: expiresAt}
	return stored, response, nil
}

func consumeChallenge(ctx context.Context, tx *gorm.DB, sessionID, operation string) (model.RuntimeCredentialChallenge, error) {
	var challenge model.RuntimeCredentialChallenge
	result := tx.WithContext(ctx).Clauses(clause.Returning{}).Delete(&challenge, "id = ? AND operation = ? AND expires_at > ?", sessionID, operation, datatype.DateTime(time.Now()))
	if result.Error != nil {
		return model.RuntimeCredentialChallenge{}, result.Error
	}
	if result.RowsAffected == 0 {
		return model.RuntimeCredentialChallenge{}, apperror.RuntimeCredentialInvalid()
	}
	return challenge, nil
}

func loadRuntimePathUser(ctx context.Context, tx *gorm.DB, userID string) (model.User, error) {
	var user model.User
	err := tx.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).First(&user, "id = ?", userID).Error
	if err != nil || user.Disabled || !user.IsAgent {
		return model.User{}, apperror.RuntimeCredentialInvalid()
	}
	return user, nil
}

func ensureRegistrationAvailable(ctx context.Context, tx *gorm.DB, user model.User) error {
	var passkeys int64
	if err := tx.WithContext(ctx).Model(&model.WebauthnCredential{}).Where("user_id = ?", user.ID).Count(&passkeys).Error; err != nil {
		return err
	}
	if passkeys > 0 {
		return apperror.AuthenticationPathMismatch()
	}
	var active int64
	if err := tx.WithContext(ctx).Model(&model.RuntimeCredential{}).Where("user_id = ? AND revoked_at IS NULL", user.ID).Count(&active).Error; err != nil {
		return err
	}
	if active > 0 {
		return apperror.RuntimeCredentialExists()
	}
	return nil
}

func decodePublicKey(algorithm, encoded string) ([]byte, error) {
	if algorithm != model.RuntimeCredentialAlgorithmEd25519 {
		return nil, apperror.InvalidField("algorithm", "unsupported", "must be Ed25519")
	}
	publicKey, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil || len(publicKey) != ed25519.PublicKeySize {
		return nil, apperror.InvalidField("publicKey", "invalid_format", "must be an unpadded base64url Ed25519 public key")
	}
	return publicKey, nil
}

func verifyProof(publicKey, challenge []byte, encodedSignature string) error {
	signature, err := base64.RawURLEncoding.DecodeString(encodedSignature)
	if err != nil || len(signature) != ed25519.SignatureSize || len(publicKey) != ed25519.PublicKeySize || !ed25519.Verify(ed25519.PublicKey(publicKey), challenge, signature) {
		return apperror.RuntimeCredentialInvalid()
	}
	return nil
}

func credentialExpired(credential model.RuntimeCredential, now time.Time) bool {
	return credential.ExpiresAt != nil && !credential.ExpiresAt.ToTime().After(now)
}

func credentialAuditData(credential model.RuntimeCredential) model.AuditLogData {
	return model.AuditLogData{"credentialID": credential.ID, "credentialName": credential.Name}
}
