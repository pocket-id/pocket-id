package oidc

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ory/fosite"
	"github.com/ory/fosite/compose"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/model"
	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
	testutils "github.com/pocket-id/pocket-id/backend/internal/utils/testing"
	"github.com/pocket-id/pocket-id/backend/resources"
)

func TestRefreshTokenRotationGrace(t *testing.T) {
	gin.SetMode(gin.TestMode)
	const clientID, userID, requestID = "grace-client", "grace-user", "grace-request"
	const issuer, secret = "https://issuer.example.com", "test-secret"
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	globalSecret, err := DeriveGlobalSecret([]byte(secret))
	require.NoError(t, err)
	strategy := compose.NewOAuth2HMACStrategy(&fosite.Config{GlobalSecret: globalSecret})

	for _, scenario := range []string{"retry", "concurrent", "reuse after grace", "expired token", "wrong client", "disabled user", "token revocation", "logout", "app revocation"} {
		t.Run(scenario, func(t *testing.T) {
			db := testutils.NewConcurrentDatabaseForTest(t)
			store := NewStore(db, nil)
			require.NoError(t, db.Create(&model.OidcClient{Base: model.Base{ID: clientID}, Name: "Grace client", IsPublic: true}).Error)
			require.NoError(t, db.Create(&model.User{Base: model.Base{ID: userID}, Username: "grace"}).Error)
			provider, err := newProvider(store, nil, testTokenSigner{key: key}, Config{BaseURL: issuer, TokenBaseURL: issuer, Secret: []byte(secret)}, nil)
			require.NoError(t, err)
			handler := newTokenHandler(provider, newClaimsService(db, nil, issuer, nil), nil)

			// Seed an old grant so the grace period must start at rotation rather than token issuance
			request := newTestRequester(requestID, clientID, userID, "grace-jti")
			request.GetSession().SetExpiresAt(fosite.RefreshToken, time.Now().UTC().Add(time.Hour))
			token, signature, err := strategy.GenerateRefreshToken(t.Context(), request)
			require.NoError(t, err)
			require.NoError(t, store.CreateRefreshTokenSession(t.Context(), signature, "original-access", request))
			require.NoError(t, store.CreateAccessTokenSession(t.Context(), "original-access", request))
			require.NoError(t, db.Model(&OAuth2Session{}).Where("kind = ? AND key = ?", sessionKindRefreshToken, signature).
				Update("created_at", datatype.DateTime(time.Now().UTC().Add(-24*time.Hour))).Error)

			// Exercise the real token endpoint, including authentication, user checks, rotation, and transactions
			refresh := func(clientID, token string) *httptest.ResponseRecorder {
				form := url.Values{"grant_type": {"refresh_token"}, "refresh_token": {token}, "client_id": {clientID}}
				req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/oidc/token", strings.NewReader(form.Encode()))
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
				rec := httptest.NewRecorder()
				c, _ := gin.CreateTestContext(rec)
				c.Request = req
				handler.token(c)
				return rec
			}
			body := func(rec *httptest.ResponseRecorder, status int) map[string]any {
				t.Helper()
				require.Equal(t, status, rec.Code, rec.Body.String())
				var result map[string]any
				require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &result))
				return result
			}

			var first, second map[string]any
			if scenario == "concurrent" {
				start := make(chan struct{})
				responses := make(chan *httptest.ResponseRecorder, 2)
				for range 2 {
					go func() {
						<-start
						responses <- refresh(clientID, token)
					}()
				}
				close(start)
				first = body(<-responses, http.StatusOK)
				second = body(<-responses, http.StatusOK)
			} else {
				first = body(refresh(clientID, token), http.StatusOK)
			}
			if scenario == "retry" {
				second = body(refresh(clientID, token), http.StatusOK)
			}

			rotated, err := store.getSession(t.Context(), sessionKindRefreshToken, signature)
			require.NoError(t, err)
			require.False(t, rotated.Active)
			require.NotNil(t, rotated.RotatedAt)
			_, err = store.GetAccessTokenSession(t.Context(), "original-access", nil)
			require.ErrorIs(t, err, fosite.ErrNotFound)

			switch scenario {
			case "retry", "concurrent":
				require.NotEqual(t, first["refresh_token"], second["refresh_token"])
				afterRetry, err := store.getSession(t.Context(), sessionKindRefreshToken, signature)
				require.NoError(t, err)
				require.Equal(t, rotated.RotatedAt, afterRetry.RotatedAt)
				for _, issued := range []map[string]any{first, second} {
					_, err := store.GetAccessTokenSession(t.Context(), provider.accessToken.AccessTokenSignature(t.Context(), issued["access_token"].(string)), nil)
					require.NoError(t, err)
					body(refresh(clientID, issued["refresh_token"].(string)), http.StatusOK)
				}
			case "reuse after grace":
				second = body(refresh(clientID, token), http.StatusOK)
				require.NoError(t, db.Model(&OAuth2Session{}).Where("kind = ? AND key = ?", sessionKindRefreshToken, signature).
					Update("rotated_at", datatype.DateTime(time.Now().UTC().Add(-refreshTokenGracePeriod-time.Second))).Error)
				require.ErrorIs(t, store.RotateRefreshToken(t.Context(), requestID, signature), fosite.ErrInactiveToken)
				require.Equal(t, "invalid_grant", body(refresh(clientID, token), http.StatusBadRequest)["error"])
				for _, issued := range []map[string]any{first, second} {
					_, err := store.GetAccessTokenSession(t.Context(), provider.accessToken.AccessTokenSignature(t.Context(), issued["access_token"].(string)), nil)
					require.ErrorIs(t, err, fosite.ErrNotFound)
					require.Equal(t, "invalid_grant", body(refresh(clientID, issued["refresh_token"].(string)), http.StatusBadRequest)["error"])
				}
			case "expired token":
				request.GetSession().SetExpiresAt(fosite.RefreshToken, time.Now().UTC().Add(-time.Hour))
				data, err := store.encodeRequester(request)
				require.NoError(t, err)
				require.NoError(t, db.Model(&OAuth2Session{}).Where("kind = ? AND key = ?", sessionKindRefreshToken, signature).Update("request_data", data).Error)
				require.Equal(t, "invalid_grant", body(refresh(clientID, token), http.StatusBadRequest)["error"])
				body(refresh(clientID, first["refresh_token"].(string)), http.StatusOK)
			case "wrong client":
				require.NoError(t, db.Create(&model.OidcClient{Base: model.Base{ID: "other-client"}, Name: "Other", IsPublic: true}).Error)
				require.Equal(t, "invalid_grant", body(refresh("other-client", token), http.StatusBadRequest)["error"])
				body(refresh(clientID, first["refresh_token"].(string)), http.StatusOK)
			case "disabled user":
				require.NoError(t, db.Model(&model.User{}).Where("id = ?", userID).Update("disabled", true).Error)
				require.Equal(t, "invalid_grant", body(refresh(clientID, token), http.StatusBadRequest)["error"])
			default:
				// Revoking the grant must cancel grace for ancestors as well as their descendants
				switch scenario {
				case "token revocation":
					require.NoError(t, store.RevokeRefreshToken(t.Context(), requestID))
				case "logout":
					require.NoError(t, store.RevokeSessionsByIDTokenHint(t.Context(), userID, clientID, "grace-jti"))
				case "app revocation":
					require.NoError(t, RevokeUserClientSessions(t.Context(), db, userID, clientID))
				}
				require.ErrorIs(t, store.RotateRefreshToken(t.Context(), requestID, signature), fosite.ErrInactiveToken)
				require.Equal(t, "invalid_grant", body(refresh(clientID, token), http.StatusBadRequest)["error"])
				require.Equal(t, "invalid_grant", body(refresh(clientID, first["refresh_token"].(string)), http.StatusBadRequest)["error"])
			}
		})
	}
}

