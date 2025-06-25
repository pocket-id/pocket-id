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

// Encrypt a byte slice using AES-CBC-HMAC
func Encrypt(key []byte, plaintext []byte, associatedData []byte) (ciphertext []byte, err error) {
	// Pad the plaintext using PKCS#7
	plaintext, err = PadPKCS7(plaintext, aes.BlockSize)
	if err != nil {
		return nil, err
	}

	// Get the correct aead based on the key size
	var aead cipher.AEAD
	switch len(key) {
	case 32:
		aead, err = NewAESCBC128SHA256(key)
	case 48:
		aead, err = NewAESCBC192SHA384(key)
	case 56:
		aead, err = NewAESCBC256SHA384(key)
	case 64:
		aead, err = NewAESCBC256SHA512(key)
	default:
		err = errors.New("invalid key size")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	// Generate a random IV
	iv := make([]byte, aead.NonceSize())
	_, err = io.ReadFull(rand.Reader, iv)
	if err != nil {
		return nil, fmt.Errorf("failed to generate random IV: %w", err)
	}

	// Allocate the slice for the result, with additional space at the beginning for the iv
	ciphertext = make([]byte, 0, len(plaintext)+aead.NonceSize()+aead.Overhead())
	ciphertext = append(ciphertext, iv...)

	// Encrypt the plaintext
	// Tag is automatically added at the end
	ciphertext = aead.Seal(ciphertext, iv, plaintext, associatedData)

	return ciphertext, nil
}

// Decrypt a byte slice using AES-CBC-HMAC
func Decrypt(key []byte, ciphertext []byte, associatedData []byte) (plaintext []byte, err error) {
	// Get the correct aead based on the key size
	var aead cipher.AEAD
	switch len(key) {
	case 32:
		aead, err = NewAESCBC128SHA256(key)
	case 48:
		aead, err = NewAESCBC192SHA384(key)
	case 56:
		aead, err = NewAESCBC256SHA384(key)
	case 64:
		aead, err = NewAESCBC256SHA512(key)
	default:
		err = errors.New("invalid key size")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	// Extract the IV
	if len(ciphertext) < (aead.NonceSize() + aead.Overhead()) {
		return nil, ErrDecrypt
	}

	// Decrypt the data
	plaintext, err = aead.Open(nil, ciphertext[:aead.NonceSize()], ciphertext[aead.NonceSize():], associatedData)
	if err != nil {
		// Note: we do not return the exact error here, to avoid disclosing information
		return nil, ErrDecrypt
	}

	// Unpad using PKCS#7
	plaintext, err = UnpadPKCS7(plaintext, aes.BlockSize)
	if err != nil {
		// Note: we do not return the exact error here, to avoid disclosing information
		return nil, ErrDecrypt
	}

	return plaintext, nil
}
