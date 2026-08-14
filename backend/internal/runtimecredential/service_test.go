//go:build unit

package runtimecredential

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/appconfig"
	"github.com/pocket-id/pocket-id/backend/internal/apperror"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
	"github.com/pocket-id/pocket-id/backend/internal/onetimeaccess"
	testutils "github.com/pocket-id/pocket-id/backend/internal/utils/testing"
)

type testSigner struct {
	method string
}

func (s *testSigner) GenerateAccessToken(_ model.User, method string, _ time.Duration) (string, error) {
	s.method = method
	return "test-access-token", nil
}

type testAuditLogger struct {
	events []model.AuditLogEvent
	data   []model.AuditLogData
}

func (a *testAuditLogger) Create(_ context.Context, event model.AuditLogEvent, _, _, _ string, data model.AuditLogData, _ *gorm.DB) (model.AuditLog, bool) {
	a.events = append(a.events, event)
	a.data = append(a.data, data)
	return model.AuditLog{}, true
}

func (a *testAuditLogger) CreateNewSignInWithEmail(_ context.Context, _, _, _ string, _ *gorm.DB, _ bool) model.AuditLog {
	a.events = append(a.events, model.AuditLogEventSignIn)
	return model.AuditLog{}
}

type testBootstrap struct {
	state    onetimeaccess.TokenState
	restored bool
}

func (b *testBootstrap) ConsumeToken(_ context.Context, _, _ string) (onetimeaccess.TokenState, error) {
	return b.state, nil
}

func (b *testBootstrap) RestoreToken(_ context.Context, _ string, _ onetimeaccess.TokenState) {
	b.restored = true
}

type testReauthIssuer struct{}

func (testReauthIssuer) CreateReauthenticationToken(_ context.Context, _ *gorm.DB, _ string) (string, error) {
	return "test-reauthentication-token", nil
}

func newRuntimeServiceForTest(t *testing.T, user model.User) (*Service, *gorm.DB, *testSigner, *testAuditLogger, *testBootstrap) {
	t.Helper()
	db := testutils.NewDatabaseForTest(t)
	require.NoError(t, db.Create(&user).Error)
	signer := &testSigner{}
	audit := &testAuditLogger{}
	bootstrap := &testBootstrap{state: onetimeaccess.TokenState{UserID: user.ID, ExpiresAt: time.Now().Add(time.Minute)}}
	service := newService(Dependencies{DB: db, Signer: signer, AuditLog: audit, Bootstrap: bootstrap, Reauth: testReauthIssuer{}})
	return service, db, signer, audit, bootstrap
}

func generateTestKey(t *testing.T) (ed25519.PublicKey, ed25519.PrivateKey) {
	t.Helper()
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	return publicKey, privateKey
}

func signChallenge(t *testing.T, challenge challengeDto, privateKey ed25519.PrivateKey) string {
	t.Helper()
	message, err := base64.RawURLEncoding.DecodeString(challenge.Challenge)
	require.NoError(t, err)
	return base64.RawURLEncoding.EncodeToString(ed25519.Sign(privateKey, message))
}

func TestRegistrationCreatesCredentialAndConsumesChallenge(t *testing.T) {
	service, db, signer, audit, _ := newRuntimeServiceForTest(t, model.User{Base: model.Base{ID: "runtime-user"}, Username: "vex", IsAgent: true})
	publicKey, privateKey := generateTestKey(t)

	challenge, err := service.BeginRegistration(t.Context(), registrationStartDto{
		Token:     "bootstrap-token",
		Name:      "Vex test runtime",
		Algorithm: model.RuntimeCredentialAlgorithmEd25519,
		PublicKey: base64.RawURLEncoding.EncodeToString(publicKey),
	}, "")
	require.NoError(t, err)

	user, credential, token, err := service.FinishRegistration(t.Context(), appconfig.NewTestConfig(nil), proofFinishDto{
		SessionID: challenge.SessionID,
		Signature: signChallenge(t, challenge, privateKey),
	}, "127.0.0.1", "test-runtime")
	require.NoError(t, err)
	require.Equal(t, "runtime-user", user.ID)
	require.Equal(t, "test-access-token", token)
	require.Equal(t, authenticationMethodProofOfPossession, signer.method)
	require.Equal(t, publicKey, ed25519.PublicKey(credential.PublicKey))
	require.Equal(t, []model.AuditLogEvent{model.AuditLogEventRuntimeCredentialRegistered, model.AuditLogEventSignIn}, audit.events)
	require.Equal(t, model.AuditLogData{"credentialID": credential.ID, "credentialName": credential.Name}, audit.data[0])

	var storedCount int64
	require.NoError(t, db.Model(&model.RuntimeCredential{}).Where("user_id = ? AND revoked_at IS NULL", user.ID).Count(&storedCount).Error)
	require.EqualValues(t, 1, storedCount)

	_, _, _, err = service.FinishRegistration(t.Context(), appconfig.NewTestConfig(nil), proofFinishDto{
		SessionID: challenge.SessionID,
		Signature: signChallenge(t, challenge, privateKey),
	}, "", "")
	require.ErrorIs(t, err, apperror.RuntimeCredentialInvalid())
}

