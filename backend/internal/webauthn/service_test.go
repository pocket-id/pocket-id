package webauthn

import (
	"bytes"
	"errors"
	"testing"
	"time"

	"github.com/go-webauthn/webauthn/metadata"
	"github.com/go-webauthn/webauthn/metadata/providers/memory"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/protocol/webauthncbor"
	"github.com/go-webauthn/webauthn/protocol/webauthncose"
	gowebauthn "github.com/go-webauthn/webauthn/webauthn"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pocket-id/pocket-id/backend/internal/appconfig"
	"github.com/pocket-id/pocket-id/backend/internal/apperror"
	"github.com/pocket-id/pocket-id/backend/internal/auditlogs"
	"github.com/pocket-id/pocket-id/backend/internal/authz"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
	testutils "github.com/pocket-id/pocket-id/backend/internal/utils/testing"
)

func TestCreateReauthenticationTokenWithSession(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	user := model.User{
		Base:     model.Base{ID: "reauth-user"},
		Username: "reauth-user",
	}
	require.NoError(t, db.Create(&user).Error)
	service := &Service{db: db}

	tests := []struct {
		name                 string
		authenticationMethod string
		authenticatedAgo     time.Duration
		wantToken            bool
	}{
		{name: "fresh passkey sign-in", authenticationMethod: authenticationMethodPhishingResistant, wantToken: true},
		{name: "passkey sign-in older than a minute", authenticationMethod: authenticationMethodPhishingResistant, authenticatedAgo: 2 * time.Minute},
		{name: "fresh login code sign-in", authenticationMethod: "otp"},
		{name: "fresh sign-in without an authentication method"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			reauthenticationToken, err := service.CreateReauthenticationTokenWithSession(t.Context(), authz.Principal{
				UserID:               user.ID,
				AuthenticationMethod: test.authenticationMethod,
				AuthenticationTime:   time.Now().Add(-test.authenticatedAgo),
			})

			if test.wantToken {
				require.NoError(t, err)
				assert.NotEmpty(t, reauthenticationToken)
				return
			}
			assert.Empty(t, reauthenticationToken)
			assert.True(t, apperror.IsCode(err, apperror.CodeReauthenticationRequired))
		})
	}
}

func TestWebAuthnDisplayNameUsesRequestConfig(t *testing.T) {
	service, err := newService(Dependencies{
		DB:     testutils.NewDatabaseForTest(t),
		AppURL: "https://example.com",
	})
	require.NoError(t, err)
	require.Equal(t, defaultRPDisplayName, service.webAuthn.Config.RPDisplayName)

	service.updateWebAuthnConfig(&appconfig.AppConfigModel{AppName: "Custom App"})
	require.Equal(t, "Custom App", service.webAuthn.Config.RPDisplayName)
}

