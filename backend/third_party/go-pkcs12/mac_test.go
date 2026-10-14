// Copyright 2015, 2018, 2019 Opsmate, Inc. All rights reserved.
// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package pkcs12

import (
	"bytes"
	"crypto/x509/pkix"
	"encoding/asn1"
	"fmt"
	"testing"
)

func TestVerifyMac(t *testing.T) {
	td := macData{
		Mac: digestInfo{
			Digest: []byte{0x18, 0x20, 0x3d, 0xff, 0x1e, 0x16, 0xf4, 0x92, 0xf2, 0xaf, 0xc8, 0x91, 0xa9, 0xba, 0xd6, 0xca, 0x9d, 0xee, 0x51, 0x93},
		},
		MacSalt:    []byte{1, 2, 3, 4, 5, 6, 7, 8},
		Iterations: 2048,
	}

	message := []byte{11, 12, 13, 14, 15}
	password, _ := bmpStringZeroTerminated("")

	td.Mac.Algorithm.Algorithm = asn1.ObjectIdentifier([]int{1, 2, 3})
	err := verifyMac(&td, message, password)
	if _, ok := err.(NotImplementedError); !ok {
		t.Errorf("err: %v", err)
	}

	td.Mac.Algorithm.Algorithm = asn1.ObjectIdentifier([]int{1, 3, 14, 3, 2, 26})
	err = verifyMac(&td, message, password)
	if err != ErrIncorrectPassword {
		t.Errorf("Expected incorrect password, got err: %v", err)
	}

	password, _ = bmpStringZeroTerminated("Sesame open")
	err = verifyMac(&td, message, password)
	if err != nil {
		t.Errorf("err: %v", err)
	}

}

func TestComputeMac(t *testing.T) {
	td := macData{
		MacSalt:    []byte{1, 2, 3, 4, 5, 6, 7, 8},
		Iterations: 2048,
	}

	message := []byte{11, 12, 13, 14, 15}
	password, _ := bmpStringZeroTerminated("Sesame open")

	td.Mac.Algorithm.Algorithm = asn1.ObjectIdentifier([]int{1, 2, 3})
	err := computeMac(&td, message, password)
	if _, ok := err.(NotImplementedError); !ok {
		t.Errorf("err: %v", err)
	}

	td.Mac.Algorithm.Algorithm = asn1.ObjectIdentifier([]int{1, 3, 14, 3, 2, 26})
	err = computeMac(&td, message, password)
	if err != nil {
		t.Errorf("err: %v", err)
	}

	expectedDigest := []byte{0x18, 0x20, 0x3d, 0xff, 0x1e, 0x16, 0xf4, 0x92, 0xf2, 0xaf, 0xc8, 0x91, 0xa9, 0xba, 0xd6, 0xca, 0x9d, 0xee, 0x51, 0x93}

	if bytes.Compare(td.Mac.Digest, expectedDigest) != 0 {
		t.Errorf("Computed incorrect MAC; expected MAC to be '%d' but got '%d'", expectedDigest, td.Mac.Digest)
	}

}

