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
	"errors"
)

var (
	ErrInvalidPKCS7BlockSize = errors.New("pkcs7: invalid block size")
	ErrInvalidPKCS7Padding   = errors.New("pkcs7: incorrect padding")
)

// PadPKCS7 adds PKCS#7 padding to a message.
func PadPKCS7(buf []byte, size int) ([]byte, error) {
	if size <= 1 || size >= 256 {
		return nil, ErrInvalidPKCS7BlockSize
	}
	bufLen := len(buf)
	padLen := size - bufLen%size
	padding := bytes.Repeat([]byte{byte(padLen)}, padLen)
	return append(buf, padding...), nil
}

// UnpadPKCS7 removes PKCS#7 from a message.
func UnpadPKCS7(buf []byte, size int) ([]byte, error) {
	if size <= 1 || size >= 256 {
		return nil, ErrInvalidPKCS7BlockSize
	}
	l := len(buf)
	if l == 0 {
		return []byte{}, nil
	}
	if l%size != 0 {
		return nil, ErrInvalidPKCS7Padding
	}

	padLen := int(buf[l-1])
	if padLen <= 0 || padLen > size {
		return nil, ErrInvalidPKCS7Padding
	}
	padLenB := byte(padLen)
	for i := l - padLen; i < l; i++ {
		if buf[i] != padLenB {
			return nil, ErrInvalidPKCS7Padding
		}
	}
	return buf[:l-padLen], nil
}
