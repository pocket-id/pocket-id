package oidc

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/base64"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
	testutils "github.com/pocket-id/pocket-id/backend/internal/utils/testing"
	"github.com/stretchr/testify/require"
)

type authorizationPARTestCase struct {
	name          string
	method        string
	queryURI      string
	bodyURI       []string
	createPAR     bool
	wantCode      bool
	multipart     bool
	requestObject bool
	malformed     bool
}

func TestAuthorizationHandlerRequiresPAR(t *testing.T) {
	gin.SetMode(gin.TestMode)
	const fakeURI = parRequestURIPrefix + "missing"
	for _, clientType := range []string{"public", "confidential"} {
		t.Run(clientType, func(t *testing.T) {
			for _, tt := range []authorizationPARTestCase{
				{name: "missing request URI", method: http.MethodPost},
				{name: "unknown PAR URI", method: http.MethodGet, queryURI: fakeURI},
				{name: "empty body shadows PAR query", method: http.MethodPost, queryURI: fakeURI, bodyURI: []string{""}},
				{name: "non-PAR body shadows PAR query", method: http.MethodPost, queryURI: fakeURI, bodyURI: []string{"not-a-par-uri"}},
				{name: "valid PAR in query", method: http.MethodGet, queryURI: "stored", createPAR: true, wantCode: true},
				{name: "valid PAR in body", method: http.MethodPost, bodyURI: []string{"stored"}, createPAR: true, wantCode: true},
				{name: "valid body overrides invalid query", method: http.MethodPost, queryURI: fakeURI, bodyURI: []string{"stored"}, createPAR: true, wantCode: true},
				{name: "empty body shadows stored PAR", method: http.MethodPost, queryURI: "stored", bodyURI: []string{""}, createPAR: true},
				{name: "valid PAR in multipart body", method: http.MethodPost, bodyURI: []string{"stored"}, createPAR: true, wantCode: true, multipart: true},
				{name: "request object cannot introduce PAR URI", method: http.MethodPost, requestObject: true},
				{name: "malformed body", method: http.MethodPost, queryURI: fakeURI, malformed: true},
			} {
				t.Run(tt.name, func(t *testing.T) {
					testAuthorizationHandlerPAR(t, clientType, tt)
				})
			}
		})
	}
}

func testAuthorizationHandlerPAR(t *testing.T, clientType string, tt authorizationPARTestCase) {
	t.Helper()
	const (
		baseURL      = "https://issuer.example.com"
		callbackURL  = "https://client.example.com/callback"
		clientID     = "par-client"
		userID       = "test-user"
		clientSecret = "test-client-secret"
		fakeURI      = parRequestURIPrefix + "missing"
	)
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	// Use the real provider and an authenticated user so a bypass would issue an authorization code
	db := testutils.NewDatabaseForTest(t)
	require.NoError(t, db.Create(&model.User{Base: model.Base{ID: userID}}).Error)
	require.NoError(t, db.Create(&model.OidcClient{
		Base:                                model.Base{ID: clientID},
		Name:                                "PAR client",
		CallbackURLs:                        datatype.StringList{callbackURL},
		IsPublic:                            clientType == "public",
		PkceEnabled:                         true,
		SkipConsent:                         true,
		RequiresPushedAuthorizationRequests: true,
		Credentials:                         testClientCredentials(clientSecret),
	}).Error)
	store := NewStore(db, nil)
	provider, err := newProvider(store, nil, testTokenSigner{key: key}, Config{
		BaseURL: baseURL, TokenBaseURL: baseURL, Secret: []byte("test-secret"),
	}, nil)
	require.NoError(t, err)
	service := newAuthorizationService(db, newInteractionSessionService(db), newClaimsService(db, nil, baseURL, nil), nil, nil, nil)
	handler := newAuthorizationHandler(provider, service)
	params := url.Values{
		"client_id":             {clientID},
		"response_type":         {"code"},
		"redirect_uri":          {callbackURL},
		"scope":                 {"openid"},
		"state":                 {"state-with-enough-entropy"},
		"prompt":                {"none"},
		"code_challenge":        {"E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM"},
		"code_challenge_method": {"S256"},
	}
	if tt.requestObject {
		header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none"}`))
		payload := base64.RawURLEncoding.EncodeToString([]byte(`{"request_uri":"` + fakeURI + `"}`))
		params.Set("request", header+"."+payload+".")
	}

	// Create valid PAR sessions through Fosite to cover both accepted transports and consumption
	var storedURI string
	if tt.createPAR {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/oidc/par", strings.NewReader(params.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		if clientType == "confidential" {
			req.SetBasicAuth(clientID, clientSecret)
		}
		ar, err := provider.NewPushedAuthorizeRequest(t.Context(), req)
		require.NoError(t, err)
		response, err := provider.NewPushedAuthorizeResponse(t.Context(), ar, NewEmptySession())
		require.NoError(t, err)
		storedURI = response.GetRequestURI()
	}

	// Keep query and body parameters separate to exercise the same precedence as incoming HTTP requests
	query := url.Values{}
	if tt.queryURI != "" {
		uri := tt.queryURI
		if uri == "stored" {
			uri = storedURI
		}
		query.Set("request_uri", uri)
	}
	for _, uri := range tt.bodyURI {
		if uri == "stored" {
			uri = storedURI
		}
		params.Add("request_uri", uri)
	}
	body := ""
	contentType := "application/x-www-form-urlencoded"
	if tt.method == http.MethodPost {
		body = params.Encode()
	} else {
		for k, values := range params {
			query[k] = values
		}
	}
	if tt.multipart {
		var buf bytes.Buffer
		writer := multipart.NewWriter(&buf)
		for k, values := range params {
			for _, value := range values {
				require.NoError(t, writer.WriteField(k, value))
			}
		}
		require.NoError(t, writer.Close())
		body = buf.String()
		contentType = writer.FormDataContentType()
	}
	if tt.malformed {
		body = "request_uri=%zz"
	}
	req := httptest.NewRequestWithContext(t.Context(), tt.method, "/authorize?"+query.Encode(), strings.NewReader(body))
	req.Header.Set("Content-Type", contentType)
	rec := httptest.NewRecorder()
	router := gin.New()
	router.Handle(tt.method, "/authorize", func(c *gin.Context) {
		c.Set("userID", userID)
		c.Set("authenticationTime", time.Now().UTC().Add(-time.Minute))
		handler.authorize(c)
	})
	router.ServeHTTP(rec, req)

	// Rejections must neither return nor persist a code, while valid PAR must be consumed
	location, err := url.Parse(rec.Header().Get("Location"))
	require.NoError(t, err)
	var codeCount int64
	require.NoError(t, db.Model(&OAuth2Session{}).Where("kind = ?", sessionKindAuthorizeCode).Count(&codeCount).Error)
	if tt.wantCode {
		require.Equal(t, http.StatusSeeOther, rec.Code)
		require.NotEmpty(t, location.Query().Get("code"), rec.Body.String())
		require.Empty(t, location.Query().Get("error"))
		require.EqualValues(t, 1, codeCount)
	} else {
		require.Empty(t, location.Query().Get("code"))
		require.NotEmpty(t, location.Query().Get("error"))
		require.Zero(t, codeCount)
	}
	if tt.createPAR {
		var parCount int64
		require.NoError(t, db.Model(&OAuth2Session{}).Where("kind = ? AND key = ? AND active = ?", sessionKindPAR, storedURI, true).Count(&parCount).Error)
		if tt.wantCode {
			require.Zero(t, parCount)
		} else {
			require.EqualValues(t, 1, parCount)
		}
	}
}
