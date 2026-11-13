package webauthn

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"net/url"
	"slices"
	"time"

	"github.com/go-webauthn/webauthn/metadata"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/protocol/webauthncose"
	gowebauthn "github.com/go-webauthn/webauthn/webauthn"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/pocket-id/pocket-id/backend/internal/appconfig"
	"github.com/pocket-id/pocket-id/backend/internal/apperror"
	"github.com/pocket-id/pocket-id/backend/internal/auditlogs"
	"github.com/pocket-id/pocket-id/backend/internal/authz"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
)

// authenticationMethodPhishingResistant identifies phishing-resistant authentication, such as passkeys
// It must match the value emitted by the JWT service in the access token's "amr" claim
const authenticationMethodPhishingResistant = "phr"

const defaultRPDisplayName = "Pocket ID"

// maxRSAKeyBits bounds passkey RSA keys, because go-webauthn sets no upper limit and verifying with a huge modulus takes minutes
const maxRSAKeyBits = 8192

// go-webauthn exposes the missing user-verification reason only through DevInfo
const missingUserVerificationErrorInfo = "User verification required but flag not set by authenticator"

type Service struct {
	db           *gorm.DB
	webAuthn     *gowebauthn.WebAuthn
	sessions     SessionStore
	auditLog     AuditLogger
	fidoMetadata *fidoMetadata
}