func TestPBMAC1(t *testing.T) {
	// Create PBKDF2 parameters for PBMAC1
	kdfParams := pbkdf2Params{
		Salt:       asn1.RawValue{Tag: asn1.TagOctetString, Bytes: []byte{1, 2, 3, 4, 5, 6, 7, 8}},
		Iterations: 1000,
		KeyLength:  32,
		Prf:        pkix.AlgorithmIdentifier{Algorithm: oidHmacWithSHA512},
	}
	kdfParamsBytes, err := asn1.Marshal(kdfParams)
	if err != nil {
		t.Fatalf("Failed to marshal KDF params: %v", err)
	}

	// Create PBMAC1 parameters
	pbmac1Params := pbmac1Params{
		Kdf:    pkix.AlgorithmIdentifier{Algorithm: oidPBKDF2, Parameters: asn1.RawValue{FullBytes: kdfParamsBytes}},
		MacAlg: pkix.AlgorithmIdentifier{Algorithm: oidHmacWithSHA256},
	}
	pbmac1ParamsBytes, err := asn1.Marshal(pbmac1Params)
	if err != nil {
		t.Fatalf("Failed to marshal PBMAC1 params: %v", err)
	}

	// Create macData with PBMAC1 algorithm
	td := macData{
		Mac: digestInfo{
			Algorithm: pkix.AlgorithmIdentifier{
				Algorithm:  oidPBMAC1,
				Parameters: asn1.RawValue{FullBytes: pbmac1ParamsBytes},
			},
		},
		// MacSalt and Iterations should be ignored for PBMAC1
		MacSalt:    []byte{9, 10, 11, 12},
		Iterations: 999,
	}

	message := []byte{11, 12, 13, 14, 15}
	password, err := bmpStringZeroTerminated("test-password")
	if err != nil {
		t.Fatalf("Failed to encode password to BMP string: %v", err)
	}

	// Test MAC computation
	err = computeMac(&td, message, password)
	if err != nil {
		t.Errorf("Failed to compute PBMAC1: %v", err)
	}

	// Verify that MAC was computed
	if len(td.Mac.Digest) == 0 {
		t.Error("No MAC digest was computed")
	}

	if !bytes.Equal(td.Mac.Digest, []byte{0xff, 0x5c, 0x9f, 0x02, 0x8c, 0xdc, 0x21, 0xa1, 0xa1, 0x17, 0x12, 0xa8, 0xa0, 0xe4, 0xd4, 0x2d, 0xf8, 0xf6, 0x8b, 0xc5, 0xbd, 0xec, 0xe7, 0xde, 0xcf, 0xd9, 0x2e, 0x0c, 0x65, 0xcd, 0x1c, 0x6f}) {
		t.Error("Wrong MAC digest was computed")
	}

	// Test MAC verification
	err = verifyMac(&td, message, password)
	if err != nil {
		t.Errorf("Failed to verify PBMAC1: %v", err)
	}

	// Test with wrong password
	wrongPassword, err := bmpStringZeroTerminated("wrong-password")
	if err != nil {
		t.Fatalf("Failed to encode wrong password to BMP string: %v", err)
	}
	err = verifyMac(&td, message, wrongPassword)
	if err != ErrIncorrectPassword {
		t.Errorf("Expected ErrIncorrectPassword, got: %v", err)
	}
}

func TestPBMAC1RejectsShortKeyLength(t *testing.T) {
	message := []byte{11, 12, 13, 14, 15}
	password, err := bmpStringZeroTerminated("test-password")
	if err != nil {
		t.Fatalf("Failed to encode password to BMP string: %v", err)
	}

	// makeMacData builds a PBMAC1 macData whose PBKDF2 parameters request the
	// given derived key length.
	makeMacData := func(keyLength int) *macData {
		kdfParams := pbkdf2Params{
			Salt:       asn1.RawValue{Tag: asn1.TagOctetString, Bytes: []byte{1, 2, 3, 4, 5, 6, 7, 8}},
			Iterations: 1000,
			KeyLength:  keyLength,
			Prf:        pkix.AlgorithmIdentifier{Algorithm: oidHmacWithSHA256},
		}
		kdfParamsBytes, err := asn1.Marshal(kdfParams)
		if err != nil {
			t.Fatalf("Failed to marshal KDF params: %v", err)
		}
		params := pbmac1Params{
			Kdf:    pkix.AlgorithmIdentifier{Algorithm: oidPBKDF2, Parameters: asn1.RawValue{FullBytes: kdfParamsBytes}},
			MacAlg: pkix.AlgorithmIdentifier{Algorithm: oidHmacWithSHA256},
		}
		paramsBytes, err := asn1.Marshal(params)
		if err != nil {
			t.Fatalf("Failed to marshal PBMAC1 params: %v", err)
		}
		return &macData{
			Mac: digestInfo{
				Algorithm: pkix.AlgorithmIdentifier{
					Algorithm:  oidPBMAC1,
					Parameters: asn1.RawValue{FullBytes: paramsBytes},
				},
			},
		}
	}

	// RFC 9579 recommends rejecting key lengths shorter than 20 octets to
	// prevent MAC-forgery/authentication-bypass attacks (e.g. CVE-2026-34181).
	for _, keyLength := range []int{1, 8, 16, 19} {
		wantErr := fmt.Sprintf("pkcs12: PBMAC1 key length %d is too short to be secure (minimum 20 octets)", keyLength)
		if _, err := doMac(makeMacData(keyLength), message, password); err == nil || err.Error() != wantErr {
			t.Errorf("KeyLength %d: got error %v, want %q", keyLength, err, wantErr)
		}
	}

	// A key length of exactly 20 octets is the minimum allowed and must succeed.
	if _, err := doMac(makeMacData(20), message, password); err != nil {
		t.Errorf("KeyLength 20: unexpected error: %v", err)
	}
}