func TestRefreshTokenGraceMigrationPreservesExistingSessions(t *testing.T) {
	const previousVersion = 20260923183637
	db := testutils.NewDatabaseForTestWithMigrationSeed(t, previousVersion, func(t *testing.T, db *gorm.DB) {
		// The claim mapping policy column doesn't exist yet at this schema version
		require.NoError(t, db.Omit("ClaimMappingPolicyId").Create(&model.OidcClient{Base: model.Base{ID: "old-client"}, Name: "Old client"}).Error)
		require.NoError(t, db.Exec(`INSERT INTO oauth2_sessions (id, created_at, kind, key, request_id, client_id, active, request_data) VALUES ('old-refresh', 1, 'refresh_token', 'old-signature', 'old-request', 'old-client', false, '{}')`).Error)
	})
	var session OAuth2Session
	require.NoError(t, db.First(&session, "id = ?", "old-refresh").Error)
	require.False(t, session.Active)
	require.Nil(t, session.RotatedAt)
	down, err := resources.FS.ReadFile("migrations/sqlite/20260929120000_refresh_token_rotation_grace.down.sql")
	require.NoError(t, err)
	require.NoError(t, db.Exec(string(down)).Error)
	var count int64
	require.NoError(t, db.Table("oauth2_sessions").Where("id = ?", "old-refresh").Count(&count).Error)
	require.EqualValues(t, 1, count)
}
