//go:build unit

package email

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/pocket-id/pocket-id/backend/internal/appconfig"
	"github.com/pocket-id/pocket-id/backend/internal/authz"
	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/middleware"
	testutils "github.com/pocket-id/pocket-id/backend/internal/utils/testing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type signingTestAuthenticator struct{}

func (signingTestAuthenticator) Kind() authz.PrincipalKind   { return authz.KindSession }
func (signingTestAuthenticator) Present(c *gin.Context) bool { return c.GetHeader("X-Test-Role") != "" }
func (signingTestAuthenticator) Authenticate(c *gin.Context) (*authz.Principal, error) {
	return &authz.Principal{Kind: authz.KindSession, UserID: "test", Scopes: authz.UserScopes(c.GetHeader("X-Test-Role") == "admin", authz.KindSession)}, nil
}

func TestSigningRoutesAuthorizationUploadAndRemoval(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := testutils.NewDatabaseForTest(t)
	module, err := New(db)
	require.NoError(t, err)
	router := gin.New()
	router.Use(middleware.NewErrorHandlerMiddleware().Add())
	module.RegisterSigningRoutes(authz.NewMiddleware(signingTestAuthenticator{}).Router(router.Group("/api")), appconfig.StaticResolver{Config: &appconfig.AppConfigModel{SmtpFrom: "sender@example.test"}})
	request := func(method, path, role, contentType string, body io.Reader) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequestWithContext(t.Context(), method, "/api/application-configuration/smime"+path, body)
		req.Header.Set("X-Test-Role", role)
		if contentType != "" {
			req.Header.Set("Content-Type", contentType)
		}
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, req)
		return recorder
	}
	for _, route := range []struct{ method, path string }{{"GET", ""}, {"PUT", ""}, {"PUT", "/certificate"}, {"DELETE", "/certificate"}} {
		assert.Equal(t, 401, request(route.method, route.path, "", "", nil).Code)
		assert.Equal(t, 403, request(route.method, route.path, "user", "", nil).Code)
	}
	assert.Equal(t, 200, request("GET", "", "admin", "", nil).Code)
	assert.Equal(t, 400, request("PUT", "", "admin", "application/json", strings.NewReader(`{}`)).Code)
	assert.Equal(t, 400, request("PUT", "", "admin", "application/json", strings.NewReader(`{"enabled":true}`)).Code)
	_, _, pfx := signingFixture(t, "RSA")
	multipartUpload := func(data []byte, password string) *httptest.ResponseRecorder {
		var body bytes.Buffer
		writer := multipart.NewWriter(&body)
		file, err := writer.CreateFormFile("file", "certificate.p12")
		require.NoError(t, err)
		_, err = file.Write(data)
		require.NoError(t, err)
		require.NoError(t, writer.WriteField("password", password))
		require.NoError(t, writer.Close())
		return request("PUT", "/certificate", "admin", writer.FormDataContentType(), &body)
	}
	assert.Equal(t, 400, multipartUpload(make([]byte, maxSigningUpload), "").Code)
	assert.Equal(t, 400, multipartUpload(pfx, "incorrect").Code)
	uploaded := multipartUpload(pfx, " password ")
	require.Equal(t, 200, uploaded.Code, uploaded.Body.String())
	assert.Contains(t, uploaded.Body.String(), "sender@example.test")
	assert.NotContains(t, uploaded.Body.String(), "privateKey")
	assert.NotContains(t, uploaded.Body.String(), " password ")
	enabled := request("PUT", "", "admin", "application/json", strings.NewReader(`{"enabled":true}`))
	require.Equal(t, 200, enabled.Code, enabled.Body.String())
	assert.Contains(t, enabled.Body.String(), `"enabled":true`)
	removed := request("DELETE", "/certificate", "admin", "", nil)
	require.Equal(t, 200, removed.Code)
	assert.JSONEq(t, `{"enabled":false,"certificate":null}`, removed.Body.String())
	// UI-disabled configuration must deny writes even to an authenticated administrator
	old := common.EnvConfig.UiConfigDisabled
	common.EnvConfig.UiConfigDisabled = true
	t.Cleanup(func() { common.EnvConfig.UiConfigDisabled = old })
	assert.Equal(t, 403, request("DELETE", "/certificate", "admin", "", nil).Code)
	assert.Equal(t, 403, request("PUT", "", "admin", "application/json", strings.NewReader(`{"enabled":false}`)).Code)
	assert.Equal(t, 403, multipartUpload(pfx, " password ").Code)
}
