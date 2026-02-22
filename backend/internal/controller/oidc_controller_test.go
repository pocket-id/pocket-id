package controller

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/dto"
	"github.com/pocket-id/pocket-id/backend/internal/service"
)

func TestCreateTokensHandler(t *testing.T) {
	createTestContext := func(t *testing.T, rawURL string, form url.Values, authHeader string, noCT bool) (*gin.Context, *httptest.ResponseRecorder) {
		t.Helper()

		mode := gin.Mode()
		gin.SetMode(gin.TestMode)
		t.Cleanup(func() { gin.SetMode(mode) })

		recorder := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(recorder)

		req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, rawURL, strings.NewReader(form.Encode()))
		require.NoError(t, err)

		if !noCT {
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		}
		if authHeader != "" {
			req.Header.Set("Authorization", authHeader)
		}

		c.Request = req
		return c, recorder
	}

	t.Run("Ignores Query String Parameters For Binding", func(t *testing.T) {
		oc := &OidcController{}

		c, _ := createTestContext(
			t,
			"http://example.com/oidc/token?grant_type=refresh_token&refresh_token=query-value",
			url.Values{},
			"",
			false,
		)

		oc.createTokensHandler(c)

		require.Len(t, c.Errors, 1)
		assert.Contains(t, c.Errors[0].Err.Error(), "GrantType")
	})

	t.Run("Missing Authorization Code", func(t *testing.T) {
		oc := &OidcController{}

		c, _ := createTestContext(
			t,
			"http://example.com/oidc/token",
			url.Values{
				"grant_type": {service.GrantTypeAuthorizationCode},
			},
			"",
			false,
		)

		oc.createTokensHandler(c)

		require.Len(t, c.Errors, 1)
		var missingCodeErr *common.OidcMissingAuthorizationCodeError
		require.ErrorAs(t, c.Errors[0].Err, &missingCodeErr)
	})

	t.Run("Missing Refresh Token", func(t *testing.T) {
		oc := &OidcController{}

		c, _ := createTestContext(
			t,
			"http://example.com/oidc/token",
			url.Values{
				"grant_type": {service.GrantTypeRefreshToken},
			},
			"",
			false,
		)

		oc.createTokensHandler(c)

		require.Len(t, c.Errors, 1)
		var missingRefreshErr *common.OidcMissingRefreshTokenError
		require.ErrorAs(t, c.Errors[0].Err, &missingRefreshErr)
	})

	t.Run("Uses Basic Auth Credentials When Body Credentials Missing", func(t *testing.T) {
		var capturedInput dto.OidcCreateTokensDto
		oc := &OidcController{
			createTokens: func(_ context.Context, input dto.OidcCreateTokensDto) (service.CreatedTokens, error) {
				capturedInput = input
				return service.CreatedTokens{
					AccessToken:  "access-token",
					IdToken:      "id-token",
					RefreshToken: "refresh-token",
					ExpiresIn:    2 * time.Minute,
				}, nil
			},
		}

		basicAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte("client-id:client-secret"))
		c, recorder := createTestContext(
			t,
			"http://example.com/oidc/token",
			url.Values{
				"grant_type":    {service.GrantTypeRefreshToken},
				"refresh_token": {"input-refresh-token"},
			},
			basicAuth,
			false,
		)

		oc.createTokensHandler(c)

		require.Empty(t, c.Errors)
		assert.Equal(t, "client-id", capturedInput.ClientID)
		assert.Equal(t, "client-secret", capturedInput.ClientSecret)
		assert.Equal(t, "input-refresh-token", capturedInput.RefreshToken)

		require.Equal(t, http.StatusOK, recorder.Code)
		var response dto.OidcTokenResponseDto
		require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
		assert.Equal(t, "access-token", response.AccessToken)
		assert.Equal(t, "Bearer", response.TokenType)
		assert.Equal(t, "id-token", response.IdToken)
		assert.Equal(t, "refresh-token", response.RefreshToken)
		assert.Equal(t, 120, response.ExpiresIn)
	})

	t.Run("Maps Authorization Pending Error", func(t *testing.T) {
		oc := &OidcController{
			createTokens: func(context.Context, dto.OidcCreateTokensDto) (service.CreatedTokens, error) {
				return service.CreatedTokens{}, &common.OidcAuthorizationPendingError{}
			},
		}

		c, recorder := createTestContext(
			t,
			"http://example.com/oidc/token",
			url.Values{
				"grant_type":    {service.GrantTypeRefreshToken},
				"refresh_token": {"input-refresh-token"},
			},
			"",
			false,
		)

		oc.createTokensHandler(c)

		require.Empty(t, c.Errors)
		require.Equal(t, http.StatusBadRequest, recorder.Code)
		var response map[string]string
		require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
		assert.Equal(t, "authorization_pending", response["error"])
	})

	t.Run("Maps Slow Down Error", func(t *testing.T) {
		oc := &OidcController{
			createTokens: func(context.Context, dto.OidcCreateTokensDto) (service.CreatedTokens, error) {
				return service.CreatedTokens{}, &common.OidcSlowDownError{}
			},
		}

		c, recorder := createTestContext(
			t,
			"http://example.com/oidc/token",
			url.Values{
				"grant_type":    {service.GrantTypeRefreshToken},
				"refresh_token": {"input-refresh-token"},
			},
			"",
			false,
		)

		oc.createTokensHandler(c)

		require.Empty(t, c.Errors)
		require.Equal(t, http.StatusBadRequest, recorder.Code)
		var response map[string]string
		require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
		assert.Equal(t, "slow_down", response["error"])
	})

	t.Run("Returns Generic Service Error In Context", func(t *testing.T) {
		expectedErr := errors.New("boom")
		oc := &OidcController{
			createTokens: func(context.Context, dto.OidcCreateTokensDto) (service.CreatedTokens, error) {
				return service.CreatedTokens{}, expectedErr
			},
		}

		c, _ := createTestContext(
			t,
			"http://example.com/oidc/token",
			url.Values{
				"grant_type":    {service.GrantTypeRefreshToken},
				"refresh_token": {"input-refresh-token"},
			},
			"",
			false,
		)

		oc.createTokensHandler(c)

		require.Len(t, c.Errors, 1)
		assert.ErrorIs(t, c.Errors[0].Err, expectedErr)
	})
}
