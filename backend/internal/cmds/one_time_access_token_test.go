package cmds

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pocket-id/pocket-id/backend/internal/model"
	testutils "github.com/pocket-id/pocket-id/backend/internal/utils/testing"
)

func TestFindUserByUsernameOrEmailRefusesAValueOfTwoUsers(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	require.NoError(t, db.Create(&model.User{Username: "admin", Email: new("admin@example.com")}).Error)
	require.NoError(t, db.Create(&model.User{Username: "admin@example.com", Email: new("other@example.com")}).Error)

	_, err := findUserByUsernameOrEmail(t.Context(), db, "admin@example.com")
	require.ErrorContains(t, err, "matches the users")

	user, err := findUserByUsernameOrEmail(t.Context(), db, "admin")
	require.NoError(t, err)
	require.Equal(t, "admin", user.Username)
}
