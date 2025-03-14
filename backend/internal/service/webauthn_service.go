package service

import (
	"fmt"
	"net/http"
	"time"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
	"gorm.io/gorm"
)

type WebAuthnService struct {
	db               *gorm.DB
	webAuthn         *webauthn.WebAuthn
	jwtService       *JwtService
	auditLogService  *AuditLogService
	appConfigService *AppConfigService
	mdsService       *MdsService
}

func NewWebAuthnService(db *gorm.DB, jwtService *JwtService, auditLogService *AuditLogService, appConfigService *AppConfigService, mdsService *MdsService) *WebAuthnService {
	webauthnConfig := &webauthn.Config{
		RPDisplayName: appConfigService.DbConfig.AppName.Value,
		RPID:          utils.GetHostnameFromURL(common.EnvConfig.AppURL),
		RPOrigins:     []string{common.EnvConfig.AppURL},
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
	}
	wa, _ := webauthn.New(webauthnConfig)
	return &WebAuthnService{db: db, webAuthn: wa, jwtService: jwtService, auditLogService: auditLogService, appConfigService: appConfigService, mdsService: mdsService}
}

func (s *WebAuthnService) BeginRegistration(userID string) (*model.PublicKeyCredentialCreationOptions, error) {
	s.updateWebAuthnConfig()

	var user model.User
	if err := s.db.Preload("Credentials").Find(&user, "id = ?", userID).Error; err != nil {
		return nil, err
	}

	options, session, err := s.webAuthn.BeginRegistration(&user, webauthn.WithResidentKeyRequirement(protocol.ResidentKeyRequirementRequired), webauthn.WithExclusions(user.WebAuthnCredentialDescriptors()))
	if err != nil {
		return nil, err
	}

	sessionToStore := &model.WebauthnSession{
		ExpiresAt:        datatype.DateTime(session.Expires),
		Challenge:        session.Challenge,
		UserVerification: string(session.UserVerification),
	}

	if err := s.db.Create(&sessionToStore).Error; err != nil {
		return nil, err
	}

	return &model.PublicKeyCredentialCreationOptions{
		Response:  options.Response,
		SessionID: sessionToStore.ID,
		Timeout:   s.webAuthn.Config.Timeouts.Registration.Timeout,
	}, nil
}

func (s *WebAuthnService) VerifyRegistration(sessionID, userID string, r *http.Request) (model.WebauthnCredential, error) {
	var storedSession model.WebauthnSession
	if err := s.db.First(&storedSession, "id = ?", sessionID).Error; err != nil {
		return model.WebauthnCredential{}, err
	}

	session := webauthn.SessionData{
		Challenge: storedSession.Challenge,
		Expires:   storedSession.ExpiresAt.ToTime(),
		UserID:    []byte(userID),
	}

	var user model.User
	if err := s.db.Find(&user, "id = ?", userID).Error; err != nil {
		return model.WebauthnCredential{}, err
	}

	credential, err := s.webAuthn.FinishRegistration(&user, session, r)
	if err != nil {
		return model.WebauthnCredential{}, err
	}

	// Determine passkey name using AAGUID and User-Agent
	passkeyName := s.determinePasskeyName(credential.Authenticator.AAGUID, r.UserAgent())

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
	if err := s.db.Create(&credentialToStore).Error; err != nil {
		return model.WebauthnCredential{}, err
	}

	return credentialToStore, nil
}

