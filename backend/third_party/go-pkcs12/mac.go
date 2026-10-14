// Copyright 2015, 2018, 2019 Opsmate, Inc. All rights reserved.
// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package pkcs12

import (
	"crypto/hmac"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"crypto/x509/pkix"
	"encoding/asn1"
	"errors"
	"fmt"
	"hash"

	"golang.org/x/crypto/pbkdf2"
)

type macData struct {
	Mac        digestInfo
	MacSalt    []byte
	Iterations int `asn1:"optional,default:1"`
}

// from PKCS#7:
type digestInfo struct {
	Algorithm pkix.AlgorithmIdentifier
	Digest    []byte
}

// PBMAC1 parameters structure from RFC 8018
// When using PBMAC1, the MAC parameters are derived from the algorithm's Parameters field
// and the macData.MacSalt and macData.Iterations fields are ignored.
type pbmac1Params struct {
	Kdf    pkix.AlgorithmIdentifier
	MacAlg pkix.AlgorithmIdentifier
}

func makePBMAC1Parameters(salt []byte, iterations int) ([]byte, error) {
	var err error

	var kdfparams pbkdf2Params
	if kdfparams.Salt.FullBytes, err = asn1.Marshal(salt); err != nil {
		return nil, err
	}
	kdfparams.Iterations = iterations
	kdfparams.KeyLength = 32
	kdfparams.Prf.Algorithm = oidHmacWithSHA256

	var params pbmac1Params
	params.Kdf.Algorithm = oidPBKDF2
	if params.Kdf.Parameters.FullBytes, err = asn1.Marshal(kdfparams); err != nil {
		return nil, err
	}
	params.MacAlg.Algorithm = oidHmacWithSHA256

	return asn1.Marshal(params)
}

var (
	oidSHA1   = asn1.ObjectIdentifier([]int{1, 3, 14, 3, 2, 26})
	oidSHA256 = asn1.ObjectIdentifier([]int{2, 16, 840, 1, 101, 3, 4, 2, 1})
	oidSHA512 = asn1.ObjectIdentifier([]int{2, 16, 840, 1, 101, 3, 4, 2, 3})
	oidPBMAC1 = asn1.ObjectIdentifier([]int{1, 2, 840, 113549, 1, 5, 14})
)

