package auditlogs

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/model"
	testutils "github.com/pocket-id/pocket-id/backend/internal/utils/testing"
)

func TestDeletingAUserKeepsTheirEntries(t *testing.T) {
	testutils.ForEachDatabase(t, func(t *testing.T, db *gorm.DB) {
		if db.Name() == "sqlite" {
			require.NoError(t, db.Exec("PRAGMA foreign_keys = ON").Error)
		}
		user := model.User{Base: model.Base{ID: "00000000-0000-0000-0000-000000000001"}, Username: "alice"}
		require.NoError(t, db.Create(&user).Error)
		s := newService(db, nil, signInLocationResolver{}, nil, browserTokenStub{})
		entry, ok := s.Create(t.Context(), EventPasskeyRemoved, "", "", user.ID, Data{"passkeyName": "Laptop"}, db)
		require.True(t, ok)

		err := db.Transaction(func(tx *gorm.DB) error {
			require.NoError(t, s.KeepUsername(t.Context(), user.ID, user.Username, tx))
			return tx.Delete(&user).Error
		})
		require.NoError(t, err)

		var kept AuditLog
		require.NoError(t, db.First(&kept, "id = ?", entry.ID).Error)
		assert.Empty(t, kept.UserID)
		assert.Equal(t, Data{"passkeyName": "Laptop", "username": "alice"}, kept.Data)
	})
}
