package oidc

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pocket-id/pocket-id/backend/internal/model"
	testutils "github.com/pocket-id/pocket-id/backend/internal/utils/testing"
)

func TestClientPreviewBuilderUsesFositeTokenStrategies(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	signerKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	provider, err := newProvider(NewStore(db, nil), nil, testTokenSigner{key: signerKey}, Config{ //nolint:gosec // static test-only provider secret
		BaseURL:      "https://issuer.example.com",
		TokenBaseURL: "https://issuer.example.com",
		Secret:       []byte("test-secret"),
	})
	require.NoError(t, err)

	builder := newClientPreviewBuilder(newClaimsService(db, nil, "https://issuer.example.com", nil), provider.tokenStrategies)

	const (
		userID   = "test-user"
		clientID = "test-client"
	)
	email := "user@example.com"
	require.NoError(t, db.Create(&model.User{
		Base:          model.Base{ID: userID},
		Username:      "test-user",
		Email:         &email,
		EmailVerified: true,
	}).Error)

	preview, err := builder.BuildClientPreview(t.Context(), model.OidcClient{
		Base: model.Base{ID: clientID},
		Name: "Test Client",
	}, userID, []string{"openid", "email"}, "phr")
	require.NoError(t, err)

	require.Equal(t, "https://issuer.example.com", preview.AccessToken["iss"])
	require.ElementsMatch(t, []string{"openid", "email"}, stringSliceClaim(t, preview.AccessToken["scp"]))
	// The identity scopes add the issuer to the audience so the previewed token would also work at /userinfo
	require.ElementsMatch(t, []string{clientID, "https://issuer.example.com"}, stringSliceClaim(t, preview.AccessToken["aud"]))
	require.NotContains(t, preview.AccessToken, "type")

	require.Equal(t, userID, preview.IDToken["sub"])
	// ID tokens carry the "type" marker (so the end-session endpoint can reject access tokens
	// passed as id_token_hint) and the amr from the authentication method.
	require.Equal(t, idTokenType, preview.IDToken["type"])
	require.ElementsMatch(t, []string{"phr"}, stringSliceClaim(t, preview.IDToken["amr"]))

	require.Equal(t, email, preview.UserInfo["email"])
	require.Equal(t, true, preview.UserInfo["email_verified"])
}

func TestClientPreviewBuilderRejectsInvalidScope(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	signerKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	provider, err := newProvider(NewStore(db, nil), nil, testTokenSigner{key: signerKey}, Config{ //nolint:gosec // static test-only provider secret
		BaseURL:      "https://issuer.example.com",
		TokenBaseURL: "https://issuer.example.com",
		Secret:       []byte("test-secret"),
	})
	require.NoError(t, err)

	builder := newClientPreviewBuilder(newClaimsService(db, nil, "https://issuer.example.com", nil), provider.tokenStrategies)
	_, err = builder.BuildClientPreview(t.Context(), model.OidcClient{
		Base: model.Base{ID: "test-client"},
		Name: "Test Client",
	}, "test-user", []string{"openid", "unknown"}, "")
	require.Error(t, err)
	require.ErrorContains(t, err, "invalid_scope")
}

func stringSliceClaim(t *testing.T, value any) []string {
	t.Helper()

	switch typed := value.(type) {
	case []string:
		return typed
	case []any:
		values := make([]string, 0, len(typed))
		for _, item := range typed {
			value, ok := item.(string)
			require.Truef(t, ok, "expected string claim item, got %T", item)
			values = append(values, value)
		}
		return values
	case string:
		return []string{typed}
	default:
		require.Failf(t, "unexpected claim type", "expected string slice claim, got %T", value)
		return nil
	}
}