func TestRegistrationRejectsInvalidInputsAndExistingCredential(t *testing.T) {
	service, db, _, _, _ := newRuntimeServiceForTest(t, model.User{Base: model.Base{ID: "validation-user"}, Username: "validation-user", IsAgent: true})
	publicKey, _ := generateTestKey(t)

	_, err := service.BeginRegistration(t.Context(), registrationStartDto{Token: "bootstrap-token", Name: "Runtime", Algorithm: "RSA", PublicKey: base64.RawURLEncoding.EncodeToString(publicKey)}, "")
	require.Error(t, err)
	_, err = service.BeginRegistration(t.Context(), registrationStartDto{Token: "bootstrap-token", Name: "Runtime", Algorithm: model.RuntimeCredentialAlgorithmEd25519, PublicKey: "invalid"}, "")
	require.Error(t, err)

	credential := model.RuntimeCredential{Base: model.Base{ID: "22222222-2222-4222-8222-222222222222"}, Name: "Existing", Algorithm: model.RuntimeCredentialAlgorithmEd25519, PublicKey: publicKey, UserID: "validation-user"}
	require.NoError(t, db.Create(&credential).Error)
	_, err = service.BeginRegistration(t.Context(), registrationStartDto{Token: "bootstrap-token", Name: "Second", Algorithm: model.RuntimeCredentialAlgorithmEd25519, PublicKey: base64.RawURLEncoding.EncodeToString(publicKey)}, "")
	require.ErrorIs(t, err, apperror.RuntimeCredentialExists())
}

func TestRegistrationRestoresBootstrapWhenPathIsInvalid(t *testing.T) {
	service, _, _, _, bootstrap := newRuntimeServiceForTest(t, model.User{Base: model.Base{ID: "passkey-user"}, Username: "human"})
	publicKey, _ := generateTestKey(t)

	_, err := service.BeginRegistration(t.Context(), registrationStartDto{
		Token:     "bootstrap-token",
		Name:      "Invalid path",
		Algorithm: model.RuntimeCredentialAlgorithmEd25519,
		PublicKey: base64.RawURLEncoding.EncodeToString(publicKey),
	}, "")
	require.ErrorIs(t, err, apperror.RuntimeCredentialInvalid())
	require.True(t, bootstrap.restored)
}

func TestLoginAndRevocationLifecycle(t *testing.T) {
	service, db, signer, audit, _ := newRuntimeServiceForTest(t, model.User{Base: model.Base{ID: "login-user"}, Username: "vex", IsAgent: true})
	publicKey, privateKey := generateTestKey(t)
	credential := model.RuntimeCredential{Base: model.Base{ID: "11111111-1111-4111-8111-111111111111"}, Name: "Runtime", Algorithm: model.RuntimeCredentialAlgorithmEd25519, PublicKey: publicKey, UserID: "login-user"}
	require.NoError(t, db.Create(&credential).Error)

	challenge, err := service.BeginLogin(t.Context(), loginStartDto{Username: "vex", CredentialID: credential.ID})
	require.NoError(t, err)
	user, token, err := service.FinishLogin(t.Context(), appconfig.NewTestConfig(nil), proofFinishDto{SessionID: challenge.SessionID, Signature: signChallenge(t, challenge, privateKey)}, "127.0.0.1", "test-runtime")
	require.NoError(t, err)
	require.Equal(t, "login-user", user.ID)
	require.Equal(t, "test-access-token", token)
	require.Equal(t, authenticationMethodProofOfPossession, signer.method)
	require.Contains(t, audit.events, model.AuditLogEventRuntimeCredentialAuthenticated)
	require.Contains(t, audit.events, model.AuditLogEventSignIn)

	var updated model.RuntimeCredential
	require.NoError(t, db.First(&updated, "id = ?", credential.ID).Error)
	require.NotNil(t, updated.LastUsedAt)

	require.NoError(t, service.RevokeCredential(t.Context(), user.ID, credential.ID, "127.0.0.1", "test-runtime", user.ID))
	require.NoError(t, db.First(&updated, "id = ?", credential.ID).Error)
	require.NotNil(t, updated.RevokedAt)
	_, err = service.BeginLogin(t.Context(), loginStartDto{Username: "vex", CredentialID: credential.ID})
	require.ErrorIs(t, err, apperror.RuntimeCredentialInvalid())
}

