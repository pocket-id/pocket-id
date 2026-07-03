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
	}

	expectedHex := "651300d35d48998d0fa66ac89091bcde8ed0fd0aa35fbb849f068410c64807e9"
	expected, err := hex.DecodeString(expectedHex)
	require.NoError(t, err)

	actual, err := opts.getPSK()
	require.NoError(t, err)
	require.Equal(t, expected, actual)
}
