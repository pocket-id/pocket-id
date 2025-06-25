package crypto

// This implements an AEAD with AES-CBC and HMAC-SHA256
// Specs:
// - https://datatracker.ietf.org/doc/html/draft-mcgrew-aead-aes-cbc-hmac-sha2-05
// - https://www.rfc-editor.org/rfc/rfc7518#section-5.2
//
// This code is copied from https://github.com/dapr/kit/tree/v0.15.2/crypto/aescbcaead
// Copyright 2022 The Dapr Authors
// License: Apache2 https://github.com/dapr/kit/blob/v0.15.2/LICENSE
//
// The code is inspired by https://github.com/codahale/etm
// Copyright (c) 2014 Coda Hale
// License: MIT https://github.com/codahale/etm/blob/master/LICENSE

import (
	"crypto"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"encoding/binary"
	"errors"
	"fmt"
	"hash"
)

// NewAESCBC128SHA256 returns an AEAD_AES_128_CBC_HMAC_SHA_256 instance given a
// 32-byte key or an error if the key is the wrong size.
// AEAD_AES_128_CBC_HMAC_SHA_256 combines AES-128 in CBC mode with
// HMAC-SHA-256-128.
func NewAESCBC128SHA256(key []byte) (cipher.AEAD, error) {
	return NewAESCBCAEAD(aesCBCAEADParams{
		macAlg:     crypto.SHA256.New,
		encKeySize: 16,
		macKeySize: 16,
		tagSize:    16,
		key:        key,
	})
}

// NewAESCBC192SHA384 returns an AEAD_AES_192_CBC_HMAC_SHA_384 instance given a
// 48-byte key or an error if the key is the wrong size.
// AEAD_AES_192_CBC_HMAC_SHA_384 combines AES-192 in CBC mode with
// HMAC-SHA-384-192.
func NewAESCBC192SHA384(key []byte) (cipher.AEAD, error) {
	return NewAESCBCAEAD(aesCBCAEADParams{
		macAlg:     crypto.SHA384.New,
		encKeySize: 24,
		macKeySize: 24,
		tagSize:    24,
		key:        key,
	})
}

// NewAESCBC256SHA384 returns an AEAD_AES_256_CBC_HMAC_SHA_384 instance given a
// 56-byte key or an error if the key is the wrong size.
// AEAD_AES_256_CBC_HMAC_SHA_384 combines AES-256 in CBC mode with
// HMAC-SHA-384-192.
func NewAESCBC256SHA384(key []byte) (cipher.AEAD, error) {
	return NewAESCBCAEAD(aesCBCAEADParams{
		macAlg:     crypto.SHA384.New,
		encKeySize: 32,
		macKeySize: 24,
		tagSize:    24,
		key:        key,
	})
}

// NewAESCBC256SHA512 returns an AEAD_AES_256_CBC_HMAC_SHA_512 instance given a
// 64-byte key or an error if the key is the wrong size.
// AEAD_AES_256_CBC_HMAC_SHA_512 combines AES-256 in CBC mode with
// HMAC-SHA-512-256.
func NewAESCBC256SHA512(key []byte) (cipher.AEAD, error) {
	return NewAESCBCAEAD(aesCBCAEADParams{
		macAlg:     crypto.SHA512.New,
		encKeySize: 32,
		macKeySize: 32,
		tagSize:    32,
		key:        key,
	})
}

type aesCBCAEADParams struct {
	encKeySize, macKeySize, tagSize int

	key    []byte
	macAlg func() hash.Hash
}

// NewAESCBCAEAD creates a new AEAD cipher based on AES-CBC with HMAC-SHA.
func NewAESCBCAEAD(p aesCBCAEADParams) (cipher.AEAD, error) {
	l := p.encKeySize + p.macKeySize
	if len(p.key) != l {
		return nil, fmt.Errorf("key must be %d bytes long", l)
	}
	macKey := p.key[0:p.macKeySize]
	encKey := p.key[len(p.key)-p.encKeySize:]
	return &aesCBCAEAD{
		aesCBCAEADParams: p,
		encKey:           encKey,
		macKey:           macKey,
	}, nil
}