func TestBeginCeremoniesUseRequestConfig(t *testing.T) {
	tests := []struct {
		name                 string
		userVerification     appconfig.AppConfigValue
		authenticator        appconfig.AppConfigValue
		attestation          appconfig.AppConfigValue
		wantUserVerification protocol.UserVerificationRequirement
		wantAuthenticator    protocol.AuthenticatorAttachment
		wantAttestation      protocol.ConveyancePreference
	}{
		{
			name:                 "required verification with any authenticator",
			userVerification:     "required",
			authenticator:        "any",
			attestation:          "none",
			wantUserVerification: protocol.VerificationRequired,
			wantAuthenticator:    "",
			wantAttestation:      "",
		},
		{
			name:                 "required verification with a verified platform authenticator",
			userVerification:     "required",
			authenticator:        "platform",
			attestation:          "verified",
			wantUserVerification: protocol.VerificationRequired,
			wantAuthenticator:    protocol.Platform,
			wantAttestation:      protocol.PreferDirectAttestation,
		},
		{
			name:                 "preferred verification with a cross-platform authenticator",
			userVerification:     "preferred",
			authenticator:        "cross-platform",
			wantUserVerification: protocol.VerificationPreferred,
			wantAuthenticator:    protocol.CrossPlatform,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			db := testutils.NewDatabaseForTest(t)
			user := model.User{
				Base:     model.Base{ID: "configured-user"},
				Username: "configured-user",
			}
			require.NoError(t, db.Create(&user).Error)

			service, err := newService(Dependencies{
				DB:     db,
				AppURL: "https://example.com",
			})
			require.NoError(t, err)

			// Loaded metadata keeps the preload from downloading the real blob
			provider, err := memory.New(memory.WithMetadata(map[uuid.UUID]*metadata.Entry{}))
			require.NoError(t, err)
			service.fidoMetadata = &fidoMetadata{provider: provider, refreshAt: time.Now().Add(time.Hour)}

			dbConfig := &appconfig.AppConfigModel{
				AppName:                         "Configured App",
				WebauthnUserVerification:        tc.userVerification,
				WebauthnAuthenticatorAttachment: tc.authenticator,
				WebauthnAttestation:             tc.attestation,
			}

			registration, err := service.BeginRegistration(t.Context(), dbConfig, user.ID)
			require.NoError(t, err)
			assert.Equal(t, tc.wantAttestation, registration.Response.Attestation)
			assert.Equal(t, tc.wantUserVerification, registration.Response.AuthenticatorSelection.UserVerification)
			assert.Equal(t, tc.wantAuthenticator, registration.Response.AuthenticatorSelection.AuthenticatorAttachment)
			assert.Equal(t, protocol.ResidentKeyRequirementRequired, registration.Response.AuthenticatorSelection.ResidentKey)

			// Preserve requested extensions so browser outputs remain valid after loading the session
			var session WebauthnSession
			require.NoError(t, db.First(&session, "id = ?", registration.SessionID).Error)
			assert.Contains(t, session.Extensions.Requested, protocol.ExtensionCredProps)
			outputs := protocol.AuthenticationExtensionsClientOutputs{
				CredProps: &protocol.CredentialPropertiesOutput{RK: new(true)},
			}
			require.NoError(t, outputs.Verify(session.Extensions, protocol.CreateCeremony, protocol.UnsolicitedOutputPolicyReject))

			login, err := service.BeginLogin(t.Context(), dbConfig)
			require.NoError(t, err)
			assert.Equal(t, tc.wantUserVerification, login.Response.UserVerification)
		})
	}
}

func TestSyncedPasskeyPolicy(t *testing.T) {
	credential := &gowebauthn.Credential{
		Flags: gowebauthn.CredentialFlags{BackupEligible: true},
	}

	require.NoError(t, validateCredentialPolicy(&appconfig.AppConfigModel{WebauthnAllowSyncedPasskeys: "true"}, credential))

	err := validateCredentialPolicy(&appconfig.AppConfigModel{WebauthnAllowSyncedPasskeys: "false"}, credential)
	require.True(t, apperror.IsCode(err, apperror.CodeSyncedPasskeyNotAllowed))

	credential.Flags.BackupEligible = false
	require.NoError(t, validateCredentialPolicy(&appconfig.AppConfigModel{WebauthnAllowSyncedPasskeys: "false"}, credential))
}

func TestAuthenticatorAttachmentPolicy(t *testing.T) {
	credential := &gowebauthn.Credential{
		Authenticator: gowebauthn.Authenticator{Attachment: protocol.Platform},
	}

	require.NoError(t, validateCredentialPolicy(&appconfig.AppConfigModel{WebauthnAuthenticatorAttachment: "any"}, credential))
	require.NoError(t, validateCredentialPolicy(&appconfig.AppConfigModel{WebauthnAuthenticatorAttachment: "platform"}, credential))

	err := validateCredentialPolicy(&appconfig.AppConfigModel{WebauthnAuthenticatorAttachment: "cross-platform"}, credential)
	require.True(t, apperror.IsCode(err, apperror.CodeAuthenticatorAttachmentNotAllowed))

	credential.Authenticator.Attachment = ""
	err = validateCredentialPolicy(&appconfig.AppConfigModel{WebauthnAuthenticatorAttachment: "cross-platform"}, credential)
	require.True(t, apperror.IsCode(err, apperror.CodeAuthenticatorAttachmentNotAllowed), "a missing attachment must not bypass the policy")
}

func TestClassifyPasskeyErrorRecognizesMissingUserVerification(t *testing.T) {
	rpIDHash := make([]byte, 32)
	authenticatorData := protocol.AuthenticatorData{
		RPIDHash: rpIDHash,
		Flags:    protocol.FlagUserPresent,
	}
	cause := authenticatorData.Verify(rpIDHash, nil, true, true)
	require.Error(t, cause)

	err := classifyPasskeyError(cause, apperror.WebAuthnAuthenticationFailed)

	require.True(t, apperror.IsCode(err, apperror.CodePasskeyUserVerificationRequired))
	require.ErrorIs(t, err, cause)

	other := protocol.ErrVerification.WithInfo("RP Hash mismatch")
	err = classifyPasskeyError(other, apperror.WebAuthnAuthenticationFailed)
	require.True(t, apperror.IsCode(err, apperror.CodeWebAuthnAuthenticationFailed))
}

