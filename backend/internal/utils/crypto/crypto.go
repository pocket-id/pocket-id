package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
)

// ErrDecrypt is returned by Decrypt when the operation failed for any reason
var ErrDecrypt = errors.New("failed to decrypt data")

// Encrypt a byte slice using AES-GCM and a random nonce
// Important: do not encrypt more than ~4 billion messages with the same key!
func Encrypt(key []byte, plaintext []byte, associatedData []byte) (ciphertext []byte, err error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create block cipher: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create AEAD cipher: %w", err)
	}

	// Generate a random nonce
	nonce := make([]byte, aead.NonceSize())
	_, err = io.ReadFull(rand.Reader, nonce)
	if err != nil {
		return nil, fmt.Errorf("failed to generate random nonce: %w", err)
	}

	// Allocate the slice for the result, with additional space for the nonce and overhead
	ciphertext = make([]byte, 0, len(plaintext)+aead.NonceSize()+aead.Overhead())
	ciphertext = append(ciphertext, nonce...)

	// Encrypt the plaintext
	// Tag is automatically added at the end
	ciphertext = aead.Seal(ciphertext, nonce, plaintext, associatedData)

	return ciphertext, nil
}

// Decrypt a byte slice using AES-GCM
func Decrypt(key []byte, ciphertext []byte, associatedData []byte) (plaintext []byte, err error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create block cipher: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create AEAD cipher: %w", err)
	}

	// Extract the nonce
	if len(ciphertext) < (aead.NonceSize() + aead.Overhead()) {
		return nil, ErrDecrypt
	}

	// Decrypt the data
	plaintext, err = aead.Open(nil, ciphertext[:aead.NonceSize()], ciphertext[aead.NonceSize():], associatedData)
	if err != nil {
		// Note: we do not return the exact error here, to avoid disclosing information
		return nil, ErrDecrypt
	}

	return plaintext, nil
}
