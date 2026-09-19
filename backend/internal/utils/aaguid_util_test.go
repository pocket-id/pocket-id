package utils

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pocket-id/pocket-id/backend/resources"
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
	// Preserve the package-level metadata cache so this test does not affect other tests
	originalMetadata := aaguidMetadata
	originalOnce := aaguidMetadataOnce
	defer func() {
		aaguidMetadata = originalMetadata
		aaguidMetadataOnce = originalOnce
	}()

	// Inject test metadata without loading the embedded manifest
	aaguidMetadata = map[string]authenticatorMetadata{
		"adce0002-35bc-c60a-648b-0b25f1f05503": {Name: "Test Authenticator"},
		"00000000-0000-0000-0000-000000000000": {Name: "Zero Authenticator"},
	}
	aaguidMetadataOnce = &sync.Once{}
	aaguidMetadataOnce.Do(func() {})

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

func TestLoadAAGUIDMetadataFromFile(t *testing.T) {
	// Reset the metadata cache so this test exercises the embedded manifest
	aaguidMetadata = nil
	aaguidMetadataOnce = &sync.Once{}

	// Trigger loading by resolving an arbitrary AAGUID
	GetAuthenticatorName([]byte{0x01, 0x02, 0x03, 0x04})

	if len(aaguidMetadata) == 0 {
		t.Error("loadAAGUIDMetadataFromFile() failed to populate aaguidMetadata")
	}

	t.Log("AAGUID metadata loaded with", len(aaguidMetadata), "entries")
}

// mustDecodeHex keeps the table fixtures readable
func mustDecodeHex(s string) []byte {
	bytes, err := hex.DecodeString(s)
	if err != nil {
		panic("invalid hex in test: " + err.Error())
	}
	return bytes
}

func TestAuthenticatorIcons(t *testing.T) {
	aaguidMetadataOnce.Do(loadAAGUIDMetadataFromFile)
	if len(aaguidMetadata) == 0 {
		t.Skip("no authenticator icons are embedded")
	}

	var withDark, withoutDark string
	referencedIcons := make(map[string]struct{})
	for aaguid, metadata := range aaguidMetadata {
		if metadata.IconLight == "" {
			continue
		}

		require.True(t, validAuthenticatorIconName(metadata.IconLight), "icon %q has an invalid light reference", aaguid)
		referencedIcons[metadata.IconLight] = struct{}{}

		if metadata.IconDark != "" {
			require.True(t, validAuthenticatorIconName(metadata.IconDark), "icon %q has an invalid dark reference", aaguid)
			referencedIcons[metadata.IconDark] = struct{}{}
		}

		if metadata.IconDark != "" && withDark == "" {
			withDark = aaguid
		}
		if metadata.IconDark == "" && withoutDark == "" {
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

	t.Run("manifest references every content-addressed icon", func(t *testing.T) {
		entries, err := resources.FS.ReadDir(aaguidIconsDir)
		require.NoError(t, err)
		require.Len(t, entries, len(referencedIcons))

		for _, entry := range entries {
			name := entry.Name()
			require.Contains(t, referencedIcons, name)

			data, err := resources.FS.ReadFile(path.Join(aaguidIconsDir, name))
			require.NoError(t, err)
			digest := sha256.Sum256(data)
			require.Equal(t, fmt.Sprintf("%x.svg", digest[:authenticatorIconHashBytes]), name)
		}
	})
}
