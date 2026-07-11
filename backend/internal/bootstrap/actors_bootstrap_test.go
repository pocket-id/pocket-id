package bootstrap

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pocket-id/pocket-id/backend/internal/common"
)

func TestNewActorsOptsGetPSKUsesStableValue(t *testing.T) {
	opts := NewActorsOpts{
		EnvConfig: &common.EnvConfigSchema{
			EncryptionKey: []byte("test-encryption-key"),
		},
		// Constant value for this test
		InstanceID: "ee05c3eb-8129-47a6-a1c7-849998b6f876",
	}

	expectedHex := "db09067fa194c3731bf77b6415a1c5d903f03d4557605ba3236b31f6eddfc8d7"
	expected, err := hex.DecodeString(expectedHex)
	require.NoError(t, err)

	actual, err := opts.getPSK()
	require.NoError(t, err)
	require.Equalf(t, expected, actual, "actual result: %s", actual)
}
