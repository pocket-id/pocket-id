package webauthn

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/go-webauthn/webauthn/protocol"
	gowebauthn "github.com/go-webauthn/webauthn/webauthn"
	"github.com/lestrrat-go/jwx/v3/jwt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pocket-id/pocket-id/backend/internal/appconfig"
	"github.com/pocket-id/pocket-id/backend/internal/apperror"
	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
	testutils "github.com/pocket-id/pocket-id/backend/internal/utils/testing"
)

// fakeSigner is an in-memory TokenService that mints opaque tokens carrying a subject,
// an issued-at time and an optional authentication method, without any real signing
type fakeSigner struct {
	tokens  map[string]jwt.Token
	counter int
}

func newFakeSigner() *fakeSigner {
	return &fakeSigner{tokens: map[string]jwt.Token{}}
}

func (s *fakeSigner) GenerateAccessToken(user model.User, authenticationMethod string, _ time.Duration) (string, error) {
	builder := jwt.NewBuilder().
		Subject(user.ID).
		IssuedAt(time.Now())
	if authenticationMethod != "" {
		builder = builder.Claim(common.AuthenticationMethodsClaim, []string{authenticationMethod})
	}
	token, err := builder.Build()
	if err != nil {
		return "", err
	}

	s.counter++
	raw := fmt.Sprintf("fake-access-token-%d", s.counter)
	s.tokens[raw] = token
	return raw, nil
}

func (s *fakeSigner) VerifyAccessToken(tokenString string) (jwt.Token, error) {
	token, ok := s.tokens[tokenString]
	if !ok {
		return nil, errors.New("invalid token")
	}
	return token, nil
}

func (s *fakeSigner) GetAuthenticationMethod(token jwt.Token) (string, error) {
	if !token.Has(common.AuthenticationMethodsClaim) {
		return "", nil
	}
	var methods []string
	if err := token.Get(common.AuthenticationMethodsClaim, &methods); err != nil {
		return "", err
	}
	if len(methods) == 0 {
		return "", nil
	}
	return methods[0], nil
}

func TestCreateReauthenticationTokenWithAccessToken(t *testing.T) {
	setupService := func(t *testing.T) (*Service, *fakeSigner, model.User) {
		t.Helper()

		db := testutils.NewDatabaseForTest(t)
		user := model.User{
			Base:     model.Base{ID: "reauth-user"},
			Username: "reauth-user",
		}
		require.NoError(t, db.Create(&user).Error)

		signer := newFakeSigner()
		return &Service{db: db, signer: signer}, signer, user
	}

	t.Run("accepts a fresh access token from WebAuthn login", func(t *testing.T) {
		service, signer, user := setupService(t)
		accessToken, err := signer.GenerateAccessToken(user, authenticationMethodPhishingResistant, time.Hour)
		require.NoError(t, err)

		reauthenticationToken, err := service.CreateReauthenticationTokenWithAccessToken(t.Context(), accessToken)

		require.NoError(t, err)
		assert.NotEmpty(t, reauthenticationToken)
	})

	t.Run("accepts a fresh access token from runtime proof login", func(t *testing.T) {
		service, signer, user := setupService(t)
		accessToken, err := signer.GenerateAccessToken(user, authenticationMethodProofOfPossession, time.Hour)
		require.NoError(t, err)

		reauthenticationToken, err := service.CreateReauthenticationTokenWithAccessToken(t.Context(), accessToken)

		require.NoError(t, err)
		assert.NotEmpty(t, reauthenticationToken)
	})

	t.Run("rejects a fresh access token from one-time access login", func(t *testing.T) {
		service, signer, user := setupService(t)
		accessToken, err := signer.GenerateAccessToken(user, "otp", time.Hour)
		require.NoError(t, err)

		reauthenticationToken, err := service.CreateReauthenticationTokenWithAccessToken(t.Context(), accessToken)

		assert.Empty(t, reauthenticationToken)
		require.Error(t, err)
		assert.True(t, apperror.IsCode(err, apperror.CodeReauthenticationRequired))
	})

	t.Run("rejects a fresh access token without an authentication method", func(t *testing.T) {
		service, signer, user := setupService(t)
		accessToken, err := signer.GenerateAccessToken(user, "", time.Hour)
		require.NoError(t, err)

		reauthenticationToken, err := service.CreateReauthenticationTokenWithAccessToken(t.Context(), accessToken)

		assert.Empty(t, reauthenticationToken)
		require.Error(t, err)
		assert.True(t, apperror.IsCode(err, apperror.CodeReauthenticationRequired))
	})

	t.Run("classifies an invalid access token as missing reauthentication", func(t *testing.T) {
		service, _, _ := setupService(t)

		reauthenticationToken, err := service.CreateReauthenticationTokenWithAccessToken(t.Context(), "invalid")

		assert.Empty(t, reauthenticationToken)
		require.True(t, apperror.IsCode(err, apperror.CodeReauthenticationRequired))
	})
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
		wantUserVerification protocol.UserVerificationRequirement
		wantAuthenticator    protocol.AuthenticatorAttachment
	}{
		{
			name:                 "required verification with any authenticator",
			userVerification:     "required",
			authenticator:        "any",
			wantUserVerification: protocol.VerificationRequired,
			wantAuthenticator:    "",
		},
		{
			name:                 "required verification with a platform authenticator",
			userVerification:     "required",
			authenticator:        "platform",
			wantUserVerification: protocol.VerificationRequired,
			wantAuthenticator:    protocol.Platform,
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

			dbConfig := &appconfig.AppConfigModel{
				AppName:                         "Configured App",
				WebauthnUserVerification:        tc.userVerification,
				WebauthnAuthenticatorAttachment: tc.authenticator,
			}

			registration, err := service.BeginRegistration(t.Context(), dbConfig, user.ID)
			require.NoError(t, err)
			assert.Equal(t, tc.wantUserVerification, registration.Response.AuthenticatorSelection.UserVerification)
			assert.Equal(t, tc.wantAuthenticator, registration.Response.AuthenticatorSelection.AuthenticatorAttachment)
			assert.Equal(t, protocol.ResidentKeyRequirementRequired, registration.Response.AuthenticatorSelection.ResidentKey)

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

	err = service.DeleteCredential(t.Context(), "missing-user", "missing-passkey", "", "", "")
	require.True(t, apperror.IsCode(err, apperror.CodeNotFound))
}

func TestBeginRegistrationRejectsRuntimeAuthenticationPath(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	user := model.User{Base: model.Base{ID: "runtime-user"}, Username: "runtime-user", IsAgent: true}
	require.NoError(t, db.Create(&user).Error)
	service, err := newService(Dependencies{DB: db, AppURL: "https://example.com"})
	require.NoError(t, err)

	_, err = service.BeginRegistration(t.Context(), &appconfig.AppConfigModel{}, user.ID)
	require.ErrorIs(t, err, apperror.AuthenticationPathMismatch())
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

		_, err := service.VerifyRegistration(t.Context(), &appconfig.AppConfigModel{}, "does-not-exist", userID, nil, "127.0.0.1")

		require.Error(t, err)
		assert.True(t, apperror.IsCode(err, apperror.CodeInvalidWebAuthnSession))
	})

	t.Run("login rejects an unknown session", func(t *testing.T) {
		service := setupService(t)

		_, token, err := service.VerifyLogin(t.Context(), &appconfig.AppConfigModel{}, "does-not-exist", nil, "127.0.0.1", "test-agent")

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
