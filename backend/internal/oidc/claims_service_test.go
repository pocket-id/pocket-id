package oidc

import (
	"context"
	"testing"

	"github.com/lestrrat-go/jwx/v3/jwa"
	"github.com/ory/fosite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/model"
	testutils "github.com/pocket-id/pocket-id/backend/internal/utils/testing"
)

// TestClaimsServiceValidateUserAccess covers the per-grant re-validation that the token
// endpoint performs on every grant (notably refresh_token, which fosite replays without
// reloading the user). A disabled user, a user removed from a group-restricted client, or
// a deleted user must be rejected so they cannot keep minting tokens from a still-valid
// refresh token.
func TestClaimsServiceValidateUserAccess(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	claimsService := newClaimsService(db, nil, "", nil)

	group := model.UserGroup{Base: model.Base{ID: "group-allowed"}, Name: "allowed", FriendlyName: "Allowed"}
	require.NoError(t, db.Create(&group).Error)

	enabledUser := model.User{Base: model.Base{ID: "user-enabled"}, Username: "enabled"}
	require.NoError(t, db.Create(&enabledUser).Error)
	require.NoError(t, db.Model(&enabledUser).Association("UserGroups").Append(&group))

	disabledUser := model.User{Base: model.Base{ID: "user-disabled"}, Username: "disabled", Disabled: true}
	require.NoError(t, db.Create(&disabledUser).Error)

	outsiderUser := model.User{Base: model.Base{ID: "user-outsider"}, Username: "outsider"}
	require.NoError(t, db.Create(&outsiderUser).Error)

	openClient := Client{OidcClient: model.OidcClient{Base: model.Base{ID: "client-open"}, Name: "Open"}}
	restrictedClient := Client{OidcClient: model.OidcClient{
		Base:              model.Base{ID: "client-restricted"},
		Name:              "Restricted",
		IsGroupRestricted: true,
		AllowedUserGroups: []model.UserGroup{group},
	}}

	t.Run("empty subject is allowed (client_credentials)", func(t *testing.T) {
		require.NoError(t, claimsService.ValidateUserAccess(t.Context(), "", openClient))
	})

	t.Run("enabled user is allowed", func(t *testing.T) {
		require.NoError(t, claimsService.ValidateUserAccess(t.Context(), enabledUser.ID, openClient))
	})

	t.Run("disabled user is rejected with invalid_grant", func(t *testing.T) {
		err := claimsService.ValidateUserAccess(t.Context(), disabledUser.ID, openClient)
		require.ErrorIs(t, err, fosite.ErrInvalidGrant)
	})

	t.Run("user in an allowed group may use a group-restricted client", func(t *testing.T) {
		require.NoError(t, claimsService.ValidateUserAccess(t.Context(), enabledUser.ID, restrictedClient))
	})

	t.Run("user outside the allowed groups is rejected with access_denied", func(t *testing.T) {
		err := claimsService.ValidateUserAccess(t.Context(), outsiderUser.ID, restrictedClient)
		require.ErrorIs(t, err, fosite.ErrAccessDenied)
	})

	t.Run("deleted user is rejected with invalid_grant", func(t *testing.T) {
		err := claimsService.ValidateUserAccess(t.Context(), "does-not-exist", openClient)
		require.ErrorIs(t, err, fosite.ErrInvalidGrant)
	})
}

type fakeCustomClaimSource struct {
	claims []model.CustomClaim
}

func (f fakeCustomClaimSource) GetCustomClaimsForUserWithUserGroups(_ context.Context, _ string, _ *gorm.DB) ([]model.CustomClaim, error) {
	return f.claims, nil
}

