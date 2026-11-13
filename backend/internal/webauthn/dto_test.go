package webauthn

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pocket-id/pocket-id/backend/internal/dto"
)

func TestWebauthnCredentialDto_iconFields(t *testing.T) {
	withIcon := anyAAGUIDWithIcon(t)

	credentials := []WebauthnCredential{
		{Name: "With icon", AAGUID: withIcon},
		{Name: "Unknown authenticator", AAGUID: "ffffffff-ffff-ffff-ffff-ffffffffffff"},
		{Name: "Authenticator that does not identify itself", AAGUID: zeroAAGUID},
	}

	var dtos []WebauthnCredentialDto
	require.NoError(t, dto.MapStructList(credentials, &dtos))
	require.Len(t, dtos, len(credentials))

	require.Equal(t, withIcon, dtos[0].AAGUID)
	require.True(t, dtos[0].HasIcon)

	require.False(t, dtos[1].HasIcon)

	require.Equal(t, zeroAAGUID, dtos[2].AAGUID)
	require.False(t, dtos[2].HasIcon)
}

func TestWebauthnCredentialDto_verifiedAuthenticatorName(t *testing.T) {
	aaguid := anyAAGUIDWithIcon(t)

	credentials := []WebauthnCredential{
		{Name: "Verified", AAGUID: aaguid, AttestationVerified: true},
		{Name: "Claimed only", AAGUID: aaguid},
	}

	var dtos []WebauthnCredentialDto
	require.NoError(t, dto.MapStructList(credentials, &dtos))

	require.Equal(t, getAuthenticatorName(aaguid), dtos[0].VerifiedAuthenticatorName)
	require.NotEmpty(t, dtos[0].VerifiedAuthenticatorName)
	require.Empty(t, dtos[1].VerifiedAuthenticatorName)
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
		if entry.IconLight != "" && hasAuthenticatorIcon(aaguid) {
			return aaguid
		}
	}

	t.Skip("no authenticator icons are embedded")
	return ""
}
