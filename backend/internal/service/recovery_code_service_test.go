package service

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/model"
)

func newRecoveryCodeTestService(t *testing.T, allow bool) (*RecoveryCodeService, model.User) {
	t.Helper()

	value := "true"
	if !allow {
		value = "false"
	}
	appConfig := NewTestAppConfigService(&model.AppConfig{
		SessionDuration:    model.AppConfigVariable{Value: "60"},
		AllowRecoveryCodes: model.AppConfigVariable{Value: value},
	})

	jwtService, db, _ := setupJwtService(t, appConfig)

	user := model.User{
		Base:     model.Base{ID: "recovery-user"},
		Username: "recovery-user",
	}
	require.NoError(t, db.Create(&user).Error)

	auditLogService := NewAuditLogService(db, appConfig, nil, NewGeoLiteService(nil))

	return NewRecoveryCodeService(db, appConfig, jwtService, auditLogService), user
}

func TestRecoveryCodeService_Generate(t *testing.T) {
	t.Run("returns a batch of codes when enabled", func(t *testing.T) {
		service, user := newRecoveryCodeTestService(t, true)

		codes, err := service.GenerateForUser(t.Context(), user.ID, "", "")
		require.NoError(t, err)
		assert.Len(t, codes, recoveryCodeBatchSize)

		for _, c := range codes {
			assert.NotEmpty(t, c)
		}

		total, unused, err := service.StatusForUser(t.Context(), user.ID)
		require.NoError(t, err)
		assert.Equal(t, recoveryCodeBatchSize, total)
		assert.Equal(t, recoveryCodeBatchSize, unused)
	})

	t.Run("invalidates previously issued codes on regeneration", func(t *testing.T) {
		service, user := newRecoveryCodeTestService(t, true)

		first, err := service.GenerateForUser(t.Context(), user.ID, "", "")
		require.NoError(t, err)

		second, err := service.GenerateForUser(t.Context(), user.ID, "", "")
		require.NoError(t, err)

		assert.NotEqual(t, first, second)

		// The first batch should no longer be redeemable.
		_, _, err = service.Redeem(t.Context(), first[0], "", "")
		require.Error(t, err)
		assert.ErrorAs(t, err, new(*common.RecoveryCodeInvalidError))
	})

	t.Run("returns an error when the feature is disabled", func(t *testing.T) {
		service, user := newRecoveryCodeTestService(t, false)

		_, err := service.GenerateForUser(t.Context(), user.ID, "", "")
		require.Error(t, err)
		assert.ErrorAs(t, err, new(*common.RecoveryCodesDisabledError))
	})
}

func TestRecoveryCodeService_Redeem(t *testing.T) {
	t.Run("a fresh code signs the user in once", func(t *testing.T) {
		service, user := newRecoveryCodeTestService(t, true)

		codes, err := service.GenerateForUser(t.Context(), user.ID, "", "")
		require.NoError(t, err)

		signedInUser, token, err := service.Redeem(t.Context(), codes[0], "", "")
		require.NoError(t, err)
		assert.Equal(t, user.ID, signedInUser.ID)
		assert.NotEmpty(t, token)

		// Reusing the same code fails.
		_, _, err = service.Redeem(t.Context(), codes[0], "", "")
		require.Error(t, err)
		assert.ErrorAs(t, err, new(*common.RecoveryCodeInvalidError))

		// Status now reports one used code.
		total, unused, err := service.StatusForUser(t.Context(), user.ID)
		require.NoError(t, err)
		assert.Equal(t, recoveryCodeBatchSize, total)
		assert.Equal(t, recoveryCodeBatchSize-1, unused)
	})

	t.Run("accepts codes with and without the dashes the user sees", func(t *testing.T) {
		service, user := newRecoveryCodeTestService(t, true)

		codes, err := service.GenerateForUser(t.Context(), user.ID, "", "")
		require.NoError(t, err)

		stripped := normalizeRecoveryCode(codes[0])
		_, _, err = service.Redeem(t.Context(), stripped, "", "")
		require.NoError(t, err)
	})

	t.Run("accepts codes typed in a different case", func(t *testing.T) {
		service, user := newRecoveryCodeTestService(t, true)

		codes, err := service.GenerateForUser(t.Context(), user.ID, "", "")
		require.NoError(t, err)

		// A user recovering under pressure may type the code in upper or
		// lower case; we must accept either so long as the characters match.
		flipped := strings.ToUpper(codes[0])
		_, _, err = service.Redeem(t.Context(), flipped, "", "")
		require.NoError(t, err)
	})

	t.Run("rejects redemption when the feature has been disabled", func(t *testing.T) {
		service, user := newRecoveryCodeTestService(t, true)

		codes, err := service.GenerateForUser(t.Context(), user.ID, "", "")
		require.NoError(t, err)

		// Flip the feature off and retry.
		service.appConfigService.GetDbConfig().AllowRecoveryCodes.Value = "false"

		_, _, err = service.Redeem(t.Context(), codes[0], "", "")
		require.Error(t, err)
		assert.ErrorAs(t, err, new(*common.RecoveryCodesDisabledError))
	})

	t.Run("rejects a garbage code", func(t *testing.T) {
		service, _ := newRecoveryCodeTestService(t, true)

		_, _, err := service.Redeem(t.Context(), "not-a-real-code", "", "")
		require.Error(t, err)
		assert.ErrorAs(t, err, new(*common.RecoveryCodeInvalidError))
	})
}

func TestRecoveryCodeService_RevokeAll(t *testing.T) {
	t.Run("wipes the user's batch", func(t *testing.T) {
		service, user := newRecoveryCodeTestService(t, true)

		_, err := service.GenerateForUser(t.Context(), user.ID, "", "")
		require.NoError(t, err)

		require.NoError(t, service.RevokeAllForUser(t.Context(), user.ID, "", ""))

		total, unused, err := service.StatusForUser(t.Context(), user.ID)
		require.NoError(t, err)
		assert.Equal(t, 0, total)
		assert.Equal(t, 0, unused)
	})

	t.Run("refuses to revoke when the feature is disabled", func(t *testing.T) {
		service, user := newRecoveryCodeTestService(t, false)

		err := service.RevokeAllForUser(t.Context(), user.ID, "", "")
		require.Error(t, err)
		assert.ErrorAs(t, err, new(*common.RecoveryCodesDisabledError))
	})
}

func TestRecoveryCodeService_Status(t *testing.T) {
	t.Run("refuses to report status when the feature is disabled", func(t *testing.T) {
		service, user := newRecoveryCodeTestService(t, false)

		_, _, err := service.StatusForUser(t.Context(), user.ID)
		require.Error(t, err)
		assert.ErrorAs(t, err, new(*common.RecoveryCodesDisabledError))
	})
}
