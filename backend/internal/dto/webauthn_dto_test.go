package dto

import (
	"encoding/json"
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

	data, err := os.ReadFile("../../resources/aaguids.json")
	require.NoError(t, err)

	var metadata map[string]struct {
		IconLight string `json:"icon_light"`
	}
	require.NoError(t, json.Unmarshal(data, &metadata))

	for aaguid, entry := range metadata {
		if entry.IconLight != "" && utils.HasAuthenticatorIcon(aaguid) {
			return aaguid
		}
	}

	t.Skip("no authenticator icons are embedded")
	return ""
}
