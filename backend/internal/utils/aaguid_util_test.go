package utils

import (
	"encoding/hex"
	"testing"
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
	originalMap := AAGUIDMap
	defer func() {
		AAGUIDMap = originalMap
	}()

	// Inject a test AAGUID map
	AAGUIDMap = map[string]string{
		"adce0002-35bc-c60a-648b-0b25f1f05503": "Test Authenticator",
		"00000000-0000-0000-0000-000000000000": "Zero Authenticator",
	}

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

// Helper function to convert hex string to bytes
func mustDecodeHex(s string) []byte {
	bytes, err := hex.DecodeString(s)
	if err != nil {
		panic("invalid hex in test: " + err.Error())
	}
	return bytes
}
