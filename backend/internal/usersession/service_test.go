//go:build unit

package usersession

import (
	"testing"
	"time"

	"github.com/lestrrat-go/jwx/v4/jwt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/model"
	testutils "github.com/pocket-id/pocket-id/backend/internal/utils/testing"
)

func TestService(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	service := newService(db)

	user := model.User{Username: "session-user"}
	require.NoError(t, db.Create(&user).Error)

	mintToken := func(t *testing.T, jti, subject string, expiresAt time.Time) string {
		t.Helper()
		builder := jwt.NewBuilder().Expiration(expiresAt)
		if jti != "" {
			builder = builder.JwtID(jti)
		}
		if subject != "" {
			builder = builder.Subject(subject)
		}
		token, err := builder.Build()
		require.NoError(t, err)
		signed, err := jwt.Sign(token, jwt.WithInsecureNoSignature())
		require.NoError(t, err)
		return string(signed)
	}

	t.Run("track and validate", func(t *testing.T) {
		accessToken := mintToken(t, "session-live", user.ID, time.Now().Add(time.Hour))
		require.NoError(t, service.Track(t.Context(), db, accessToken))

		require.NoError(t, service.Validate(t.Context(), "session-live"))

		var session UserSession
		require.NoError(t, db.First(&session, "id = ?", "session-live").Error)
		assert.Equal(t, user.ID, session.UserID)
	})

	t.Run("validate unknown session", func(t *testing.T) {
		err := service.Validate(t.Context(), "never-tracked")
		assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
	})

	t.Run("track rejects tokens with missing claims", func(t *testing.T) {
		require.Error(t, service.Track(t.Context(), db, mintToken(t, "", user.ID, time.Now().Add(time.Hour))))
		require.Error(t, service.Track(t.Context(), db, mintToken(t, "session-no-subject", "", time.Now().Add(time.Hour))))
		require.Error(t, service.Track(t.Context(), db, "not-a-jwt"))
	})

	t.Run("delete ends the session", func(t *testing.T) {
		accessToken := mintToken(t, "session-to-end", user.ID, time.Now().Add(time.Hour))
		require.NoError(t, service.Track(t.Context(), db, accessToken))
		require.NoError(t, service.Validate(t.Context(), "session-to-end"))

		require.NoError(t, service.Delete(t.Context(), "session-to-end"))
		err := service.Validate(t.Context(), "session-to-end")
		require.ErrorIs(t, err, gorm.ErrRecordNotFound)

		require.NoError(t, service.Delete(t.Context(), "session-to-end"))
	})

	t.Run("cleanup deletes only expired sessions", func(t *testing.T) {
		live := mintToken(t, "cleanup-live", user.ID, time.Now().Add(time.Hour))
		expired := mintToken(t, "cleanup-expired", user.ID, time.Now().Add(-time.Minute))
		require.NoError(t, service.Track(t.Context(), db, live))
		require.NoError(t, service.Track(t.Context(), db, expired))

		count, err := service.cleanupExpired(t.Context())
		require.NoError(t, err)
		assert.Equal(t, int64(1), count)

		require.NoError(t, service.Validate(t.Context(), "cleanup-live"))
		err = service.Validate(t.Context(), "cleanup-expired")
		assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
	})
}
