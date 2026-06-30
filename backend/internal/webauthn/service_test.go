package webauthn

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/lestrrat-go/jwx/v3/jwt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
	testutils "github.com/pocket-id/pocket-id/backend/internal/utils/testing"
)

// fakeSigner is an in-memory TokenService that mints opaque tokens carrying a subject,
// an issued-at time and an optional authentication method, without any real signing
type fakeSigner struct {
	tokens  map[string]jwt.Token
	counter int
}

func newFakeSigner() *fakeSigner {
	return &fakeSigner{tokens: map[string]jwt.Token{}}
}

func (s *fakeSigner) GenerateAccessToken(user model.User, authenticationMethod string) (string, error) {
	builder := jwt.NewBuilder().
		Subject(user.ID).
		IssuedAt(time.Now())
	if authenticationMethod != "" {
		builder = builder.Claim(common.AuthenticationMethodsClaim, []string{authenticationMethod})
	}
	token, err := builder.Build()
	if err != nil {
		return "", err
	}

	s.counter++
	raw := fmt.Sprintf("fake-access-token-%d", s.counter)
	s.tokens[raw] = token
	return raw, nil
}

func (s *fakeSigner) VerifyAccessToken(tokenString string) (jwt.Token, error) {
	token, ok := s.tokens[tokenString]
	if !ok {
		return nil, errors.New("invalid token")
	}
	return token, nil
}

func (s *fakeSigner) GetAuthenticationMethod(token jwt.Token) (string, error) {
	if !token.Has(common.AuthenticationMethodsClaim) {
		return "", nil
	}
	var methods []string
	if err := token.Get(common.AuthenticationMethodsClaim, &methods); err != nil {
		return "", err
	}
	if len(methods) == 0 {
		return "", nil
	}
	return methods[0], nil
}

func TestCreateReauthenticationTokenWithAccessToken(t *testing.T) {
	setupService := func(t *testing.T) (*Service, *fakeSigner, model.User) {
		t.Helper()

		db := testutils.NewDatabaseForTest(t)
		user := model.User{
			Base:     model.Base{ID: "reauth-user"},
			Username: "reauth-user",
		}
		require.NoError(t, db.Create(&user).Error)

		signer := newFakeSigner()
		return &Service{db: db, signer: signer}, signer, user
	}

	t.Run("accepts a fresh access token from WebAuthn login", func(t *testing.T) {
		service, signer, user := setupService(t)
		accessToken, err := signer.GenerateAccessToken(user, authenticationMethodPhishingResistant)
		require.NoError(t, err)

		reauthenticationToken, err := service.CreateReauthenticationTokenWithAccessToken(t.Context(), accessToken)

		require.NoError(t, err)
		assert.NotEmpty(t, reauthenticationToken)
	})

	t.Run("rejects a fresh access token from one-time access login", func(t *testing.T) {
		service, signer, user := setupService(t)
		accessToken, err := signer.GenerateAccessToken(user, "otp")
		require.NoError(t, err)

		reauthenticationToken, err := service.CreateReauthenticationTokenWithAccessToken(t.Context(), accessToken)

		assert.Empty(t, reauthenticationToken)
		require.Error(t, err)
		assert.ErrorAs(t, err, new(*common.ReauthenticationRequiredError))
	})

	t.Run("rejects a fresh access token without an authentication method", func(t *testing.T) {
		service, signer, user := setupService(t)
		accessToken, err := signer.GenerateAccessToken(user, "")
		require.NoError(t, err)

		reauthenticationToken, err := service.CreateReauthenticationTokenWithAccessToken(t.Context(), accessToken)

		assert.Empty(t, reauthenticationToken)
		require.Error(t, err)
		assert.ErrorAs(t, err, new(*common.ReauthenticationRequiredError))
	})
}

func TestConsumeReauthenticationTokenReturnsTokenCreationTime(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	service := &Service{db: db}

	const (
		userID = "reauth-user"
		token  = "reauthentication-token"
	)
	require.NoError(t, db.Create(&model.User{
		Base: model.Base{ID: userID},
	}).Error)
	require.NoError(t, db.Create(&ReauthenticationToken{
		Token:     utils.CreateSha256Hash(token),
		ExpiresAt: datatype.DateTime(time.Now().Add(time.Minute)),
		UserID:    userID,
	}).Error)

	var storedToken ReauthenticationToken
	require.NoError(t, db.First(&storedToken, "user_id = ?", userID).Error)

	tx := db.Begin()
	reauthenticatedAt, err := service.ConsumeReauthenticationToken(t.Context(), tx, token, userID)
	require.NoError(t, err)
	require.NoError(t, tx.Commit().Error)

	require.Equal(t, storedToken.CreatedAt.UTC(), reauthenticatedAt)
}