func TestLoginRejectsWrongIdentitySignatureExpiryAndDisabledUser(t *testing.T) {
	service, db, _, _, _ := newRuntimeServiceForTest(t, model.User{Base: model.Base{ID: "login-negative-user"}, Username: "login-negative", IsAgent: true})
	publicKey, _ := generateTestKey(t)
	credential := model.RuntimeCredential{Base: model.Base{ID: "33333333-3333-4333-8333-333333333333"}, Name: "Runtime", Algorithm: model.RuntimeCredentialAlgorithmEd25519, PublicKey: publicKey, UserID: "login-negative-user"}
	require.NoError(t, db.Create(&credential).Error)

	_, err := service.BeginLogin(t.Context(), loginStartDto{Username: "someone-else", CredentialID: credential.ID})
	require.ErrorIs(t, err, apperror.RuntimeCredentialInvalid())

	challenge, err := service.BeginLogin(t.Context(), loginStartDto{Username: "login-negative", CredentialID: credential.ID})
	require.NoError(t, err)
	_, wrongPrivateKey := generateTestKey(t)
	_, _, err = service.FinishLogin(t.Context(), appconfig.NewTestConfig(nil), proofFinishDto{SessionID: challenge.SessionID, Signature: signChallenge(t, challenge, wrongPrivateKey)}, "", "")
	require.ErrorIs(t, err, apperror.RuntimeCredentialInvalid())

	expiredAt := datatype.DateTime(time.Now().Add(-time.Minute))
	require.NoError(t, db.Model(&credential).Update("expires_at", expiredAt).Error)
	_, err = service.BeginLogin(t.Context(), loginStartDto{Username: "login-negative", CredentialID: credential.ID})
	require.ErrorIs(t, err, apperror.RuntimeCredentialInvalid())

	require.NoError(t, db.Model(&credential).Update("expires_at", nil).Error)
	require.NoError(t, db.Model(&model.User{}).Where("id = ?", credential.UserID).Update("disabled", true).Error)
	_, err = service.BeginLogin(t.Context(), loginStartDto{Username: "login-negative", CredentialID: credential.ID})
	require.ErrorIs(t, err, apperror.RuntimeCredentialInvalid())
}

func TestReauthenticationRenameAndListing(t *testing.T) {
	service, db, _, audit, _ := newRuntimeServiceForTest(t, model.User{Base: model.Base{ID: "managed-runtime-user"}, Username: "managed-runtime", IsAgent: true, IsAdmin: true})
	publicKey, privateKey := generateTestKey(t)
	credential := model.RuntimeCredential{Base: model.Base{ID: "44444444-4444-4444-8444-444444444444"}, Name: "Original", Algorithm: model.RuntimeCredentialAlgorithmEd25519, PublicKey: publicKey, UserID: "managed-runtime-user"}
	require.NoError(t, db.Create(&credential).Error)

	challenge, err := service.BeginReauthentication(t.Context(), credential.UserID, credential.ID)
	require.NoError(t, err)
	token, err := service.FinishReauthentication(t.Context(), credential.UserID, proofFinishDto{SessionID: challenge.SessionID, Signature: signChallenge(t, challenge, privateKey)}, "127.0.0.1", "test-runtime")
	require.NoError(t, err)
	require.Equal(t, "test-reauthentication-token", token)
	require.Contains(t, audit.events, model.AuditLogEventRuntimeCredentialAuthenticated)

	updated, err := service.UpdateCredential(t.Context(), credential.UserID, credential.ID, "Renamed runtime")
	require.NoError(t, err)
	require.Equal(t, "Renamed runtime", updated.Name)
	_, err = service.UpdateCredential(t.Context(), credential.UserID, credential.ID, "   ")
	require.Error(t, err)

	credentials, err := service.ListCredentials(t.Context(), credential.UserID)
	require.NoError(t, err)
	require.Len(t, credentials, 1)
	require.Equal(t, "Renamed runtime", credentials[0].Name)

	_, err = service.ListCredentials(t.Context(), "missing-user")
	require.ErrorIs(t, err, apperror.UserNotFound())
}

func TestAuthenticationPathDatabaseGuards(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	passkeyPathUser := model.User{Base: model.Base{ID: "passkey-path"}, Username: "passkey-path"}
	require.NoError(t, db.Create(&passkeyPathUser).Error)
	publicKey, _ := generateTestKey(t)
	err := db.Create(&model.RuntimeCredential{Name: "Wrong path", Algorithm: model.RuntimeCredentialAlgorithmEd25519, PublicKey: publicKey, UserID: passkeyPathUser.ID}).Error
	require.ErrorContains(t, err, "runtime credentials require the runtime authentication path")

	runtimePathUser := model.User{Base: model.Base{ID: "runtime-path"}, Username: "runtime-path", IsAgent: true}
	require.NoError(t, db.Create(&runtimePathUser).Error)
	err = db.Create(&model.WebauthnCredential{Name: "Wrong path", CredentialID: []byte("id"), PublicKey: []byte("key"), UserID: runtimePathUser.ID}).Error
	require.ErrorContains(t, err, "passkeys are not allowed on the runtime authentication path")
}
