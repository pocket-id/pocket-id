package datatype

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDeriveEncryptedStringKeyUsesStableValue(t *testing.T) {
	masterKey := []byte("test-encryption-key")

	expectedHex := "8a08281e815dac248bd216b8ceff063c390f3515475909bfd4380ed640bd854e"
	expected, err := hex.DecodeString(expectedHex)
	require.NoError(t, err)

	actual, err := DeriveEncryptedStringKey(masterKey)
	require.NoError(t, err)
	require.Equal(t, expected, actual)
}
