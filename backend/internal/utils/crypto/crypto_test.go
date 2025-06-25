package crypto

import (
	"crypto/rand"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEncryptDecrypt(t *testing.T) {
	tests := []struct {
		name           string
		keySize        int
		plaintext      string
		associatedData []byte
	}{
		{
			name:           "AES-128 with empty plaintext",
			keySize:        32,
			plaintext:      "",
			associatedData: nil,
		},
		{
			name:           "AES-128 with short plaintext",
			keySize:        32,
			plaintext:      "Hello, World!",
			associatedData: []byte("test-aad"),
		},
		{
			name:           "AES-192 with medium plaintext",
			keySize:        48,
			plaintext:      "This is a longer message to test encryption and decryption",
			associatedData: []byte("associated-data-192"),
		},
		{
			name:           "AES-256-384 with long plaintext",
			keySize:        56,
			plaintext:      "Lorem ipsum dolor sit amet, consectetur adipiscing elit. Sed do eiusmod tempor incididunt ut labore et dolore magna aliqua.",
			associatedData: []byte("aad-256-384"),
		},
		{
			name:           "AES-256-512 with unicode",
			keySize:        64,
			plaintext:      "Hello 世界! 🌍 Testing unicode characters", //nolint:gosmopolitan
			associatedData: []byte("unicode-test"),
		},
		{
			name:           "No associated data",
			keySize:        32,
			plaintext:      "Testing without associated data",
			associatedData: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Generate random key
			key := make([]byte, tt.keySize)
			_, err := rand.Read(key)
			require.NoError(t, err, "Failed to generate random key")

			plaintext := []byte(tt.plaintext)

			// Test encryption
			ciphertext, err := Encrypt(key, plaintext, tt.associatedData)
			require.NoError(t, err, "Encrypt should succeed")

			// Verify ciphertext is different from plaintext (unless empty)
			if len(plaintext) > 0 {
				assert.NotEqual(t, plaintext, ciphertext)
			}

			// Test decryption
			decrypted, err := Decrypt(key, ciphertext, tt.associatedData)
			require.NoError(t, err, "Decrypt should succeed")

			// Verify decrypted text matches original
			assert.Equal(t, plaintext, decrypted, "Decrypted text should match original")
		})
	}
}

func TestEncryptWithInvalidKeySize(t *testing.T) {
	invalidKeySizes := []int{16, 24, 33, 47, 55, 63, 65, 128}

	for _, keySize := range invalidKeySizes {
		t.Run(fmt.Sprintf("Key size %d", keySize), func(t *testing.T) {
			key := make([]byte, keySize)
			plaintext := []byte("test message")

			_, err := Encrypt(key, plaintext, nil)
			require.Error(t, err)
			assert.ErrorContains(t, err, "invalid key size")
		})
	}
}

func TestDecryptWithInvalidKeySize(t *testing.T) {
	invalidKeySizes := []int{16, 24, 33, 47, 55, 63, 65, 128}

	for _, keySize := range invalidKeySizes {
		t.Run(fmt.Sprintf("Key size %d", keySize), func(t *testing.T) {
			key := make([]byte, keySize)
			ciphertext := []byte("fake ciphertext")

			_, err := Decrypt(key, ciphertext, nil)
			require.Error(t, err)
			assert.ErrorContains(t, err, "invalid key size")
		})
	}
}

func TestDecryptWithInvalidCiphertext(t *testing.T) {
	key := make([]byte, 32)
	_, err := rand.Read(key)
	require.NoError(t, err, "Failed to generate random key")

	tests := []struct {
		name       string
		ciphertext []byte
	}{
		{
			name:       "empty ciphertext",
			ciphertext: []byte{},
		},
		{
			name:       "too short ciphertext",
			ciphertext: []byte("short"),
		},
		{
			name:       "random invalid data",
			ciphertext: []byte("this is not valid encrypted data"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Decrypt(key, tt.ciphertext, nil)
			require.Error(t, err)
			require.ErrorIs(t, err, ErrDecrypt)
		})
	}
}

func TestDecryptWithWrongKey(t *testing.T) {
	// Generate two different keys
	key1 := make([]byte, 32)
	key2 := make([]byte, 32)
	_, err := rand.Read(key1)
	require.NoError(t, err)
	_, err = rand.Read(key2)
	require.NoError(t, err)

	plaintext := []byte("secret message")

	// Encrypt with key1
	ciphertext, err := Encrypt(key1, plaintext, nil)
	require.NoError(t, err)

	// Try to decrypt with key2
	_, err = Decrypt(key2, ciphertext, nil)
	require.Error(t, err)
	require.ErrorIs(t, err, ErrDecrypt)
}

func TestDecryptWithWrongAssociatedData(t *testing.T) {
	key := make([]byte, 32)
	_, err := rand.Read(key)
	require.NoError(t, err, "Failed to generate random key")

	plaintext := []byte("secret message")
	correctAAD := []byte("correct-aad")
	wrongAAD := []byte("wrong-aad")

	// Encrypt with correct AAD
	ciphertext, err := Encrypt(key, plaintext, correctAAD)
	require.NoError(t, err)

	// Try to decrypt with wrong AAD
	_, err = Decrypt(key, ciphertext, wrongAAD)
	require.Error(t, err)
	require.ErrorIs(t, err, ErrDecrypt)

	// Verify correct AAD works
	decrypted, err := Decrypt(key, ciphertext, correctAAD)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decrypted, "Decrypted text should match original when using correct AAD")
}

func TestEncryptDecryptConsistency(t *testing.T) {
	key := make([]byte, 32)
	_, err := rand.Read(key)
	require.NoError(t, err)

	plaintext := []byte("consistency test message")
	associatedData := []byte("test-aad")

	// Encrypt multiple times and verify we get different ciphertexts (due to random IV)
	ciphertext1, err := Encrypt(key, plaintext, associatedData)
	require.NoError(t, err)

	ciphertext2, err := Encrypt(key, plaintext, associatedData)
	require.NoError(t, err)

	// Ciphertexts should be different (due to random IV)
	assert.NotEqual(t, ciphertext1, ciphertext2, "Multiple encryptions of same plaintext should produce different ciphertexts")

	// Both should decrypt to the same plaintext
	decrypted1, err := Decrypt(key, ciphertext1, associatedData)
	require.NoError(t, err)

	decrypted2, err := Decrypt(key, ciphertext2, associatedData)
	require.NoError(t, err)

	assert.Equal(t, plaintext, decrypted1, "First decrypted text should match original")
	assert.Equal(t, plaintext, decrypted2, "Second decrypted text should match original")
	assert.Equal(t, decrypted1, decrypted2, "Both decrypted texts should be identical")
}
