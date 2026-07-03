package oidc

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	testutils "github.com/pocket-id/pocket-id/backend/internal/utils/testing"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type fakeReauthenticationConsumer struct {
	token             string
	userID            string
	reauthenticatedAt time.Time
	calls             int
}

func (f *fakeReauthenticationConsumer) ConsumeReauthenticationToken(_ context.Context, _ *gorm.DB, token string, userID string) (time.Time, error) {
	f.calls++
	if token != f.token || userID != f.userID {
		return time.Time{}, &common.ReauthenticationRequiredError{}
	}

	return f.reauthenticatedAt, nil
}

func TestDeviceServiceAcceptRequiresReauthenticationTokenWhenClientRequiresIt(t *testing.T) {
	const (
		userID   = "test-user"
		clientID = "test-client"
	)
	reauth := &fakeReauthenticationConsumer{ //nolint:gosec // test fixture token, not a real credential
		token:             "valid-reauth-token",
		userID:            userID,
		reauthenticatedAt: time.Now().UTC().Truncate(time.Second),
	}
	service, _, _, userCode, _ := newTestDeviceServiceWithCode(t, clientID, userID, true, reauth)

	err := service.acceptDeviceCode(t.Context(), userCode, userID, "phr", time.Now().UTC(), "", requestMeta{})
	require.ErrorAs(t, err, new(*common.ReauthenticationRequiredError))
	require.Zero(t, reauth.calls)

	info, err := service.getDeviceCodeInfo(t.Context(), userCode, userID)
	require.NoError(t, err)
	require.True(t, info.ReauthenticationRequired)

	err = service.acceptDeviceCode(t.Context(), userCode, userID, "phr", time.Now().UTC(), reauth.token, requestMeta{})
	require.NoError(t, err)
	require.Equal(t, 1, reauth.calls)
}

func TestDeviceServiceAcceptUsesReauthenticationTimeForDeviceSession(t *testing.T) {
	const (
		userID   = "test-user"
		clientID = "test-client"
	)
	reauthenticatedAt := time.Now().Add(-30 * time.Second).UTC().Truncate(time.Second)
	reauth := &fakeReauthenticationConsumer{ //nolint:gosec // test fixture token, not a real credential
		token:             "valid-reauth-token",
		userID:            userID,
		reauthenticatedAt: reauthenticatedAt,
	}
	service, store, provider, userCode, deviceCode := newTestDeviceServiceWithCode(t, clientID, userID, true, reauth)

	err := service.acceptDeviceCode(t.Context(), userCode, userID, "phr", time.Now().Add(-time.Hour).UTC(), reauth.token, requestMeta{})
	require.NoError(t, err)

	deviceCodeSignature, err := provider.deviceStrategy.DeviceCodeSignature(t.Context(), deviceCode)
	require.NoError(t, err)
	acceptedRequest, err := store.GetDeviceCodeSession(t.Context(), deviceCodeSignature, NewEmptySession())
	require.NoError(t, err)
	session := acceptedRequest.GetSession().(*Session)
	require.Equal(t, reauthenticatedAt, session.IDTokenClaims().AuthTime)
}

func newTestDeviceServiceWithCode(t *testing.T, clientID, userID string, requiresReauthentication bool, reauth ReauthenticationTokenConsumer) (*deviceService, *Store, *oidcProvider, string, string) {
	t.Helper()

	db := testutils.NewDatabaseForTest(t)
	require.NoError(t, db.Create(&model.User{Base: model.Base{ID: userID}}).Error)
	require.NoError(t, db.Create(&model.OidcClient{
		Base:                     model.Base{ID: clientID},
		Name:                     "Test Client",
		IsPublic:                 true,
		RequiresReauthentication: requiresReauthentication,
	}).Error)

	store := NewStore(db)
	signerKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	provider, err := newProvider(store, nil, testTokenSigner{key: signerKey}, Config{ //nolint:gosec // static test-only provider secret
		BaseURL:      "https://issuer.example.com",
		TokenBaseURL: "https://issuer.example.com",
		Secret:       []byte("test-secret"),
	})
	require.NoError(t, err)

	claimsService := newClaimsService(db, nil, "", nil)
	authorizationService := newAuthorizationService(db, newInteractionSessionService(db), claimsService, reauth, &fakeAuditLogger{})
	service := newDeviceService(provider, store, provider.deviceStrategy, authorizationService, claimsService, &fakeAuditLogger{}, db)

	form := url.Values{
		"client_id": {clientID},
		"scope":     {"openid"},
	}
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/oidc/device/authorize", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response, _, err := service.createDeviceAuthorization(t.Context(), req)
	require.NoError(t, err)

	return service, store, provider, response.UserCode, response.DeviceCode
}
