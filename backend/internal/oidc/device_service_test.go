package oidc

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pocket-id/fosite"
	"github.com/pocket-id/pocket-id/backend/internal/appconfig"
	"github.com/pocket-id/pocket-id/backend/internal/apperror"
	"github.com/pocket-id/pocket-id/backend/internal/authz"
	"github.com/pocket-id/pocket-id/backend/internal/claimmappingpolicy"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	testutils "github.com/pocket-id/pocket-id/backend/internal/utils/testing"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// #nosec G101
const testReauthenticationToken = "valid-reauth-token"

type fakeReauthenticationConsumer struct {
	token             string
	userID            string
	reauthenticatedAt time.Time
	calls             int
}

func (f *fakeReauthenticationConsumer) ConsumeReauthenticationToken(_ context.Context, _ *gorm.DB, token string, userID string) (time.Time, error) {
	f.calls++
	if token != f.token || userID != f.userID {
		return time.Time{}, apperror.ReauthenticationRequired()
	}

	return f.reauthenticatedAt, nil
}

func TestDeviceServiceAcceptRequiresReauthenticationTokenWhenClientRequiresIt(t *testing.T) {
	const (
		userID   = "test-user"
		clientID = "test-client"
	)
	reauth := &fakeReauthenticationConsumer{
		token:             testReauthenticationToken,
		userID:            userID,
		reauthenticatedAt: time.Now().UTC().Truncate(time.Second),
	}
	service, _, _, userCode, _ := newTestDeviceServiceWithCode(t, clientID, userID, true, reauth)

	err := service.acceptDeviceCode(t.Context(), userCode, userID, "phr", time.Now().UTC(), nil, "", requestMeta{})
	require.True(t, apperror.IsCode(err, apperror.CodeReauthenticationRequired))
	require.Zero(t, reauth.calls)

	info, err := service.getDeviceCodeInfo(t.Context(), userCode, userID)
	require.NoError(t, err)
	require.True(t, info.ReauthenticationRequired)

	err = service.acceptDeviceCode(t.Context(), userCode, userID, "phr", time.Now().UTC(), nil, reauth.token, requestMeta{})
	require.NoError(t, err)
	require.Equal(t, 1, reauth.calls)
}

// TestDeviceServiceInfoIgnoresConsentAndSkipConsent guards that a prior consent or the client's skip-consent setting never changes what the device consent screen gets, since anyone may have started the device request
func TestDeviceServiceInfoIgnoresConsentAndSkipConsent(t *testing.T) {
	const (
		userID   = "test-user"
		clientID = "test-client"
	)
	service, store, _ := newTestDeviceService(t, clientID, userID, false, nil, nil)

	// Describe and approve a first device code, which records the user's consent
	userCode, _ := createTestDeviceCode(t, service, clientID, "openid email")
	firstInfo, err := service.getDeviceCodeInfo(t.Context(), userCode, userID)
	require.NoError(t, err)
	require.NoError(t, service.acceptDeviceCode(t.Context(), userCode, userID, "phr", time.Now().UTC(), nil, "", requestMeta{}))
	require.NoError(t, store.db.First(&model.UserAuthorizedOidcClient{}, "user_id = ? AND client_id = ?", userID, clientID).Error)

	// A second device code for the now consented and skip-consent client is described exactly like the first one
	require.NoError(t, store.db.Model(&model.OidcClient{}).Where("id = ?", clientID).Update("skip_consent", true).Error)
	userCode, _ = createTestDeviceCode(t, service, clientID, "openid email")
	secondInfo, err := service.getDeviceCodeInfo(t.Context(), userCode, userID)
	require.NoError(t, err)
	require.Equal(t, firstInfo, secondInfo)
}

func TestDeviceServiceCreatesUserCodeWithOAuthPrefix(t *testing.T) {
	service, _, _, userCode, _ := newTestDeviceServiceWithCode(t, "test-client", "test-user", false, nil)

	require.Regexp(t, `^E[ABCDEFGHJKMNPQRSTUVWXYZ0123456789]{7}$`, userCode)
	_, _, err := service.deviceRequestFromUserCode(t.Context(), strings.ToLower(userCode))
	require.NoError(t, err)
}

func TestDeviceServiceAcceptUsesReauthenticationTimeForDeviceSession(t *testing.T) {
	const (
		userID   = "test-user"
		clientID = "test-client"
	)
	reauthenticatedAt := time.Now().Add(-30 * time.Second).UTC().Truncate(time.Second)
	reauth := &fakeReauthenticationConsumer{
		token:             testReauthenticationToken,
		userID:            userID,
		reauthenticatedAt: reauthenticatedAt,
	}
	service, store, provider, userCode, deviceCode := newTestDeviceServiceWithCode(t, clientID, userID, true, reauth)

	err := service.acceptDeviceCode(t.Context(), userCode, userID, "phr", time.Now().Add(-time.Hour).UTC(), nil, reauth.token, requestMeta{})
	require.NoError(t, err)

	deviceCodeSignature, err := provider.deviceStrategy.DeviceCodeSignature(t.Context(), deviceCode)
	require.NoError(t, err)
	acceptedRequest, err := store.GetDeviceCodeSession(t.Context(), deviceCodeSignature, NewEmptySession())
	require.NoError(t, err)
	session := acceptedRequest.GetSession().(*Session)
	require.Equal(t, reauthenticatedAt, session.IDTokenClaims().AuthTime)
}