// doPBMAC1 handles PBMAC1 MAC computation using parameters from Algorithm.Parameters
// PBMAC1 (RFC 8018) uses PBKDF2 for key derivation and supports various HMAC algorithms.
// Unlike traditional PKCS#12 MAC algorithms, PBMAC1 gets all its parameters from
// the Algorithm.Parameters field, ignoring macData.MacSalt and macData.Iterations.
func doPBMAC1(algorithm pkix.AlgorithmIdentifier, message, password []byte) ([]byte, error) {
	var params pbmac1Params
	if err := unmarshal(algorithm.Parameters.FullBytes, &params); err != nil {
		return nil, fmt.Errorf("error decoding PBMAC1 parameters: %w", err)
	}

	// Only PBKDF2 is supported as KDF
	if !params.Kdf.Algorithm.Equal(oidPBKDF2) {
		return nil, NotImplementedError("PBMAC1 KDF algorithm " + params.Kdf.Algorithm.String() + " is not supported")
	}

	var kdfParams pbkdf2Params
	if err := unmarshal(params.Kdf.Parameters.FullBytes, &kdfParams); err != nil {
		return nil, err
	}

	if kdfParams.Salt.Tag != asn1.TagOctetString {
		return nil, NotImplementedError("only octet string salts are supported for PBMAC1/PBKDF2")
	}

	// Determine PRF function for PBKDF2
	var prf func() hash.Hash
	switch {
	case kdfParams.Prf.Algorithm.Equal(oidHmacWithSHA256):
		prf = sha256.New
	case kdfParams.Prf.Algorithm.Equal(oidHmacWithSHA512):
		prf = sha512.New
	case kdfParams.Prf.Algorithm.Equal(oidHmacWithSHA1):
		prf = sha1.New
	case kdfParams.Prf.Algorithm == nil:
		// Algorithm not specified; defaults to SHA1 according to ASN1 definition
		prf = sha1.New
	default:
		return nil, NotImplementedError("PBMAC1 PRF " + kdfParams.Prf.Algorithm.String() + " is not supported")
	}

	// Determine MAC algorithm
	var hFn func() hash.Hash
	switch {
	case params.MacAlg.Algorithm.Equal(oidHmacWithSHA1):
		hFn = sha1.New
	case params.MacAlg.Algorithm.Equal(oidHmacWithSHA256):
		hFn = sha256.New
	case params.MacAlg.Algorithm.Equal(oidHmacWithSHA512):
		hFn = sha512.New
	default:
		return nil, NotImplementedError("PBMAC1 MAC algorithm " + params.MacAlg.Algorithm.String() + " is not supported")
	}

	// KeyLength is mandatory in RFC 9579
	if kdfParams.KeyLength <= 0 {
		return nil, errors.New("pkcs12: PBMAC1 requires explicit KeyLength parameter in PBKDF2 parameters")
	}
	// RFC 9579 RECOMMENDS rejecting key lengths less than 20; this is necessary to prevent possible authentication bypass (e.g. OpenSSL's CVE-2026-34181)
	if kdfParams.KeyLength < 20 {
		return nil, fmt.Errorf("pkcs12: PBMAC1 key length %d is too short to be secure (minimum 20 octets)", kdfParams.KeyLength)
	}
	// RFC 9879 RECOMMENDS that the key length match the HMAC output size, which
	// is at most 64 octets (SHA-512).  Reject anything larger so that a huge
	// KeyLength can't cause a huge PBKDF2 allocation (matching OpenSSL's EVP_MAX_MD_SIZE cap)
	const maxKeyLength = 64
	if kdfParams.KeyLength > maxKeyLength {
		return nil, fmt.Errorf("pkcs12: PBMAC1 key length %d is too large (maximum %d octets)", kdfParams.KeyLength, maxKeyLength)
	}
	keyLen := kdfParams.KeyLength

	// Derive key using PBKDF2
	key := pbkdf2.Key(password, kdfParams.Salt.Bytes, kdfParams.Iterations, keyLen, prf)

	// Compute HMAC
	mac := hmac.New(hFn, key)
	mac.Write(message)
	return mac.Sum(nil), nil
}

func doMac(macData *macData, message, password []byte) ([]byte, error) {
	// Handle PBMAC1 separately - it uses its own parameters structure from Algorithm.Parameters
	// and ignores macData.MacSalt and macData.Iterations fields
	if macData.Mac.Algorithm.Algorithm.Equal(oidPBMAC1) {
		// PBMAC1 expects UTF-8 passwords (for compatibility; see Erratum 7974), but
		// PKCS#12 passwords are BMP strings, so we convert the BMP string back to UTF-8
		originalPassword, err := decodeBMPString(password)
		if err != nil {
			return nil, err
		}
		utf8Password := []byte(originalPassword)
		return doPBMAC1(macData.Mac.Algorithm, message, utf8Password)
	}

	var hFn func() hash.Hash
	var key []byte
	switch {
	case macData.Mac.Algorithm.Algorithm.Equal(oidSHA1):
		hFn = sha1.New
		key = pbkdf(sha1Sum, 20, 64, macData.MacSalt, password, macData.Iterations, 3, 20)
	case macData.Mac.Algorithm.Algorithm.Equal(oidSHA256):
		hFn = sha256.New
		key = pbkdf(sha256Sum, 32, 64, macData.MacSalt, password, macData.Iterations, 3, 32)
	case macData.Mac.Algorithm.Algorithm.Equal(oidSHA512):
		hFn = sha512.New
		key = pbkdf(sha512Sum, 64, 128, macData.MacSalt, password, macData.Iterations, 3, 64)
	default:
		return nil, NotImplementedError("MAC digest algorithm not supported: " + macData.Mac.Algorithm.Algorithm.String())
	}

	mac := hmac.New(hFn, key)
	mac.Write(message)
	return mac.Sum(nil), nil
}

func verifyMac(macData *macData, message, password []byte) error {
	expectedMAC, err := doMac(macData, message, password)
	if err != nil {
		return err
	}
	if !hmac.Equal(macData.Mac.Digest, expectedMAC) {
		return ErrIncorrectPassword
	}
	return nil
}

func computeMac(macData *macData, message, password []byte) error {
	digest, err := doMac(macData, message, password)
	if err != nil {
		return err
	}
	macData.Mac.Digest = digest
	return nil
}
