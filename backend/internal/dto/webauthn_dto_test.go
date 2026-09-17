package dto

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pocket-id/pocket-id/backend/internal/model"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
)

func TestWebauthnCredentialDto_iconFields(t *testing.T) {
	withIcon := anyAAGUIDWithIcon(t)

	credentials := []model.WebauthnCredential{
		{Name: "With icon", AAGUID: withIcon},
		{Name: "Unknown authenticator", AAGUID: "ffffffff-ffff-ffff-ffff-ffffffffffff"},
		{Name: "Authenticator that does not identify itself", AAGUID: utils.ZeroAAGUID},
	}

	var dtos []WebauthnCredentialDto
	require.NoError(t, MapStructList(credentials, &dtos))
	require.Len(t, dtos, len(credentials))

	require.Equal(t, withIcon, dtos[0].AAGUID)
	require.True(t, dtos[0].HasIcon)

	require.False(t, dtos[1].HasIcon)

	require.Equal(t, utils.ZeroAAGUID, dtos[2].AAGUID)
	require.False(t, dtos[2].HasIcon)
}

func anyAAGUIDWithIcon(t *testing.T) string {
	t.Helper()

	entries, err := os.ReadDir("../../resources/aaguid-icons")
	require.NoError(t, err)

	for _, entry := range entries {
		aaguid := entry.Name()[:len(entry.Name())-len(".svg")]
		if utils.HasAuthenticatorIcon(aaguid) {
			return aaguid
		}
	}

	t.Skip("no authenticator icons are embedded")
	return ""
}
