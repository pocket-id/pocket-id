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
		{Name: "Icon cleared", AAGUID: withIcon, IconHidden: true},
		{Name: "Unknown authenticator", AAGUID: "ffffffff-ffff-ffff-ffff-ffffffffffff"},
		{Name: "Authenticator that does not identify itself", AAGUID: utils.ZeroAAGUID},
	}

	var dtos []WebauthnCredentialDto
	require.NoError(t, MapStructList(credentials, &dtos))
	require.Len(t, dtos, len(credentials))

	require.Equal(t, withIcon, dtos[0].AAGUID)
	require.Equal(t, model.WebauthnIconShown, dtos[0].Icon)

	require.Equal(t, model.WebauthnIconHidden, dtos[1].Icon)

	require.Equal(t, model.WebauthnIconNone, dtos[2].Icon)

	require.Equal(t, utils.ZeroAAGUID, dtos[3].AAGUID)
	require.Equal(t, model.WebauthnIconNone, dtos[3].Icon)

	var cleared []WebauthnCredentialDto
	require.NoError(t, MapStructList([]model.WebauthnCredential{
		{Name: "Hidden but unknown authenticator", AAGUID: "ffffffff-ffff-ffff-ffff-ffffffffffff", IconHidden: true},
	}, &cleared))
	require.Equal(t, model.WebauthnIconNone, cleared[0].Icon)
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
