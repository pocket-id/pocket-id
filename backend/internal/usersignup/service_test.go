package usersignup

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/appconfig"
	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/dto"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
	testutils "github.com/pocket-id/pocket-id/backend/internal/utils/testing"
)

type fakeUserCreator struct {
	err  error
	user model.User
}

func (f fakeUserCreator) CreateUserInternal(_ context.Context, _ *appconfig.AppConfigModel, _ dto.UserCreateDto, _ bool, _ *gorm.DB) (model.User, error) {
	if f.err != nil {
		return model.User{}, f.err
	}
	return f.user, nil
}

type fakeSigner struct{}

func (fakeSigner) GenerateAccessToken(_ model.User, _ string, _ time.Duration) (string, error) {
	return "access-token", nil
}

type fakeAuditLogger struct{}

func (fakeAuditLogger) Create(_ context.Context, _ model.AuditLogEvent, _, _, _ string, _ model.AuditLogData, _ *gorm.DB) (model.AuditLog, bool) {
	return model.AuditLog{}, true
}

func newSignupServiceForTest(t *testing.T, db *gorm.DB, userCreator UserCreator) *Service {
	t.Helper()
	actorService := newSignupTokenActorService(t, nil)
	return newService(Dependencies{
		DB:          db,
		UserCreator: userCreator,
		Signer:      fakeSigner{},
		AuditLog:    fakeAuditLogger{},
	}, actorService)
}

func signupTokenUsageCount(t *testing.T, svc *Service, tokenID string) int {
	t.Helper()
	tokens, _, err := svc.ListSignupTokens(t.Context(), listAllOptions())
	require.NoError(t, err)
	for _, tok := range tokens {
		if tok.ID == tokenID {
			return tok.UsageCount
		}
	}
	t.Fatalf("signup token %q not found", tokenID)
	return 0
}

func TestSignUpConsumesTokenOnSuccess(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	svc := newSignupServiceForTest(t, db, fakeUserCreator{user: model.User{Base: model.Base{ID: "new-user"}}})

	token, err := svc.CreateSignupToken(t.Context(), time.Hour, 2, nil)
	require.NoError(t, err)

	config := appconfig.NewTestConfig(nil)
	user, accessToken, err := svc.SignUp(t.Context(), config, signUpDto{
		Username: "newuser",
		Token:    token.Token,
	}, "1.2.3.4", "test-agent")
	require.NoError(t, err)
	require.Equal(t, "new-user", user.ID)
	require.Equal(t, "access-token", accessToken)

	// The token's usage count must have been incremented and not rolled back
	require.Equal(t, 1, signupTokenUsageCount(t, svc, token.ID))
}

func TestSignUpCompensatesTokenOnFailure(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	boom := errors.New("could not create user")
	svc := newSignupServiceForTest(t, db, fakeUserCreator{err: boom})

	token, err := svc.CreateSignupToken(t.Context(), time.Hour, 2, nil)
	require.NoError(t, err)

	config := appconfig.NewTestConfig(nil)
	_, _, err = svc.SignUp(t.Context(), config, signUpDto{
		Username: "newuser",
		Token:    token.Token,
	}, "1.2.3.4", "test-agent")
	require.ErrorIs(t, err, boom)

	// The usage count increment must have been compensated (reverted back to 0)
	require.Equal(t, 0, signupTokenUsageCount(t, svc, token.ID))
}

func TestSignUpRejectsInvalidToken(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	svc := newSignupServiceForTest(t, db, fakeUserCreator{user: model.User{Base: model.Base{ID: "new-user"}}})

	config := appconfig.NewTestConfig(nil)
	_, _, err := svc.SignUp(t.Context(), config, signUpDto{
		Username: "newuser",
		Token:    "not-a-real-token",
	}, "1.2.3.4", "test-agent")

	var invalidErr *common.TokenInvalidOrExpiredError
	require.ErrorAs(t, err, &invalidErr)
}

// listAllOptions returns list options that return every token on a single page.
func listAllOptions() utils.ListRequestOptions {
	var opts utils.ListRequestOptions
	opts.Pagination.Page = 1
	opts.Pagination.Limit = 100
	return opts
}
