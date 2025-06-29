package crypto

// This code is copied from https://github.com/dapr/kit/tree/v0.15.2/crypto/aescbcaead
// Copyright 2022 The Dapr Authors
// License: Apache2 https://github.com/dapr/kit/blob/v0.15.2/LICENSE

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewAESCBCAEAD(t *testing.T) {
	t.Run("test cases from RFC", func(t *testing.T) {
		// These test cases come from https://datatracker.ietf.org/doc/html/draft-mcgrew-aead-aes-cbc-hmac-sha2-05#section-5
		t.Run("AEAD_AES_128_CBC_HMAC_SHA256", func(t *testing.T) {
			key, _ := hex.DecodeString("000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f")
			plaintext, _ := hex.DecodeString("41206369706865722073797374656d206d757374206e6f7420626520726571756972656420746f206265207365637265742c20616e64206974206d7573742062652061626c6520746f2066616c6c20696e746f207468652068616e6473206f662074686520656e656d7920776974686f757420696e636f6e76656e69656e6365")
			nonce, _ := hex.DecodeString("1af38c2dc2b96ffdd86694092341bc04")
			aad, _ := hex.DecodeString("546865207365636f6e64207072696e6369706c65206f662041756775737465204b6572636b686f666673")
			ciphertext, _ := hex.DecodeString("c80edfa32ddf39d5ef00c0b468834279a2e46a1b8049f792f76bfe54b903a9c9a94ac9b47ad2655c5f10f9aef71427e2fc6f9b3f399a221489f16362c703233609d45ac69864e3321cf82935ac4096c86e133314c54019e8ca7980dfa4b9cf1b384c486f3a54c51078158ee5d79de59fbd34d848b3d69550a67646344427ade54b8851ffb598f7f80074b9473c82e2db652c3fa36b0a7c5b3219fab3a30bc1c4")

			aead, err := NewAESCBC128SHA256(key)
			require.NoError(t, err)
			require.Len(t, nonce, aead.NonceSize())
			require.Equal(t, 16, aead.Overhead())

			gotCiphertext := aead.Seal(nil, nonce, plaintext, aad)
			require.Equal(t, ciphertext, gotCiphertext)

			gotPlaintext, err := aead.Open(nil, nonce, gotCiphertext, aad)
			require.NoError(t, err)
			require.Equal(t, plaintext, gotPlaintext)
		})

		t.Run("AEAD_AES_192_CBC_HMAC_SHA384", func(t *testing.T) {
			key, _ := hex.DecodeString("000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f202122232425262728292a2b2c2d2e2f")
			plaintext, _ := hex.DecodeString("41206369706865722073797374656d206d757374206e6f7420626520726571756972656420746f206265207365637265742c20616e64206974206d7573742062652061626c6520746f2066616c6c20696e746f207468652068616e6473206f662074686520656e656d7920776974686f757420696e636f6e76656e69656e6365")
			nonce, _ := hex.DecodeString("1af38c2dc2b96ffdd86694092341bc04")
			aad, _ := hex.DecodeString("546865207365636f6e64207072696e6369706c65206f662041756775737465204b6572636b686f666673")
			ciphertext, _ := hex.DecodeString("ea65da6b59e61edb419be62d19712ae5d303eeb50052d0dfd6697f77224c8edb000d279bdc14c1072654bd30944230c657bed4ca0c9f4a8466f22b226d1746214bf8cfc2400add9f5126e479663fc90b3bed787a2f0ffcbf3904be2a641d5c2105bfe591bae23b1d7449e532eef60a9ac8bb6c6b01d35d49787bcd57ef484927f280adc91ac0c4e79c7b11efc60054e38490ac0e58949bfe51875d733f93ac2075168039ccc733d7")

			aead, err := NewAESCBC192SHA384(key)
			require.NoError(t, err)
			require.Len(t, nonce, aead.NonceSize())
			require.Equal(t, 24, aead.Overhead())

			gotCiphertext := aead.Seal(nil, nonce, plaintext, aad)
			require.Equal(t, ciphertext, gotCiphertext)

			gotPlaintext, err := aead.Open(nil, nonce, gotCiphertext, aad)
			require.NoError(t, err)
			require.Equal(t, plaintext, gotPlaintext)
		})

		t.Run("AEAD_AES_256_CBC_HMAC_SHA384", func(t *testing.T) {
			key, _ := hex.DecodeString("000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f202122232425262728292a2b2c2d2e2f3031323334353637")
			plaintext, _ := hex.DecodeString("41206369706865722073797374656d206d757374206e6f7420626520726571756972656420746f206265207365637265742c20616e64206974206d7573742062652061626c6520746f2066616c6c20696e746f207468652068616e6473206f662074686520656e656d7920776974686f757420696e636f6e76656e69656e6365")
			nonce, _ := hex.DecodeString("1af38c2dc2b96ffdd86694092341bc04")
			aad, _ := hex.DecodeString("546865207365636f6e64207072696e6369706c65206f662041756775737465204b6572636b686f666673")
			ciphertext, _ := hex.DecodeString("893129b0f4ee9eb18d75eda6f2aaa9f3607c98c4ba0444d34162170d8961884e58f27d4a35a5e3e3234aa99404f327f5c2d78e986e5749858b88bcddc2ba05218f195112d6ad48fa3b1e89aa7f20d596682f10b3648d3bb0c983c3185f59e36d28f647c1c13988de8ea0d821198c150977e28ca768080bc78c35faed69d8c0b7d9f506232198a489a1a6ae03a319fb30dd131d05ab3467dd056f8e882bad70637f1e9a541d9c23e7")

			aead, err := NewAESCBC256SHA384(key)
			require.NoError(t, err)
			require.Len(t, nonce, aead.NonceSize())
			require.Equal(t, 24, aead.Overhead())

			gotCiphertext := aead.Seal(nil, nonce, plaintext, aad)
			require.Equal(t, ciphertext, gotCiphertext)

			gotPlaintext, err := aead.Open(nil, nonce, gotCiphertext, aad)
			require.NoError(t, err)
			require.Equal(t, plaintext, gotPlaintext)
		})

		t.Run("AEAD_AES_256_CBC_HMAC_SHA512", func(t *testing.T) {
			key, _ := hex.DecodeString("000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f202122232425262728292a2b2c2d2e2f303132333435363738393a3b3c3d3e3f")
			plaintext, _ := hex.DecodeString("41206369706865722073797374656d206d757374206e6f7420626520726571756972656420746f206265207365637265742c20616e64206974206d7573742062652061626c6520746f2066616c6c20696e746f207468652068616e6473206f662074686520656e656d7920776974686f757420696e636f6e76656e69656e6365")
			nonce, _ := hex.DecodeString("1af38c2dc2b96ffdd86694092341bc04")
			aad, _ := hex.DecodeString("546865207365636f6e64207072696e6369706c65206f662041756775737465204b6572636b686f666673")
			ciphertext, _ := hex.DecodeString("4affaaadb78c31c5da4b1b590d10ffbd3dd8d5d302423526912da037ecbcc7bd822c301dd67c373bccb584ad3e9279c2e6d12a1374b77f077553df829410446b36ebd97066296ae6427ea75c2e0846a11a09ccf5370dc80bfecbad28c73f09b3a3b75e662a2594410ae496b2e2e6609e31e6e02cc837f053d21f37ff4f51950bbe2638d09dd7a4930930806d0703b1f64dd3b4c088a7f45c216839645b2012bf2e6269a8c56a816dbc1b267761955bc5")

			aead, err := NewAESCBC256SHA512(key)
			require.NoError(t, err)
			require.Len(t, nonce, aead.NonceSize())
			require.Equal(t, 32, aead.Overhead())

			gotCiphertext := aead.Seal(nil, nonce, plaintext, aad)
			require.Equal(t, ciphertext, gotCiphertext)

			gotPlaintext, err := aead.Open(nil, nonce, gotCiphertext, aad)
			require.NoError(t, err)
			require.Equal(t, plaintext, gotPlaintext)
		})
	})

	t.Run("dst buffer", func(t *testing.T) {
		key, _ := hex.DecodeString("000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f")
		plaintext, _ := hex.DecodeString("41206369706865722073797374656d206d757374206e6f7420626520726571756972656420746f206265207365637265742c20616e64206974206d7573742062652061626c6520746f2066616c6c20696e746f207468652068616e6473206f662074686520656e656d7920776974686f757420696e636f6e76656e69656e6365")
		nonce, _ := hex.DecodeString("1af38c2dc2b96ffdd86694092341bc04")
		aad, _ := hex.DecodeString("546865207365636f6e64207072696e6369706c65206f662041756775737465204b6572636b686f666673")
		ciphertext, _ := hex.DecodeString("c80edfa32ddf39d5ef00c0b468834279a2e46a1b8049f792f76bfe54b903a9c9a94ac9b47ad2655c5f10f9aef71427e2fc6f9b3f399a221489f16362c703233609d45ac69864e3321cf82935ac4096c86e133314c54019e8ca7980dfa4b9cf1b384c486f3a54c51078158ee5d79de59fbd34d848b3d69550a67646344427ade54b8851ffb598f7f80074b9473c82e2db652c3fa36b0a7c5b3219fab3a30bc1c4")

		aead, err := NewAESCBC128SHA256(key)
		require.NoError(t, err)

		t.Run("encrypt with empty dst buffer", func(t *testing.T) {
			dst := []byte{}
			gotCiphertext := aead.Seal(dst, nonce, plaintext, aad)
			require.Equal(t, ciphertext, gotCiphertext)
		})

		t.Run("encrypt with dst buffer with capacity", func(t *testing.T) {
			dst := make([]byte, 0, 1024)
			gotCiphertext := aead.Seal(dst, nonce, plaintext, aad)
			require.Equal(t, ciphertext, gotCiphertext)
		})

		t.Run("encrypt with non-empty dst buffer", func(t *testing.T) {
			dst := []byte{0x01, 0x02, 0x03, 0x04}
			gotCiphertext := aead.Seal(dst, nonce, plaintext, aad)
			require.Equal(t, append([]byte{0x01, 0x02, 0x03, 0x04}, ciphertext...), gotCiphertext)
		})

		t.Run("encrypt with non-empty dst buffer with capacity", func(t *testing.T) {
			dst := make([]byte, 4, 1024)
			copy(dst, []byte{0x01, 0x02, 0x03, 0x04})
			gotCiphertext := aead.Seal(dst, nonce, plaintext, aad)
			require.Equal(t, append([]byte{0x01, 0x02, 0x03, 0x04}, ciphertext...), gotCiphertext)
		})

		t.Run("decrypt with empty dst buffer", func(t *testing.T) {
			dst := []byte{}
			gotPlaintext, err := aead.Open(dst, nonce, ciphertext, aad)
			require.NoError(t, err)
			require.Equal(t, plaintext, gotPlaintext)
		})

		t.Run("decrypt with dst buffer with capacity", func(t *testing.T) {
			dst := make([]byte, 0, 1024)
			gotPlaintext, err := aead.Open(dst, nonce, ciphertext, aad)
			require.NoError(t, err)
			require.Equal(t, plaintext, gotPlaintext)
		})

		t.Run("decrypt with non-empty dst buffer", func(t *testing.T) {
			dst := []byte{0x01, 0x02, 0x03, 0x04}
			gotPlaintext, err := aead.Open(dst, nonce, ciphertext, aad)
			require.NoError(t, err)
			require.Equal(t, append([]byte{0x01, 0x02, 0x03, 0x04}, plaintext...), gotPlaintext)
		})

		t.Run("decrypt with non-empty dst buffer with capacity", func(t *testing.T) {
			dst := []byte{0x01, 0x02, 0x03, 0x04}
			gotPlaintext, err := aead.Open(dst, nonce, ciphertext, aad)
			require.NoError(t, err)
			require.Equal(t, append([]byte{0x01, 0x02, 0x03, 0x04}, plaintext...), gotPlaintext)
		})
	})
}
