package oidc

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pocket-id/fosite"
	"github.com/stretchr/testify/require"

	"github.com/pocket-id/pocket-id/backend/internal/claimmappingpolicy"
	"github.com/pocket-id/pocket-id/backend/internal/model"
)

func TestDeviceCodeExchangeConsumesOpenIDConnectSession(t *testing.T) {
	const (
		userID   = "test-user"
		clientID = "test-client"
	)
	service, store, provider, userCode, deviceCode := newTestDeviceServiceWithCode(t, clientID, userID, false, nil)
	require.NoError(t, service.acceptDeviceCode(t.Context(), userCode, userID, "phr", time.Now().UTC(), nil, "", requestMeta{}))

	signature, err := provider.deviceStrategy.DeviceCodeSignature(t.Context(), deviceCode)
	require.NoError(t, err)

	// The approved authorization keeps an OpenID Connect session that expires together with the device code
	var deviceSession, openIDSession OAuth2Session
	require.NoError(t, store.db.First(&deviceSession, "kind = ? AND key = ?", sessionKindDeviceCode, signature).Error)
	require.NoError(t, store.db.First(&openIDSession, "kind = ? AND key = ?", sessionKindOpenID, signature).Error)
	require.NotNil(t, deviceSession.ExpiresAt)
	require.NotNil(t, openIDSession.ExpiresAt)
	require.True(t, deviceSession.ExpiresAt.ToTime().Equal(openIDSession.ExpiresAt.ToTime()))

	// The token exchange issues an ID token and consumes the OpenID Connect session
	response, err := exchangeDeviceCode(t, provider, clientID, deviceCode)
	require.NoError(t, err)
	require.NotNil(t, response.GetExtra("id_token"))

	var count int64
	require.NoError(t, store.db.Model(&OAuth2Session{}).Where("kind = ?", sessionKindOpenID).Count(&count).Error)
	require.Zero(t, count)
	require.NoError(t, store.db.Model(&OAuth2Session{}).Where("kind = ? AND request_id = ?", sessionKindAccessToken, deviceSession.RequestID).Count(&count).Error)
	require.EqualValues(t, 1, count)

	// A replayed device code is rejected before an OpenID Connect session could be reused
	_, err = exchangeDeviceCode(t, provider, clientID, deviceCode)
	require.ErrorIs(t, err, fosite.ErrInvalidGrant)

	// The replay revokes the access token that the first exchange issued
	require.NoError(t, store.db.Model(&OAuth2Session{}).Where("kind = ? AND request_id = ?", sessionKindAccessToken, deviceSession.RequestID).Count(&count).Error)
	require.Zero(t, count)
}

func TestTokenHandlerDeviceCodeExchangeBuildsClaimsFromCurrentUserData(t *testing.T) {
	const (
		userID   = "test-user"
		clientID = "test-client"
	)
	service, store, provider, userCode, deviceCode := newTestDeviceServiceWithCode(t, clientID, userID, false, nil)
	require.NoError(t, store.db.Model(&claimmappingpolicy.OidcClaimMappingPolicy{}).Where("is_default = ?", true).Update("claim_mappings", claimmappingpolicy.OidcClaimMappings{
		{ClaimName: "mail", SourceType: claimmappingpolicy.MappingSourceUserField, SourceValue: "email", Scope: []string{"openid"}, AccessToken: true, IDToken: true},
	}).Error)
	require.NoError(t, store.db.Model(&model.User{}).Where("id = ?", userID).Update("email", "approved@example.com").Error)
	require.NoError(t, service.acceptDeviceCode(t.Context(), userCode, userID, "phr", time.Now().UTC(), nil, "", requestMeta{}))
	require.NoError(t, store.db.Model(&model.User{}).Where("id = ?", userID).Update("email", "current@example.com").Error)

	handler := newTokenHandler(provider, service.claimsService, nil, nil)
	form := url.Values{
		"grant_type":  {string(fosite.GrantTypeDeviceCode)},
		"device_code": {deviceCode},
		"client_id":   {clientID},
	}
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/oidc/token", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = req
	handler.token(c)

	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.NotEmpty(t, body["access_token"], "expected tokens, got error: %v", body["error"])
	require.Equal(t, "current@example.com", decodeJWTPart(t, body["access_token"].(string), 1)["mail"], "access token")
	require.Equal(t, "current@example.com", decodeJWTPart(t, body["id_token"].(string), 1)["mail"], "ID token")
}

func TestRevokeUserClientSessionsRevokesApprovedDeviceCode(t *testing.T) {
	const (
		userID   = "test-user"
		clientID = "test-client"
	)
	service, store, provider, userCode, deviceCode := newTestDeviceServiceWithCode(t, clientID, userID, false, nil)
	require.NoError(t, service.acceptDeviceCode(t.Context(), userCode, userID, "phr", time.Now().UTC(), nil, "", requestMeta{}))

	require.NoError(t, RevokeUserClientSessions(t.Context(), store.db, userID, clientID))

	_, err := exchangeDeviceCode(t, provider, clientID, deviceCode)
	require.ErrorIs(t, err, fosite.ErrInvalidGrant)
}

func TestStoreInvalidateDeviceCodeSessionReportsLostRaceAsInvalidGrant(t *testing.T) {
	_, store, provider, _, deviceCode := newTestDeviceServiceWithCode(t, "test-client", "test-user", false, nil)

	signature, err := provider.deviceStrategy.DeviceCodeSignature(t.Context(), deviceCode)
	require.NoError(t, err)

	// The first redemption wins and a concurrent second one must not mint another token set
	require.NoError(t, store.InvalidateDeviceCodeSession(t.Context(), signature))
	require.ErrorIs(t, store.InvalidateDeviceCodeSession(t.Context(), signature), fosite.ErrInvalidGrant)
}

// exchangeDeviceCode redeems the device code at the token endpoint like a polling device would
func exchangeDeviceCode(t *testing.T, provider *oidcProvider, clientID, deviceCode string) (fosite.AccessResponder, error) {
	t.Helper()

	form := url.Values{
		"grant_type":  {string(fosite.GrantTypeDeviceCode)},
		"device_code": {deviceCode},
		"client_id":   {clientID},
	}
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/oidc/token", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	accessRequest, err := provider.NewAccessRequest(t.Context(), req, NewEmptySession())
	if err != nil {
		return nil, err
	}
	return provider.NewAccessResponse(t.Context(), accessRequest)
}