func newService(deps Dependencies) (*Service, error) {
	appURL, err := url.Parse(deps.AppURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse app URL: %w", err)
	}

	wa, err := gowebauthn.New(&gowebauthn.Config{
		// Set a default value, it will be set again later
		RPDisplayName: defaultRPDisplayName,
		RPID:          appURL.Hostname(),
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
		db:       deps.DB,
		webAuthn: wa,
		sessions: deps.Sessions,
		auditLog: deps.AuditLog,
		fidoMetadata: &fidoMetadata{
			httpClient: deps.HTTPClient,
			url:        deps.FidoMetadataURL,
			path:       fidoMetadataPath,
		},
	}, nil
}

func (s *Service) BeginRegistration(ctx context.Context, dbConfig *appconfig.AppConfigModel, userID string) (*PublicKeyCredentialCreationOptions, error) {
	s.updateWebAuthnConfig(dbConfig)

	tx := s.db.Begin()
	defer func() {
		tx.Rollback()
	}()

	user, err := loadPasskeyUser(ctx, tx, userID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, apperror.UserNotFound()
	}
	if err != nil {
		return nil, fmt.Errorf("failed to load user: %w", err)
	}

	// Ask for an attestation only when it will be verified, because browsers prompt before sending one
	// The FIDO metadata loads while the user completes the ceremony, so a download doesn't delay finishing the registration
	var conveyance protocol.ConveyancePreference
	if _, required := requiredAttestation(dbConfig); required {
		conveyance = protocol.PreferDirectAttestation
		go func() {
			_, _ = s.fidoMetadata.get()
		}()
	}

	options, session, err := s.webAuthn.BeginRegistration(
		user,
		gowebauthn.WithAuthenticatorSelection(protocol.AuthenticatorSelection{
			AuthenticatorAttachment: authenticatorAttachment(dbConfig),
			UserVerification:        userVerificationRequirement(dbConfig),
		}),
		gowebauthn.WithResidentKeyRequirement(protocol.ResidentKeyRequirementRequired),
		gowebauthn.WithCredentialParameters(gowebauthn.CredentialParametersPQCRecommendedL3()),
		gowebauthn.WithExclusions(gowebauthn.Credentials(user.WebAuthnCredentials()).CredentialDescriptors()),
		gowebauthn.WithExtensions(gowebauthn.WithExtensionCredProps()), // Required for Firefox Android to properly save the key in Google password manager
		gowebauthn.WithConveyancePreference(conveyance),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to begin WebAuthn registration: %w", err)
	}

	sessionToStore := &WebauthnSession{
		ExpiresAt:        datatype.DateTime(session.Expires),
		Challenge:        session.Challenge,
		CredentialParams: session.CredParams,
		UserVerification: string(session.UserVerification),
		Extensions:       session.Extensions,
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

func (s *Service) VerifyRegistration(ctx context.Context, dbConfig *appconfig.AppConfigModel, sessionID string, userID string, credentialCreationData *protocol.ParsedCredentialCreationData, ipAddress, userAgent string) (WebauthnCredential, error) {
	// Load the FIDO metadata before the transaction starts, because the first load downloads it
	minCertification, attestationRequired := requiredAttestation(dbConfig)
	var metadataProvider metadata.Provider
	if attestationRequired {
		var err error
		metadataProvider, err = s.fidoMetadata.get()
		if err != nil {
			return WebauthnCredential{}, apperror.AttestationUnavailable(err)
		}
	}

	tx := s.db.Begin()
	defer func() {
		tx.Rollback()
	}()

	// Load & delete the session row
	var storedSession WebauthnSession
	result := tx.
		WithContext(ctx).
		Clauses(clause.Returning{}).
		Delete(&storedSession, "id = ?", sessionID)
	if result.Error != nil {
		return WebauthnCredential{}, fmt.Errorf("failed to load WebAuthn session: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return WebauthnCredential{}, apperror.InvalidWebAuthnSession()
	}

	session := gowebauthn.SessionData{
		Challenge:        storedSession.Challenge,
		Expires:          storedSession.ExpiresAt.ToTime(),
		CredParams:       storedSession.CredentialParams,
		UserVerification: protocol.UserVerificationRequirement(storedSession.UserVerification),
		Extensions:       storedSession.Extensions,
		UserID:           []byte(userID),
	}

	var user passkeyUser
	err := tx.
		WithContext(ctx).
		First(&user.User, "id = ?", userID).
		Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return WebauthnCredential{}, apperror.UserNotFound()
	}
	if err != nil {
		return WebauthnCredential{}, fmt.Errorf("failed to load user: %w", err)
	}

	// Self attestation is verified with the new key, so its size has to be checked before the response is verified
	if hasOversizedPublicKey(credentialCreationData.Response.AttestationObject.AuthData.AttData.CredentialPublicKey) {
		return WebauthnCredential{}, apperror.InvalidWebAuthnResponse(fmt.Errorf("RSA public key is larger than %d bits", maxRSAKeyBits))
	}

	credential, err := s.webAuthn.CreateCredential(&user, session, credentialCreationData)
	if err != nil {
		return WebauthnCredential{}, classifyPasskeyError(err, apperror.InvalidWebAuthnResponse)
	}
	if err := validateCredentialPolicy(dbConfig, credential); err != nil {
		return WebauthnCredential{}, err
	}
	if attestationRequired {
		err = s.verifyAttestation(ctx, metadataProvider, credential, minCertification)
		if err != nil {
			return WebauthnCredential{}, err
		}
	}

	// Determine passkey name using AAGUID and User-Agent
	passkeyName := s.determinePasskeyName(credential.Authenticator.AAGUID)
	aaguid := formatAAGUID(credential.Authenticator.AAGUID)

	credentialToStore := WebauthnCredential{
		Name:            passkeyName,
		CredentialID:    credential.ID,
		AttestationType: credential.AttestationType,
		PublicKey:       credential.PublicKey,
		Transport:       credential.Transport,
		UserID:          user.ID,
		BackupEligible:  credential.Flags.BackupEligible,
		BackupState:     credential.Flags.BackupState,
		AAGUID:          aaguid,
		// The attestation was verified above whenever it is required
		AttestationVerified: attestationRequired,
	}
	err = tx.
		WithContext(ctx).
		Create(&credentialToStore).
		Error
	if err != nil {
		return WebauthnCredential{}, fmt.Errorf("failed to store WebAuthn credential: %w", err)
	}

	auditLogData := auditlogs.Data{"credentialID": hex.EncodeToString(credential.ID), "passkeyName": passkeyName}
	s.auditLog.Create(ctx, auditlogs.EventPasskeyAdded, ipAddress, userAgent, userID, auditLogData, tx)

	err = tx.Commit().Error
	if err != nil {
		return WebauthnCredential{}, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return credentialToStore, nil
}

func (s *Service) determinePasskeyName(aaguid []byte) string {
	// First try to identify by AAGUID using a combination of builtin + MDS
	authenticatorName := getAuthenticatorName(formatAAGUID(aaguid))
	if authenticatorName != "" {
		return authenticatorName + " Passkey"
	}

	return "New Passkey" // Default fallback
}

func (s *Service) BeginLogin(ctx context.Context, dbConfig *appconfig.AppConfigModel) (*PublicKeyCredentialRequestOptions, error) {
	options, session, err := s.webAuthn.BeginDiscoverableLogin(
		gowebauthn.WithUserVerification(userVerificationRequirement(dbConfig)),
	)
	if err != nil {
		return nil, err
	}

	sessionToStore := &WebauthnSession{
		ExpiresAt:        datatype.DateTime(session.Expires),
		Challenge:        session.Challenge,
		UserVerification: string(session.UserVerification),
		Extensions:       session.Extensions,
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

func (s *Service) VerifyLogin(ctx context.Context, dbConfig *appconfig.AppConfigModel, sessionID string, credentialAssertionData *protocol.ParsedCredentialAssertionData, ipAddress, userAgent, browserToken string, remember bool) (model.User, model.LoginTokens, error) {
	tx := s.db.Begin()
	defer func() {
		tx.Rollback()
	}()

	// Load & delete the session row
	var storedSession WebauthnSession
	result := tx.
		WithContext(ctx).
		Clauses(clause.Returning{}).
		Delete(&storedSession, "id = ?", sessionID)
	if result.Error != nil {
		return model.User{}, model.LoginTokens{}, fmt.Errorf("failed to load WebAuthn session: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return model.User{}, model.LoginTokens{}, apperror.InvalidWebAuthnSession()
	}

	session := gowebauthn.SessionData{
		Challenge:        storedSession.Challenge,
		Expires:          storedSession.ExpiresAt.ToTime(),
		UserVerification: protocol.UserVerificationRequirement(storedSession.UserVerification),
		Extensions:       storedSession.Extensions,
		CredParams:       storedSession.CredentialParams,
	}
	discardUnrequestedFalseAppIDOutput(session.Extensions, credentialAssertionData)

	var user *passkeyUser
	_, err := s.webAuthn.ValidateDiscoverableLogin(func(_, userHandle []byte) (gowebauthn.User, error) {
		var innerErr error
		user, innerErr = loadPasskeyUser(ctx, tx, string(userHandle))
		// Preserve infrastructure failures through go-webauthn's wrapped callback error
		if innerErr != nil {
			if !errors.Is(innerErr, gorm.ErrRecordNotFound) {
				return nil, apperror.Internal(innerErr)
			}
			return nil, innerErr
		}
		discardOversizedCredentials(user)
		return user, nil
	}, session, credentialAssertionData)

	if err != nil {
		return model.User{}, model.LoginTokens{}, classifyPasskeyError(err, apperror.WebAuthnAuthenticationFailed)
	}
	if user == nil {
		return model.User{}, model.LoginTokens{}, apperror.WebAuthnAuthenticationFailed(errors.New("WebAuthn response did not resolve to a user"))
	}

	if user.Disabled {
		return model.User{}, model.LoginTokens{}, apperror.UserDisabled()
	}

	// Prepare browser recognition and the notification within the login transaction
	signIn := s.auditLog.CreateSignIn(ctx, auditlogs.EventSignIn, ipAddress, userAgent, user.ID, browserToken, tx, dbConfig.EmailLoginNotificationMode)
	if !signIn.Created {
		return model.User{}, model.LoginTokens{}, errors.New("failed to create sign-in audit log")
	}

	err = tx.Commit().Error
	if err != nil {
		return model.User{}, model.LoginTokens{}, err
	}

	// Start the session only after the login transaction has committed, since it invokes an actor
	token, err := s.sessions.Create(ctx, user.ID, authenticationMethodPhishingResistant, ipAddress, userAgent, dbConfig.SessionDuration.AsDurationMinutes(), remember)
	if err != nil {
		return model.User{}, model.LoginTokens{}, err
	}

	// Deliver the notification only after the login transaction has committed
	s.auditLog.SendSignInNotification(ctx, signIn)
	return user.User, model.LoginTokens{SessionToken: token, KnownBrowserToken: signIn.KnownBrowserToken}, nil
}

func (s *Service) ListCredentials(ctx context.Context, userID string) ([]WebauthnCredential, error) {
	var credentials []WebauthnCredential
	err := s.db.
		WithContext(ctx).
		Find(&credentials, "user_id = ?", userID).
		Error
	if err != nil {
		return nil, err
	}
	return credentials, nil
}

// DeleteCredential removes a passkey of userID on behalf of origin, which is named in the audit log when it isn't the passkey's owner
func (s *Service) DeleteCredential(ctx context.Context, userID string, credentialID string, origin auditlogs.Origin) error {
	tx := s.db.Begin()
	defer func() {
		tx.Rollback()
	}()

	credential := &WebauthnCredential{}
	result := tx.
		WithContext(ctx).
		Clauses(clause.Returning{}).
		Delete(credential, "id = ? AND user_id = ?", credentialID, userID)
	if result.Error != nil {
		return fmt.Errorf("failed to delete record: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return apperror.NotFound("Passkey")
	}

	s.auditLog.Record(ctx, auditlogs.EventPasskeyRemoved, origin, userID, auditlogs.Data{"credentialID": hex.EncodeToString(credential.CredentialID), "passkeyName": credential.Name}, tx)

	err := tx.Commit().Error
	if err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

func (s *Service) UpdateCredential(ctx context.Context, userID, credentialID, name string) (WebauthnCredential, error) {
	tx := s.db.Begin()
	defer func() {
		tx.Rollback()
	}()

	var credential WebauthnCredential
	err := tx.
		WithContext(ctx).
		Where("id = ? AND user_id = ?", credentialID, userID).
		First(&credential).
		Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return WebauthnCredential{}, apperror.NotFound("Passkey")
	}
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
func (s *Service) updateWebAuthnConfig(dbConfig *appconfig.AppConfigModel) {
	s.webAuthn.Config.RPDisplayName = dbConfig.AppName.String()
}

func userVerificationRequirement(dbConfig *appconfig.AppConfigModel) protocol.UserVerificationRequirement {
	if dbConfig.WebauthnUserVerification == "preferred" {
		return protocol.VerificationPreferred
	}

	return protocol.VerificationRequired
}

func authenticatorAttachment(dbConfig *appconfig.AppConfigModel) protocol.AuthenticatorAttachment {
	switch dbConfig.WebauthnAuthenticatorAttachment {
	case "platform":
		return protocol.Platform
	case "cross-platform":
		return protocol.CrossPlatform
	default:
		return ""
	}
}

func allowsSyncedPasskeys(dbConfig *appconfig.AppConfigModel) bool {
	return dbConfig.WebauthnAllowSyncedPasskeys.IsTrue()
}

// minCertifications maps the attestation setting to the minimum FIDO certification, where an empty status accepts any verified authenticator
var minCertifications = map[string]metadata.AuthenticatorStatus{
	"verified": "",
	"l1":       metadata.FidoCertifiedL1,
	"l2":       metadata.FidoCertifiedL2,
	"l3":       metadata.FidoCertifiedL3,
}

// certificationRanks orders the FIDO certification levels, where the phased-out FIDO_CERTIFIED counts as level 1
var certificationRanks = map[metadata.AuthenticatorStatus]int{
	metadata.FidoCertified:       1,
	metadata.FidoCertifiedL1:     1,
	metadata.FidoCertifiedL1plus: 2,
	metadata.FidoCertifiedL2:     3,
	metadata.FidoCertifiedL2plus: 4,
	metadata.FidoCertifiedL3:     5,
	metadata.FidoCertifiedL3plus: 6,
}

// trustedAttestationTypes are the attestation types whose certificate proves the authenticator model, unlike self attestation which any key can produce
var trustedAttestationTypes = []string{
	string(metadata.BasicFull),
	string(metadata.AttCA),
	string(metadata.AnonCA),
}

func requiredAttestation(dbConfig *appconfig.AppConfigModel) (minCertification metadata.AuthenticatorStatus, required bool) {
	minCertification, required = minCertifications[dbConfig.WebauthnAttestation.String()]
	return minCertification, required
}

// verifyAttestation proves the authenticator's make and model through the FIDO Metadata Service and checks its certification level
func (s *Service) verifyAttestation(ctx context.Context, provider metadata.Provider, credential *gowebauthn.Credential, minCertification metadata.AuthenticatorStatus) error {
	// Synced passkeys send no attestation at all, so they end up here too
	if !slices.Contains(trustedAttestationTypes, credential.AttestationType) {
		return apperror.PasskeyAttestationRequired()
	}

	// Verify the attestation certificate against the roots FIDO publishes for this AAGUID, which also rejects models with a revoked or compromised status
	err := credential.Verify(provider, s.webAuthn.Config.Attestation, s.webAuthn.Config.Signature)
	if err != nil {
		// go-webauthn leaves the message empty for some metadata failures, such as an untrusted chain, and keeps the reason in DevInfo
		var protocolError *protocol.Error
		if errors.As(err, &protocolError) {
			err = fmt.Errorf("%w: %s", err, protocolError.DevInfo)
		}
		return apperror.AuthenticatorNotCertified(err)
	}
	if minCertification == "" {
		return nil
	}

	// Compare the highest certification in effect today with the required one, because FIDO lists earlier levels and announces upcoming ones in the same history
	aaguid, err := uuid.FromBytes(credential.Authenticator.AAGUID)
	if err != nil {
		return apperror.AuthenticatorNotCertified(err)
	}
	entry, err := provider.GetEntry(ctx, aaguid)
	if err != nil || entry == nil {
		return apperror.AuthenticatorNotCertified(err)
	}
	for _, report := range metadata.EffectiveStatusReports(entry.StatusReports, time.Now()) {
		if certificationRanks[report.Status] >= certificationRanks[minCertification] {
			return nil
		}
	}
	return apperror.AuthenticatorNotCertified(nil)
}

func validateCredentialPolicy(dbConfig *appconfig.AppConfigModel, credential *gowebauthn.Credential) error {
	if !allowsSyncedPasskeys(dbConfig) && credential.Flags.BackupEligible {
		return apperror.SyncedPasskeyNotAllowed()
	}

	// Browsers treat the requested attachment as a filter they may ignore, e.g. Windows still offers Windows Hello when only security keys are requested
	// The reported attachment isn't signed by the authenticator, so this enforces the admin's policy rather than proving the authenticator type
	if attachment := authenticatorAttachment(dbConfig); attachment != "" && credential.Authenticator.Attachment != attachment {
		return apperror.AuthenticatorAttachmentNotAllowed()
	}

	return nil
}

// CreateReauthenticationTokenWithSession lets a passkey sign-in that just happened count as reauthentication, so the user isn't asked for their passkey twice in a row
func (s *Service) CreateReauthenticationTokenWithSession(ctx context.Context, principal authz.Principal) (string, error) {
	if principal.AuthenticationMethod != authenticationMethodPhishingResistant || time.Since(principal.AuthenticationTime) > time.Minute {
		return "", apperror.ReauthenticationRequired()
	}

	tx := s.db.Begin()
	defer func() {
		tx.Rollback()
	}()

	reauthToken, err := s.createReauthenticationToken(ctx, tx, principal.UserID)
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
	result := tx.
		WithContext(ctx).
		Clauses(clause.Returning{}).
		Delete(&storedSession, "id = ? AND expires_at > ?", sessionID, datatype.DateTime(time.Now()))
	if result.Error != nil {
		return "", fmt.Errorf("failed to load WebAuthn session: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return "", apperror.InvalidWebAuthnSession()
	}

	session := gowebauthn.SessionData{
		Challenge:        storedSession.Challenge,
		Expires:          storedSession.ExpiresAt.ToTime(),
		UserVerification: protocol.UserVerificationRequirement(storedSession.UserVerification),
		Extensions:       storedSession.Extensions,
		CredParams:       storedSession.CredentialParams,
	}
	discardUnrequestedFalseAppIDOutput(session.Extensions, credentialAssertionData)

	// Validate the credential assertion
	var user *passkeyUser
	_, err := s.webAuthn.ValidateDiscoverableLogin(func(_, userHandle []byte) (gowebauthn.User, error) {
		var innerErr error
		user, innerErr = loadPasskeyUser(ctx, tx, string(userHandle))
		// Preserve infrastructure failures through go-webauthn's wrapped callback error
		if innerErr != nil {
			if !errors.Is(innerErr, gorm.ErrRecordNotFound) {
				return nil, apperror.Internal(innerErr)
			}
			return nil, innerErr
		}
		discardOversizedCredentials(user)
		return user, nil
	}, session, credentialAssertionData)

	if err != nil {
		return "", classifyPasskeyError(err, apperror.WebAuthnAuthenticationFailed)
	}
	if user == nil {
		return "", apperror.WebAuthnAuthenticationFailed(errors.New("WebAuthn response did not resolve to a user"))
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

func hasOversizedPublicKey(publicKey []byte) bool {
	key, err := webauthncose.ParsePublicKey(publicKey)
	if err != nil {
		// go-webauthn rejects a key it can't parse before verifying anything with it
		return false
	}

	rsaKey, ok := key.(webauthncose.RSAPublicKeyData)
	return ok && new(big.Int).SetBytes(rsaKey.Modulus).BitLen() > maxRSAKeyBits
}

// loadPasskeyUser loads a user with their passkeys and returns gorm.ErrRecordNotFound if the user doesn't exist
func loadPasskeyUser(ctx context.Context, tx *gorm.DB, userID string) (*passkeyUser, error) {
	user := &passkeyUser{}
	err := tx.WithContext(ctx).First(&user.User, "id = ?", userID).Error
	if err != nil {
		return nil, err
	}

	err = tx.WithContext(ctx).Find(&user.credentials, "user_id = ?", userID).Error
	if err != nil {
		return nil, fmt.Errorf("failed to load passkeys: %w", err)
	}
	return user, nil
}

// discardOversizedCredentials drops passkeys registered before RSA keys were bounded, so a sign-in can't be pointed at one of them
func discardOversizedCredentials(user *passkeyUser) {
	user.credentials = slices.DeleteFunc(user.credentials, func(credential WebauthnCredential) bool {
		return hasOversizedPublicKey(credential.PublicKey)
	})
}

func classifyPasskeyError(err error, fallback func(error) *apperror.Error) *apperror.Error {
	if appErr, ok := errors.AsType[*apperror.Error](err); ok {
		return appErr
	}

	if protocolError, ok := errors.AsType[*protocol.Error](err); ok &&
		protocolError.Type == protocol.ErrVerification.Type &&
		protocolError.DevInfo == missingUserVerificationErrorInfo {
		return apperror.PasskeyUserVerificationRequired(err)
	}

	return fallback(err)
}

func discardUnrequestedFalseAppIDOutput(session protocol.SessionExtensions, credential *protocol.ParsedCredentialAssertionData) {
	if credential == nil || credential.ClientExtensionResults.AppID == nil || *credential.ClientExtensionResults.AppID {
		return
	}

	for _, requested := range session.Requested {
		if requested == protocol.ExtensionAppID {
			return
		}
	}

	// Safari reports appid=false for security keys even when the relying party did not request the legacy extension
	credential.ClientExtensionResults.AppID = nil
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
		return time.Time{}, apperror.ReauthenticationRequired()
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