func TestClassifyPasskeyErrorPreservesStructuredLookupFailure(t *testing.T) {
	// Wrap a structured database failure exactly as go-webauthn wraps callback errors
	cause := errors.New("database unavailable")
	lookupErr := protocol.ErrBadRequest.WithError(apperror.Internal(cause))

	// Verify the structured failure and diagnostic cause both survive classification
	err := classifyPasskeyError(lookupErr, apperror.WebAuthnAuthenticationFailed)

	require.True(t, apperror.IsCode(err, apperror.CodeInternal))
	require.ErrorIs(t, err, cause)
}

func TestDiscardUnrequestedFalseAppIDOutput(t *testing.T) {
	tests := []struct {
		name      string
		requested []string
		appID     bool
		extra     map[string]any
		wantAppID *bool
		wantError string
	}{
		{
			name:      "unrequested false appid is discarded",
			appID:     false,
			wantAppID: nil,
		},
		{
			name:      "requested false appid is preserved",
			requested: []string{protocol.ExtensionAppID},
			appID:     false,
			wantAppID: new(false),
		},
		{
			name:      "unrequested true appid is rejected",
			appID:     true,
			wantAppID: new(true),
			wantError: "appid",
		},
		{
			name:      "other unsolicited output is rejected",
			appID:     false,
			extra:     map[string]any{"example": true},
			wantAppID: nil,
			wantError: "example",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			credential := &protocol.ParsedCredentialAssertionData{
				ParsedPublicKeyCredential: protocol.ParsedPublicKeyCredential{
					ClientExtensionResults: protocol.AuthenticationExtensionsClientOutputs{
						AppID: new(tc.appID),
						Extra: tc.extra,
					},
				},
			}
			session := protocol.SessionExtensions{Requested: tc.requested}

			discardUnrequestedFalseAppIDOutput(session, credential)

			assert.Equal(t, tc.wantAppID, credential.ClientExtensionResults.AppID)
			err := credential.ClientExtensionResults.Verify(session, protocol.AssertCeremony, protocol.UnsolicitedOutputPolicyReject)
			if tc.wantError == "" {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, tc.wantError)
			}
		})
	}
}

func TestWebAuthnManagementOperationsReturnSpecificNotFoundErrors(t *testing.T) {
	service, err := newService(Dependencies{
		DB:     testutils.NewDatabaseForTest(t),
		AppURL: "https://example.com",
	})
	require.NoError(t, err)

	_, err = service.BeginRegistration(t.Context(), &appconfig.AppConfigModel{}, "missing-user")
	require.True(t, apperror.IsCode(err, apperror.CodeUserNotFound))

	_, err = service.UpdateCredential(t.Context(), "missing-user", "missing-passkey", "New name")
	require.True(t, apperror.IsCode(err, apperror.CodeNotFound))

	err = service.DeleteCredential(t.Context(), "missing-user", "missing-passkey", auditlogs.Origin{})
	require.True(t, apperror.IsCode(err, apperror.CodeNotFound))
}