func TestPBMAC1RejectsLongKeyLength(t *testing.T) {
	message := []byte{11, 12, 13, 14, 15}
	password, err := bmpStringZeroTerminated("test-password")
	if err != nil {
		t.Fatalf("Failed to encode password to BMP string: %v", err)
	}

	// makeMacData builds a PBMAC1 macData whose PBKDF2 parameters request the
	// given derived key length.
	makeMacData := func(keyLength int) *macData {
		kdfParams := pbkdf2Params{
			Salt:       asn1.RawValue{Tag: asn1.TagOctetString, Bytes: []byte{1, 2, 3, 4, 5, 6, 7, 8}},
			Iterations: 1000,
			KeyLength:  keyLength,
			Prf:        pkix.AlgorithmIdentifier{Algorithm: oidHmacWithSHA256},
		}
		kdfParamsBytes, err := asn1.Marshal(kdfParams)
		if err != nil {
			t.Fatalf("Failed to marshal KDF params: %v", err)
		}
		params := pbmac1Params{
			Kdf:    pkix.AlgorithmIdentifier{Algorithm: oidPBKDF2, Parameters: asn1.RawValue{FullBytes: kdfParamsBytes}},
			MacAlg: pkix.AlgorithmIdentifier{Algorithm: oidHmacWithSHA256},
		}
		paramsBytes, err := asn1.Marshal(params)
		if err != nil {
			t.Fatalf("Failed to marshal PBMAC1 params: %v", err)
		}
		return &macData{
			Mac: digestInfo{
				Algorithm: pkix.AlgorithmIdentifier{
					Algorithm:  oidPBMAC1,
					Parameters: asn1.RawValue{FullBytes: paramsBytes},
				},
			},
		}
	}

	// A KeyLength larger than the maximum HMAC output size (64 octets, SHA-512) is rejected so it can't
	// cause a huge PBKDF2 allocation (same as OpenSSL's EVP_MAX_MD_SIZE cap)
	for _, keyLength := range []int{65, 128, 1 << 20, 1 << 30} {
		wantErr := fmt.Sprintf("pkcs12: PBMAC1 key length %d is too large (maximum 64 octets)", keyLength)
		if _, err := doMac(makeMacData(keyLength), message, password); err == nil || err.Error() != wantErr {
			t.Errorf("KeyLength %d: got error %v, want %q", keyLength, err, wantErr)
		}
	}

	// A key length of exactly 64 octets is the maximum allowed and must succeed.
	if _, err := doMac(makeMacData(64), message, password); err != nil {
		t.Errorf("KeyLength 64: unexpected error: %v", err)
	}
}

// TestPBMAC1ShortKeyAuthenticationBypass demonstrates the attack that the
// short-key check prevents: a forged trust store whose 1-octet PBMAC1 key is
// derived from one password can be "opened" with a different password whenever
// the two passwords' PBKDF2 outputs collide in that single octet (~1/256).
func TestPBMAC1ShortKeyAuthenticationBypass(t *testing.T) {
	// pbmac1-short-key-bypass.txt is a complete PKCS#12 trust store whose PBMAC1 MAC
	// uses a 1-octet PBKDF2 key. Its salt was chosen by brute force -- feasible only
	// because a 1-octet key has just 256 values -- so that the key derived from the
	// password used to compute the MAC ("fakepassword") collides with the key derived
	// from a different password ("realpassword"). The file therefore authenticates
	// under "realpassword" even though that password was never used to create it.
	//
	// A correct short-key check rejects the file outright; without it,
	// DecodeTrustStore authenticates the file under the wrong password, which is an
	// authentication bypass (cf. OpenSSL CVE-2026-34181).
	pfxData := loadTestData(t, "pbmac1-short-key-bypass.txt")

	_, err := DecodeTrustStore(pfxData, "realpassword")
	if err == nil {
		t.Fatal("authentication bypass: a 1-octet-key PBMAC1 trust store decoded under the wrong password")
	}
	// Guard against the file silently becoming undecodable for some unrelated reason,
	// which would make the assertion above vacuous: the only acceptable failure is the
	// short-key rejection. (The testdata file uses a 1-octet key.)
	if want := "pkcs12: PBMAC1 key length 1 is too short to be secure (minimum 20 octets)"; err.Error() != want {
		t.Fatalf("expected the short-key check to reject the file, got a different error: %v", err)
	}
}
