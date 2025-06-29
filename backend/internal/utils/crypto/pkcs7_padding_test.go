package crypto

/*!
This code is copied from https://github.com/dapr/kit/tree/v0.15.2/crypto/aescbcaead
Copyright 2022 The Dapr Authors
License: Apache2 https://github.com/dapr/kit/blob/v0.15.2/LICENSE

This code is adapted from https://github.com/mergermarket/go-pkcs7/tree/153b18ea13c9b94f698070cadb23701e51a55b3e
Copyright (c) 2017 Richard Zadorozny
License: MIT https://github.com/mergermarket/go-pkcs7/blob/153b18ea13c9b94f698070cadb23701e51a55b3e/LICENSE
*/

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPkcs7(t *testing.T) {
	const blockSize = 16

	t.Run("Pads", func(t *testing.T) {
		expected := []byte("1234567890\x06\x06\x06\x06\x06\x06")
		result, err := PadPKCS7([]byte("1234567890"), blockSize)
		require.NoError(t, err)
		assert.Equal(t, expected, result)
	})

	t.Run("Unpads", func(t *testing.T) {
		result, _ := UnpadPKCS7([]byte("1234567890\x06\x06\x06\x06\x06\x06"), blockSize)
		expected := []byte("1234567890")
		assert.Equal(t, expected, result)
	})

	t.Run("Handles long", func(t *testing.T) {
		longStr := []byte("123456789012345678901234567890123456789012345678901234567890")
		expected := []byte("123456789012345678901234567890123456789012345678901234567890\x04\x04\x04\x04")
		padded, err := PadPKCS7(longStr, blockSize)
		require.NoError(t, err)
		assert.Equal(t, expected, padded)
		if bytes.Equal(padded, expected) == false {
			panic(fmt.Sprintf(`Padding wrong - expected "%x" but got "%x"`, expected, padded))
		}

		unpadded, err := UnpadPKCS7(padded, blockSize)
		require.NoError(t, err)
		assert.Equal(t, longStr, unpadded)
	})

	t.Run("Handles short", func(t *testing.T) {
		shortStr := []byte("1")
		expected := []byte("1\x0f\x0f\x0f\x0f\x0f\x0f\x0f\x0f\x0f\x0f\x0f\x0f\x0f\x0f\x0f")
		padded, err := PadPKCS7(shortStr, blockSize)
		require.NoError(t, err)
		assert.Equal(t, expected, padded)

		unpadded, _ := UnpadPKCS7(padded, blockSize)
		assert.Equal(t, shortStr, unpadded)
	})

	t.Run("Handles empty", func(t *testing.T) {
		emptyStr := []byte("")
		expected := []byte("\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10")
		padded, err := PadPKCS7(emptyStr, blockSize)
		require.NoError(t, err)
		assert.Equal(t, expected, padded)

		unpadded, err := UnpadPKCS7(padded, blockSize)
		require.NoError(t, err)
		assert.Equal(t, emptyStr, unpadded)
	})

	t.Run("Handles block size", func(t *testing.T) {
		val := []byte("1234567890ABCDEF")
		expected := []byte("1234567890ABCDEF\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10\x10")
		padded, err := PadPKCS7(val, blockSize)
		require.NoError(t, err)
		assert.Equal(t, expected, padded)

		unpadded, err := UnpadPKCS7(padded, blockSize)
		require.NoError(t, err)
		assert.Equal(t, val, unpadded)
	})

	t.Run("Invalid length while unpadding", func(t *testing.T) {
		unpadded, err := UnpadPKCS7([]byte("1234567890\x06\x06\x06\x06"), blockSize)
		require.Error(t, err)
		require.ErrorIs(t, err, ErrInvalidPKCS7Padding)
		assert.Nil(t, unpadded)
	})

	t.Run("Invalid padding bytes", func(t *testing.T) {
		tests := [][]byte{
			[]byte("1234567890\x06\x06\x06\x06\x06\x07"),
			[]byte("1234567890\x06\x06\x07\x07\x06\x06"),
			[]byte("1234567890\x01\x06\x06\x06\x06\x06"),
			[]byte("1234567890\x0A\x0A\x0A\x0A\x0A\x0A"),
			[]byte("1234567890\xEE\xEE\xEE\xEE\xEE\xEE"),
		}
		for _, tt := range tests {
			unpadded, err := UnpadPKCS7(tt, blockSize)
			require.Error(t, err)
			require.ErrorIs(t, err, ErrInvalidPKCS7Padding)
			assert.Nil(t, unpadded)
		}
	})

	t.Run("Invalid block size", func(t *testing.T) {
		res, err := PadPKCS7([]byte("1234567890ABCDEF"), 260)
		require.Error(t, err)
		require.ErrorIs(t, err, ErrInvalidPKCS7BlockSize)
		assert.Nil(t, res)

		res, err = UnpadPKCS7([]byte("1234567890ABCDEF"), 260)
		require.Error(t, err)
		require.ErrorIs(t, err, ErrInvalidPKCS7BlockSize)
		assert.Nil(t, res)
	})

	t.Run("Unpad empty string", func(t *testing.T) {
		res, err := UnpadPKCS7([]byte{}, blockSize)
		require.NoError(t, err)
		assert.Empty(t, res)
	})
}