// TestDeviceServiceAcceptGrantsOnlyApprovedScopes guards that scopes the user deselects on the device consent screen are neither consented to nor granted, even though the device authorization endpoint granted them up front
func TestDeviceServiceAcceptGrantsOnlyApprovedScopes(t *testing.T) {
	const (
		userID   = "test-user"
		clientID = "test-client"
	)
	service, store, provider := newTestDeviceService(t, clientID, userID, false, nil, nil)
	userCode, deviceCode := createTestDeviceCode(t, service, clientID, "openid email groups")

	// The user deselects email and does not send openid, which must still be kept
	require.NoError(t, service.acceptDeviceCode(t.Context(), userCode, userID, "phr", time.Now().UTC(), []string{"groups"}, "", requestMeta{}))

	// Consent is only recorded for the approved scopes
	var authorizedClient model.UserAuthorizedOidcClient
	require.NoError(t, store.db.First(&authorizedClient, "user_id = ? AND client_id = ?", userID, clientID).Error)
	require.ElementsMatch(t, []string{"openid", "groups"}, []string(authorizedClient.Scope))

	// The token the device receives only carries the approved scopes
	response, err := exchangeDeviceCode(t, provider, clientID, deviceCode)
	require.NoError(t, err)
	require.Equal(t, "openid groups", response.GetExtra("scope"))
}

// TestDeviceServiceAcceptDropsScopesTheUserCantHold guards that a scope granted up front, before the user was known, is dropped once a user who can't hold it approves
func TestDeviceServiceAcceptDropsScopesTheUserCantHold(t *testing.T) {
	const (
		userID   = "test-user"
		clientID = "test-client"
		audience = "https://id.example.com/api"
	)
	apiAccess := userAccess(map[string][]string{audience: {"users:read", "users:write"}})
	apiAccess.unheld = []string{"users:write"}
	service, _, provider := newTestDeviceService(t, clientID, userID, false, nil, apiAccess)

	form := url.Values{
		"client_id": {clientID},
		"resource":  {audience},
		"scope":     {"users:read users:write"},
	}
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/oidc/device/authorize", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	authorization, _, err := service.createDeviceAuthorization(t.Context(), req)
	require.NoError(t, err)

	info, err := service.getDeviceCodeInfo(t.Context(), authorization.UserCode, userID)
	require.NoError(t, err)
	require.Equal(t, []string{"users:read"}, info.Scope)

	require.NoError(t, service.acceptDeviceCode(t.Context(), authorization.UserCode, userID, "phr", time.Now().UTC(), nil, "", requestMeta{}))

	response, err := exchangeDeviceCode(t, provider, clientID, authorization.DeviceCode)
	require.NoError(t, err)
	require.Equal(t, "users:read", response.GetExtra("scope"))
}

func TestDeviceServiceAcceptRejectsUnrequestedScopes(t *testing.T) {
	const (
		userID   = "test-user"
		clientID = "test-client"
	)
	service, store, _ := newTestDeviceService(t, clientID, userID, false, nil, nil)
	userCode, _ := createTestDeviceCode(t, service, clientID, "openid email")

	// The approval may only remove scopes, so a scope the device never requested is rejected
	err := service.acceptDeviceCode(t.Context(), userCode, userID, "phr", time.Now().UTC(), []string{"email", "groups"}, "", requestMeta{})
	require.Error(t, err)

	// Nothing is consented to and the user code can still be approved
	require.ErrorIs(t, store.db.First(&model.UserAuthorizedOidcClient{}, "user_id = ? AND client_id = ?", userID, clientID).Error, gorm.ErrRecordNotFound)
	require.NoError(t, service.acceptDeviceCode(t.Context(), userCode, userID, "phr", time.Now().UTC(), nil, "", requestMeta{}))
}

func TestDeviceServiceCreateRejectsResourceWithoutUserDelegatedGrant(t *testing.T) {
	const (
		userID   = "test-user"
		clientID = "test-client"
		audience = "https://api.orders.example.com"
	)
	apiAccess := fakeAPIAccess{allowed: map[string]map[SubjectType][]string{
		audience: {
			SubjectTypeClient: {"write:orders"},
		},
	}}
	service, store, _ := newTestDeviceService(t, clientID, userID, false, nil, apiAccess)

	form := url.Values{
		"client_id": {clientID},
		"resource":  {audience},
		"scope":     {"write:orders"},
	}
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/oidc/device/authorize", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	_, _, err := service.createDeviceAuthorization(t.Context(), req)
	require.ErrorIs(t, err, fosite.ErrInvalidTarget)

	var count int64
	require.NoError(t, store.db.Model(&OAuth2Session{}).Where("kind IN ?", []string{sessionKindDeviceCode, sessionKindUserCode}).Count(&count).Error)
	require.Zero(t, count)
}

