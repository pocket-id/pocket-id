package utils

import (
	"encoding/hex"
	"io"
	"os"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFormatAAGUID(t *testing.T) {
	tests := []struct {
		name   string
		aaguid []byte
		want   string
	}{
		{
			name:   "empty byte slice",
			aaguid: []byte{},
			want:   "",
		},
		{
			name:   "16 byte slice - standard UUID",
			aaguid: []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10},
			want:   "01020304-0506-0708-090a-0b0c0d0e0f10",
		},
		{
			name:   "non-16 byte slice",
			aaguid: []byte{0x01, 0x02, 0x03, 0x04, 0x05},
			want:   "0102030405",
		},
		{
			name:   "specific UUID example",
			aaguid: mustDecodeHex("adce000235bcc60a648b0b25f1f05503"),
			want:   "adce0002-35bc-c60a-648b-0b25f1f05503",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatAAGUID(tt.aaguid)
			if got != tt.want {
				t.Errorf("FormatAAGUID() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetAuthenticatorName(t *testing.T) {
	// Reset the aaguidMap for testing
	originalMap := aaguidMap
	originalOnce := aaguidMapOnce
	defer func() {
		aaguidMap = originalMap
		aaguidMapOnce = originalOnce
	}()

	// Inject a test AAGUID map
	aaguidMap = map[string]string{
		"adce0002-35bc-c60a-648b-0b25f1f05503": "Test Authenticator",
		"00000000-0000-0000-0000-000000000000": "Zero Authenticator",
	}
	aaguidMapOnce = &sync.Once{}
	aaguidMapOnce.Do(func() {}) // Mark as done to avoid loading from file

	tests := []struct {
		name   string
		aaguid []byte
		want   string
	}{
		{
			name:   "empty byte slice",
			aaguid: []byte{},
			want:   "",
		},
		{
			name:   "known AAGUID",
			aaguid: mustDecodeHex("adce000235bcc60a648b0b25f1f05503"),
			want:   "Test Authenticator Passkey",
		},
		{
			name:   "zero UUID",
			aaguid: mustDecodeHex("00000000000000000000000000000000"),
			want:   "Zero Authenticator Passkey",
		},
		{
			name:   "unknown AAGUID",
			aaguid: mustDecodeHex("ffffffffffffffffffffffffffffffff"),
			want:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetAuthenticatorName(tt.aaguid)
			if got != tt.want {
				t.Errorf("GetAuthenticatorName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLoadAAGUIDsFromFile(t *testing.T) {
	// Reset the map and once flag for clean testing
	aaguidMap = nil
	aaguidMapOnce = &sync.Once{}

	// Trigger loading of AAGUIDs by calling GetAuthenticatorName
	GetAuthenticatorName([]byte{0x01, 0x02, 0x03, 0x04})

	if len(aaguidMap) == 0 {
		t.Error("loadAAGUIDsFromFile() failed to populate aaguidMap")
	}

	// Check for a few known entries that should be in the embedded file
	// This test will be more brittle as it depends on the content of aaguids.json,
	// but it helps verify that the loading actually worked
	t.Log("AAGUID map loaded with", len(aaguidMap), "entries")
}

// Helper function to convert hex string to bytes
func mustDecodeHex(s string) []byte {
	bytes, err := hex.DecodeString(s)
	if err != nil {
		panic("invalid hex in test: " + err.Error())
	}
	return bytes
}

func TestAuthenticatorIcons(t *testing.T) {
	aaguidIconsOnce.Do(loadAAGUIDIconsFromFS)
	if len(aaguidIcons) == 0 {
		t.Skip("no authenticator icons are embedded")
	}

	var withDark, withoutDark string
	for aaguid, icon := range aaguidIcons {
		require.NotEmpty(t, icon.lightPath, "icon %q has no light variant", aaguid)

		if icon.darkPath != "" && withDark == "" {
			withDark = aaguid
		}
		if icon.darkPath == "" && withoutDark == "" {
			withoutDark = aaguid
		}
	}
	require.NotEmpty(t, withDark, "expected at least one authenticator with a separate dark icon")
	require.NotEmpty(t, withoutDark, "expected at least one authenticator with a single icon")

	readIcon := func(t *testing.T, aaguid string, light bool) []byte {
		t.Helper()

		file, size, err := OpenAuthenticatorIcon(aaguid, light)
		require.NoError(t, err)
		defer file.Close()

		data, err := io.ReadAll(file)
		require.NoError(t, err)
		require.Len(t, data, int(size))
		require.Contains(t, string(data), "<svg")

		return data
	}

	t.Run("known AAGUID", func(t *testing.T) {
		require.True(t, HasAuthenticatorIcon(withDark))
		require.NotEqual(t, readIcon(t, withDark, true), readIcon(t, withDark, false))
	})

	t.Run("dark falls back to the light icon", func(t *testing.T) {
		require.Equal(t, readIcon(t, withoutDark, true), readIcon(t, withoutDark, false))
	})

	t.Run("unknown AAGUID", func(t *testing.T) {
		tests := []string{
			"",
			"ffffffff-ffff-ffff-ffff-ffffffffffff",
			"../aaguids.json",
			"..%2faaguids.json",
			"a/b",
			".",
		}

		for _, aaguid := range tests {
			t.Run(aaguid, func(t *testing.T) {
				require.False(t, HasAuthenticatorIcon(aaguid))

				_, _, err := OpenAuthenticatorIcon(aaguid, true)
				require.ErrorIs(t, err, os.ErrNotExist)
			})
		}
	})
}