func (s *WebAuthnService) determinePasskeyName(aaguid []byte, userAgent string) string {
	// First try to identify by AAGUID using a combination of builtin + MDS
	authenticatorName := s.mdsService.GetAuthenticatorName(aaguid)
	fmt.Println("AAGUID: ", authenticatorName)
	if authenticatorName != "" {
		return authenticatorName
	}

	// Now try to identify from User-Agent string
	// userAgentMap := map[string]string{
	// 	"Bitwarden": "Bitwarden",
	// 	"1Password": "1Password",
	// 	"LastPass":  "LastPass",
	// 	"Dashlane":  "Dashlane",
	// 	"KeePass":   "KeePass",
	// 	"Keeper":    "Keeper",
	// 	"Chrome":    "Chrome",
	// 	"Firefox":   "Firefox",
	// 	"Safari":    "Safari",
	// 	"Edge":      "Edge",
	// 	"Opera":     "Opera",
	// 	"Brave":     "Brave",
	// 	"Vivaldi":   "Vivaldi",
	// 	"iPhone":    "iOS Device",
	// 	"iPad":      "iPad",
	// 	"Android":   "Android Device",
	// 	"Windows":   "Windows Device",
	// 	"Macintosh": "Mac",
	// 	"Linux":     "Linux Device",
	// 	"Ubuntu":    "Ubuntu Device",
	// }

	// for keyword, name := range userAgentMap {
	// 	if strings.Contains(userAgent, keyword) {
	// 		return name + " Passkey"
	// 	}
	// }

	return "New Passkey" // Default fallback
}

func (s *WebAuthnService) BeginLogin() (*model.PublicKeyCredentialRequestOptions, error) {
	options, session, err := s.webAuthn.BeginDiscoverableLogin()
	if err != nil {
		return nil, err
	}

	sessionToStore := &model.WebauthnSession{
		ExpiresAt:        datatype.DateTime(session.Expires),
		Challenge:        session.Challenge,
		UserVerification: string(session.UserVerification),
	}

	if err := s.db.Create(&sessionToStore).Error; err != nil {
		return nil, err
	}

	return &model.PublicKeyCredentialRequestOptions{
		Response:  options.Response,
		SessionID: sessionToStore.ID,
		Timeout:   s.webAuthn.Config.Timeouts.Registration.Timeout,
	}, nil
}

func (s *WebAuthnService) VerifyLogin(sessionID string, credentialAssertionData *protocol.ParsedCredentialAssertionData, ipAddress, userAgent string) (model.User, string, error) {
	var storedSession model.WebauthnSession
	if err := s.db.First(&storedSession, "id = ?", sessionID).Error; err != nil {
		return model.User{}, "", err
	}

	session := webauthn.SessionData{
		Challenge: storedSession.Challenge,
		Expires:   storedSession.ExpiresAt.ToTime(),
	}

	var user *model.User
	_, err := s.webAuthn.ValidateDiscoverableLogin(func(_, userHandle []byte) (webauthn.User, error) {
		if err := s.db.Preload("Credentials").First(&user, "id = ?", string(userHandle)).Error; err != nil {
			return nil, err
		}
		return user, nil
	}, session, credentialAssertionData)

	if err != nil {
		return model.User{}, "", err
	}

	token, err := s.jwtService.GenerateAccessToken(*user)
	if err != nil {
		return model.User{}, "", err
	}

	s.auditLogService.CreateNewSignInWithEmail(ipAddress, userAgent, user.ID)

	return *user, token, nil
}

func (s *WebAuthnService) ListCredentials(userID string) ([]model.WebauthnCredential, error) {
	var credentials []model.WebauthnCredential
	if err := s.db.Find(&credentials, "user_id = ?", userID).Error; err != nil {
		return nil, err
	}
	return credentials, nil
}

func (s *WebAuthnService) DeleteCredential(userID, credentialID string) error {
	var credential model.WebauthnCredential
	if err := s.db.First(&credential, "id = ? AND user_id = ?", credentialID, userID).Error; err != nil {
		return err
	}

	if err := s.db.Delete(&credential).Error; err != nil {
		return err
	}

	return nil
}

func (s *WebAuthnService) UpdateCredential(userID, credentialID, name string) (model.WebauthnCredential, error) {
	var credential model.WebauthnCredential
	if err := s.db.Where("id = ? AND user_id = ?", credentialID, userID).First(&credential).Error; err != nil {
		return credential, err
	}

	credential.Name = name

	if err := s.db.Save(&credential).Error; err != nil {
		return credential, err
	}

	return credential, nil
}

// updateWebAuthnConfig updates the WebAuthn configuration with the app name as it can change during runtime
func (s *WebAuthnService) updateWebAuthnConfig() {
	s.webAuthn.Config.RPDisplayName = s.appConfigService.DbConfig.AppName.Value
}