// TestClaimsServiceGetUserClaims pins the scope-to-claims mapping that powers both the ID
// token and the userinfo endpoint: each OIDC scope must only release its own claims, "sub"
// is always present, and custom claims are emitted as parsed JSON when the stored value is
// valid JSON and as a raw string otherwise.
func TestClaimsServiceGetUserClaims(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	const (
		baseURL = "https://id.example.com"
		userID  = "user-1"
	)

	customClaims := fakeCustomClaimSource{claims: []model.CustomClaim{
		{Key: "department", Value: "engineering"}, // plain string
		{Key: "roles", Value: `["admin","dev"]`},  // JSON document
	}}
	service := newClaimsService(db, customClaims, baseURL, nil)

	group := model.UserGroup{Base: model.Base{ID: "group-1"}, Name: "developers", FriendlyName: "Developers"}
	require.NoError(t, db.Create(&group).Error)

	user := model.User{
		Base:          model.Base{ID: userID},
		Username:      "tim",
		FirstName:     "Tim",
		LastName:      "Cook",
		DisplayName:   "Tim Cook",
		Email:         stringPointer("tim@example.com"),
		EmailVerified: true,
	}
	require.NoError(t, db.Create(&user).Error)
	require.NoError(t, db.Model(&user).Association("UserGroups").Append(&group))

	t.Run("openid only releases sub", func(t *testing.T) {
		claims, err := service.GetUserClaims(t.Context(), userID, []string{"openid"})
		require.NoError(t, err)
		require.Equal(t, map[string]any{"sub": userID}, claims)
	})

	t.Run("email scope releases email claims", func(t *testing.T) {
		claims, err := service.GetUserClaims(t.Context(), userID, []string{"openid", "email"})
		require.NoError(t, err)
		require.Equal(t, userID, claims["sub"])
		require.Equal(t, "tim@example.com", claims["email"])
		require.Equal(t, true, claims["email_verified"])
		require.NotContains(t, claims, "given_name")
		require.NotContains(t, claims, "groups")
	})

	t.Run("groups scope releases group names", func(t *testing.T) {
		claims, err := service.GetUserClaims(t.Context(), userID, []string{"groups"})
		require.NoError(t, err)
		require.Equal(t, []string{"developers"}, claims["groups"])
	})

	t.Run("profile scope releases profile and custom claims", func(t *testing.T) {
		claims, err := service.GetUserClaims(t.Context(), userID, []string{"profile"})
		require.NoError(t, err)
		require.Equal(t, "Tim", claims["given_name"])
		require.Equal(t, "Cook", claims["family_name"])
		require.Equal(t, "Tim Cook", claims["name"])
		require.Equal(t, "Tim Cook", claims["display_name"])
		require.Equal(t, "tim", claims["preferred_username"])
		require.Equal(t, baseURL+"/api/users/"+userID+"/profile-picture.png", claims["picture"])

		// Custom claims: plain string stays a string, JSON document is decoded.
		require.Equal(t, "engineering", claims["department"])
		require.Equal(t, []any{"admin", "dev"}, claims["roles"])

		// Profile must not leak email when the email scope was not requested.
		require.NotContains(t, claims, "email")
	})
}

// TestClaimsServiceAppliesSigningAlgToIDTokenHeader verifies the ID token header carries the
// signing algorithm so fosite derives the at_hash/c_hash digest from it (e.g. RS384 ->
// SHA-384, ES512 -> SHA-512) instead of always defaulting to SHA-256.
func TestClaimsServiceAppliesSigningAlgToIDTokenHeader(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	require.NoError(t, db.Create(&model.User{Base: model.Base{ID: "alg-user"}, Username: "alg"}).Error)

	for _, alg := range []jwa.SignatureAlgorithm{jwa.RS256(), jwa.RS384(), jwa.ES512()} {
		t.Run(alg.String(), func(t *testing.T) {
			service := newClaimsService(db, nil, "", algTestSigner{alg: alg})

			session := NewEmptySession()
			session.Subject = "alg-user"

			require.NoError(t, service.applyIDTokenClaims(t.Context(), session, fosite.Arguments{"openid"}))
			require.Equal(t, alg.String(), session.IDTokenHeaders().Get("alg"))
		})
	}
}