// A ceremony that references a session which does not exist must be rejected outright
// The delete-and-return leaves the struct zero-valued when nothing matched, and a zero session has an
// empty user verification requirement, a zero expiry and an empty challenge, so letting it reach the
// library would validate the assertion with user verification and expiry enforcement silently disabled
func TestCeremoniesRejectSessionThatDoesNotExist(t *testing.T) {
	const userID = "ceremony-user"

	setupService := func(t *testing.T) *Service {
		t.Helper()

		db := testutils.NewDatabaseForTest(t)
		require.NoError(t, db.Create(&model.User{
			Base:     model.Base{ID: userID},
			Username: userID,
		}).Error)

		return &Service{db: db}
	}

	t.Run("registration rejects an unknown session", func(t *testing.T) {
		service := setupService(t)

		_, err := service.VerifyRegistration(t.Context(), &appconfig.AppConfigModel{}, "does-not-exist", userID, nil, "127.0.0.1", "")

		require.Error(t, err)
		assert.True(t, apperror.IsCode(err, apperror.CodeInvalidWebAuthnSession))
	})

	t.Run("login rejects an unknown session", func(t *testing.T) {
		service := setupService(t)

		_, token, err := service.VerifyLogin(t.Context(), &appconfig.AppConfigModel{}, "does-not-exist", nil, "127.0.0.1", "test-agent", "", true)

		assert.Empty(t, token)
		require.Error(t, err)
		assert.True(t, apperror.IsCode(err, apperror.CodeInvalidWebAuthnSession))
	})

	t.Run("reauthentication rejects an unknown session", func(t *testing.T) {
		service := setupService(t)

		token, err := service.CreateReauthenticationTokenWithWebauthn(t.Context(), "does-not-exist", nil)

		assert.Empty(t, token)
		require.Error(t, err)
		assert.True(t, apperror.IsCode(err, apperror.CodeInvalidWebAuthnSession))
	})

	// The reauthentication query filters expired rows out in SQL, so an expired session matches no row
	// and previously produced the same zero-valued session as an unknown one
	t.Run("reauthentication rejects an expired session", func(t *testing.T) {
		service := setupService(t)

		expiredSession := WebauthnSession{
			Challenge:        "expired-challenge",
			ExpiresAt:        datatype.DateTime(time.Now().Add(-time.Minute)),
			UserVerification: string(protocol.VerificationRequired),
		}
		require.NoError(t, service.db.Create(&expiredSession).Error)

		token, err := service.CreateReauthenticationTokenWithWebauthn(t.Context(), expiredSession.ID, nil)

		assert.Empty(t, token)
		require.Error(t, err)
		assert.True(t, apperror.IsCode(err, apperror.CodeInvalidWebAuthnSession))
	})
}

func TestConsumeReauthenticationTokenReturnsTokenCreationTime(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	service := &Service{db: db}

	const (
		userID = "reauth-user"
		token  = "reauthentication-token"
	)
	require.NoError(t, db.Create(&model.User{
		Base: model.Base{ID: userID},
	}).Error)
	require.NoError(t, db.Create(&ReauthenticationToken{
		Token:     utils.CreateSha256Hash(token),
		ExpiresAt: datatype.DateTime(time.Now().Add(time.Minute)),
		UserID:    userID,
	}).Error)

	var storedToken ReauthenticationToken
	require.NoError(t, db.First(&storedToken, "user_id = ?", userID).Error)

	tx := db.Begin()
	reauthenticatedAt, err := service.ConsumeReauthenticationToken(t.Context(), tx, token, userID)
	require.NoError(t, err)
	require.NoError(t, tx.Commit().Error)

	require.Equal(t, storedToken.CreatedAt.UTC(), reauthenticatedAt)
}

func TestUpdateCredentialKeepsAAGUID(t *testing.T) {
	const (
		userID = "icon-user"
		aaguid = "bada5566-a7aa-401f-bd96-45619a55120d"
	)

	db := testutils.NewDatabaseForTest(t)
	require.NoError(t, db.Create(&model.User{
		Base:     model.Base{ID: userID},
		Username: userID,
	}).Error)

	credential := WebauthnCredential{
		Name:         "Original name",
		CredentialID: []byte("test-credential"),
		PublicKey:    []byte("test-public-key"),
		UserID:       userID,
		AAGUID:       aaguid,
	}
	require.NoError(t, db.Create(&credential).Error)

	service := &Service{db: db}

	updated, err := service.UpdateCredential(t.Context(), userID, credential.ID, "New name")
	require.NoError(t, err)
	assert.Equal(t, "New name", updated.Name)
	assert.Equal(t, aaguid, updated.AAGUID)

	var stored WebauthnCredential
	require.NoError(t, db.First(&stored, "id = ?", credential.ID).Error)
	assert.Equal(t, "New name", stored.Name)
	assert.Equal(t, aaguid, stored.AAGUID)
}

func TestHasOversizedPublicKey(t *testing.T) {
	rsaKey := func(bits int) []byte {
		key, err := webauthncbor.Marshal(webauthncose.RSAPublicKeyData{
			PublicKeyData: webauthncose.PublicKeyData{KeyType: int64(webauthncose.RSAKey), Algorithm: int64(webauthncose.AlgRS256)},
			Modulus:       bytes.Repeat([]byte{0xff}, bits/8),
			Exponent:      []byte{1, 0, 1},
		})
		require.NoError(t, err)
		return key
	}

	assert.False(t, hasOversizedPublicKey(rsaKey(maxRSAKeyBits)))
	assert.True(t, hasOversizedPublicKey(rsaKey(maxRSAKeyBits+8)))
}