type aesCBCAEAD struct {
	aesCBCAEADParams
	encKey, macKey []byte
}

func (aead *aesCBCAEAD) Overhead() int {
	return aead.tagSize
}

func (aead *aesCBCAEAD) NonceSize() int {
	return aes.BlockSize
}

func (aead *aesCBCAEAD) Seal(dst, nonce, plaintext, additionalData []byte) []byte {
	// In this method, we panic in case of errors because the aead.Seal() interface doesn't allow returning errors
	// However, errors in this method should only happen due to development-time mistakes, so we should never have to panic at runtime.
	if len(nonce) != aes.BlockSize {
		panic("invalid nonce")
	}

	// Create the cipher
	block, err := aes.NewCipher(aead.encKey)
	if err != nil {
		panic(err)
	}

	// Pad the plaintext with PKCS#7 per specs
	plaintext, err = PadPKCS7(plaintext, aes.BlockSize)
	if err != nil {
		panic(err)
	}

	// Allocate a byte slice large enough to contain the ciphertext and the tag
	size := len(plaintext) + aead.tagSize
	dstLen := len(dst)
	if cap(dst) >= (dstLen + size) {
		dst = dst[:dstLen+size]
	} else {
		d := make([]byte, dstLen+size)
		copy(d, dst)
		dst = d
	}
	out := dst[dstLen:]

	// Encrypt the message
	cipher.NewCBCEncrypter(block, nonce).
		CryptBlocks(out[:len(out)-aead.tagSize], plaintext)

	// Compute the authentication tag and append it at the end
	tag := aead.hmacTag(hmac.New(aead.macAlg, aead.macKey), additionalData, nonce, out[:len(out)-aead.tagSize], aead.tagSize)
	copy(out[len(out)-aead.tagSize:], tag)

	return dst
}

func (aead *aesCBCAEAD) Open(dst, nonce, ciphertext, additionalData []byte) ([]byte, error) {
	if len(ciphertext) < aead.tagSize {
		return nil, errors.New("invalid ciphertext size")
	}

	// Remove the tag from the end of the ciphertext
	ciphertextTag := ciphertext[len(ciphertext)-aead.tagSize:]
	ciphertext = ciphertext[:len(ciphertext)-aead.tagSize]

	// First, check the authentication tag matches
	expectTag := aead.hmacTag(hmac.New(aead.macAlg, aead.macKey), additionalData, nonce, ciphertext, aead.tagSize)
	if !hmac.Equal(ciphertextTag, expectTag) {
		return nil, errors.New("message authentication failed")
	}

	// Ensure the destination slice has enough capacity
	size := len(ciphertext)
	dstLen := len(dst)
	if cap(dst) >= (dstLen + size) {
		dst = dst[:dstLen+size]
	} else {
		d := make([]byte, dstLen+size)
		copy(d, dst)
		dst = d
	}
	out := dst[dstLen:]

	// Decrypt the ciphertext
	block, err := aes.NewCipher(aead.encKey)
	if err != nil {
		// Should never happen
		return nil, err
	}
	cipher.NewCBCDecrypter(block, nonce).
		CryptBlocks(out, ciphertext)

	// Remove PKCS#7 padding
	out, err = UnpadPKCS7(out, aes.BlockSize)
	if err != nil {
		return nil, err
	}
	dst = dst[:dstLen+len(out)]
	return dst, nil
}

// Computes the HMAC tag as per specs.
func (aead *aesCBCAEAD) hmacTag(h hash.Hash, additionalData, nonce, ciphertext []byte, l int) []byte {
	al := make([]byte, 8)
	binary.BigEndian.PutUint64(al, uint64(len(additionalData))<<3) // In bits

	h.Write(additionalData)
	h.Write(nonce)
	h.Write(ciphertext)
	h.Write(al)

	return h.Sum(nil)[:l]
}