func TestDeviceServiceAcceptDoesNotResolveClientInsideTransaction(t *testing.T) {
	const userID = "test-user"
	service, store, _, userCode, _ := newTestDeviceServiceWithCode(t, "test-client", userID, false, nil)

	var resolved, resolvedInTx int
	store.clientResolver = fosite.ClientResolverFunc(func(ctx context.Context, id string, next fosite.ClientLookupFunc) (fosite.Client, error) {
		resolved++
		if _, inTx := ctx.Value(txContextKey{}).(*gorm.DB); inTx {
			resolvedInTx++
		}
		return next(ctx, id)
	})

	require.NoError(t, service.acceptDeviceCode(t.Context(), userCode, userID, "phr", time.Now().UTC(), nil, "", requestMeta{}))
	require.Positive(t, resolved)
	require.Zero(t, resolvedInTx, "a metadata document client could be fetched while the database write lock is held")
}

func newTestDeviceServiceWithCode(t *testing.T, clientID, userID string, requiresReauthentication bool, reauth ReauthenticationTokenConsumer) (*deviceService, *Store, *oidcProvider, string, string) {
	t.Helper()

	service, store, provider := newTestDeviceService(t, clientID, userID, requiresReauthentication, reauth, nil)
	userCode, deviceCode := createTestDeviceCode(t, service, clientID, "openid")

	return service, store, provider, userCode, deviceCode
}

// createTestDeviceCode starts a device authorization for the given scope like a device would
func createTestDeviceCode(t *testing.T, service *deviceService, clientID, scope string) (userCode string, deviceCode string) {
	t.Helper()

	form := url.Values{
		"client_id": {clientID},
		"scope":     {scope},
	}
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/oidc/device/authorize", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response, _, err := service.createDeviceAuthorization(t.Context(), req)
	require.NoError(t, err)

	return response.UserCode, response.DeviceCode
}

func newTestDeviceService(t *testing.T, clientID, userID string, requiresReauthentication bool, reauth ReauthenticationTokenConsumer, apiAccess APIAccessProvider) (*deviceService, *Store, *oidcProvider) {
	t.Helper()

	db := testutils.NewDatabaseForTest(t)
	require.NoError(t, claimmappingpolicy.Bootstrap(t.Context(), db))
	require.NoError(t, db.Create(&model.User{Base: model.Base{ID: userID}}).Error)
	require.NoError(t, db.Create(&model.OidcClient{
		Base:                     model.Base{ID: clientID},
		Name:                     "Test Client",
		IsPublic:                 true,
		RequiresReauthentication: requiresReauthentication,
	}).Error)

	store := NewStore(db, apiAccess)
	signerKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	// #nosec G101
	provider, err := newProvider(store, nil, testTokenSigner{key: signerKey}, Config{
		BaseURL:      "https://issuer.example.com",
		TokenBaseURL: "https://issuer.example.com",
		Secret:       []byte("test-secret"),
	}, nil)
	require.NoError(t, err)

	claimsService := newClaimsService(db, nil, "", nil, appconfig.StaticResolver{})
	authorizationService := newAuthorizationService(db, newInteractionSessionService(db), claimsService, reauth, &fakeAuditLogger{}, apiAccess)
	service := newDeviceService(provider, store, provider.deviceStrategy, authorizationService, claimsService, &fakeAuditLogger{}, db)

	return service, store, provider
}

func TestDeviceHandlerVerifyRequiresJSONBody(t *testing.T) {
	const (
		userID   = "test-user"
		clientID = "test-client"
	)
	service, _, provider, userCode, deviceCode := newTestDeviceServiceWithCode(t, clientID, userID, false, nil)
	handler := newDeviceHandler(provider, service)

	verify := func(body string) (int, *gin.Error) {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/verify?code="+url.QueryEscape(userCode), strings.NewReader(body))
		if body != "" {
			req.Header.Set("Content-Type", "application/json")
		}
		rec := httptest.NewRecorder()
		var handlerErr *gin.Error
		router := gin.New()
		router.POST("/verify", func(c *gin.Context) {
			authz.SetPrincipal(c, &authz.Principal{Kind: authz.KindSession, UserID: userID, AuthenticationTime: time.Now().UTC()})
			handler.verifyDeviceCode(c)
			handlerErr = c.Errors.Last()
		})
		router.ServeHTTP(rec, req)
		return rec.Code, handlerErr
	}

	// A forged POST that carries only the user code, like one smuggled through another endpoint's path, must not approve the device
	_, ginErr := verify("")
	require.NotNil(t, ginErr)
	_, err := exchangeDeviceCode(t, provider, clientID, deviceCode)
	require.ErrorIs(t, err, fosite.ErrAuthorizationPending)

	// The device page always sends a JSON body, and an empty one still approves every requested scope
	status, ginErr := verify("{}")
	require.Nil(t, ginErr)
	require.Equal(t, http.StatusNoContent, status)
	_, err = exchangeDeviceCode(t, provider, clientID, deviceCode)
	require.NoError(t, err)
}
