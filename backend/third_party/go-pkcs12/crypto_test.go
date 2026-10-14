// Copyright 2015, 2018, 2019 Opsmate, Inc. All rights reserved.
// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package pkcs12

import (
	"bytes"
	"crypto/x509/pkix"
	"encoding/asn1"
	"testing"
)

var sha1WithTripleDES = asn1.ObjectIdentifier([]int{1, 2, 840, 113549, 1, 12, 1, 3})

func TestPbDecrypterFor(t *testing.T) {
	params, _ := asn1.Marshal(pbeParams{
		Salt:       []byte{1, 2, 3, 4, 5, 6, 7, 8},
		Iterations: 2048,
	})
	alg := pkix.AlgorithmIdentifier{
		Algorithm: asn1.ObjectIdentifier([]int{1, 2, 3}),
		Parameters: asn1.RawValue{
			FullBytes: params,
		},
	}

	pass, _ := bmpStringZeroTerminated("Sesame open")

	_, _, err := pbDecrypterFor(alg, pass)
	if _, ok := err.(NotImplementedError); !ok {
		t.Errorf("expected not implemented error, got: %T %s", err, err)
	}

	alg.Algorithm = sha1WithTripleDES
	cbc, blockSize, err := pbDecrypterFor(alg, pass)
	if err != nil {
		t.Errorf("unexpected error from pbDecrypterFor %v", err)
	}
	if blockSize != 8 {
		t.Errorf("unexpected block size %d, wanted 8", blockSize)
	}

	plaintext := []byte{1, 2, 3, 4, 5, 6, 7, 8}
	expectedCiphertext := []byte{185, 73, 135, 249, 137, 1, 122, 247}
	ciphertext := make([]byte, len(plaintext))
	cbc.CryptBlocks(ciphertext, plaintext)

	if bytes.Compare(ciphertext, expectedCiphertext) != 0 {
		t.Errorf("bad ciphertext, got %x but wanted %x", ciphertext, expectedCiphertext)
	}
}

func TestPbEncrypterFor(t *testing.T) {
	params, _ := asn1.Marshal(pbeParams{
		Salt:       []byte{1, 2, 3, 4, 5, 6, 7, 8},
		Iterations: 2048,
	})
	alg := pkix.AlgorithmIdentifier{
		Algorithm: asn1.ObjectIdentifier([]int{1, 2, 3}),
		Parameters: asn1.RawValue{
			FullBytes: params,
		},
	}

	pass, _ := bmpStringZeroTerminated("Sesame open")

	_, _, err := pbEncrypterFor(alg, pass)
	if _, ok := err.(NotImplementedError); !ok {
		t.Errorf("expected not implemented error, got: %T %s", err, err)
	}

	alg.Algorithm = asn1.ObjectIdentifier([]int{1, 2, 840, 113549, 1, 12, 1, 3})
	cbc, _, err := pbEncrypterFor(alg, pass)
	if err != nil {
		t.Errorf("err: %v", err)
	}

	expectedM := []byte{1, 2, 3, 4, 5, 6, 7, 8}
	M := []byte{185, 73, 135, 249, 137, 1, 122, 247}
	cbc.CryptBlocks(M, M)

	if bytes.Compare(M, expectedM) != 0 {
		t.Errorf("expected M to be '%d', but found '%d", expectedM, M)
	}
}

var pbDecryptTests = []struct {
	in            []byte
	expected      []byte
	expectedError error
}{
	{
		[]byte("\x33\x73\xf3\x9f\xda\x49\xae\xfc\xa0\x9a\xdf\x5a\x58\xa0\xea\x46"), // 7 padding bytes
		[]byte("A secret!"),
		nil,
	},
	{
		[]byte("\x33\x73\xf3\x9f\xda\x49\xae\xfc\x96\x24\x2f\x71\x7e\x32\x3f\xe7"), // 8 padding bytes
		[]byte("A secret"),
		nil,
	},
	{
		[]byte("\x35\x0c\xc0\x8d\xab\xa9\x5d\x30\x7f\x9a\xec\x6a\xd8\x9b\x9c\xd9"), // 9 padding bytes, incorrect
		nil,
		ErrDecryption,
	},
	{
		[]byte("\xb2\xf9\x6e\x06\x60\xae\x20\xcf\x08\xa0\x7b\xd9\x6b\x20\xef\x41"), // incorrect padding bytes: [ ... 0x04 0x02 ]
		nil,
		ErrDecryption,
	},
}

func TestPbDecrypt(t *testing.T) {
	for i, test := range pbDecryptTests {
		decryptable := testDecryptable{
			data: test.in,
			algorithm: pkix.AlgorithmIdentifier{
				Algorithm: sha1WithTripleDES,
				Parameters: pbeParams{
					Salt:       []byte("\xf1\xf2\xf3\xf4\xf5\xf6\xf7\xf8"),
					Iterations: 4096,
				}.RawASN1(),
			},
		}
		password, _ := bmpStringZeroTerminated("sesame")

		plaintext, err := pbDecrypt(decryptable, password)
		if err != test.expectedError {
			t.Errorf("#%d: got error %q, but wanted %q", i, err, test.expectedError)
			continue
		}

		if !bytes.Equal(plaintext, test.expected) {
			t.Errorf("#%d: got %x, but wanted %x", i, plaintext, test.expected)
		}
	}
}

func TestPbEncrypt(t *testing.T) {
	tests := [][]byte{
		[]byte("A secret!"),
		[]byte("A secret"),
	}
	expected := [][]byte{
		[]byte("\x33\x73\xf3\x9f\xda\x49\xae\xfc\xa0\x9a\xdf\x5a\x58\xa0\xea\x46"), // 7 padding bytes
		[]byte("\x33\x73\xf3\x9f\xda\x49\xae\xfc\x96\x24\x2f\x71\x7e\x32\x3f\xe7"), // 8 padding bytes
	}

	for i, c := range tests {
		td := testDecryptable{
			algorithm: pkix.AlgorithmIdentifier{
				Algorithm: asn1.ObjectIdentifier([]int{1, 2, 840, 113549, 1, 12, 1, 3}), // SHA1/3TDES
				Parameters: pbeParams{
					Salt:       []byte("\xf1\xf2\xf3\xf4\xf5\xf6\xf7\xf8"),
					Iterations: 4096,
				}.RawASN1(),
			},
		}
		p, _ := bmpStringZeroTerminated("sesame")

		err := pbEncrypt(&td, c, p)
		if err != nil {
			t.Errorf("error encrypting %d: %v", c, err)
		}

		if bytes.Compare(td.data, expected[i]) != 0 {
			t.Errorf("expected %d to be encrypted to %d, but found %d", c, expected[i], td.data)
		}
	}
}

// pbes2AlgorithmWithIV builds a PBES2 (PBKDF2-HMAC-SHA256 + AES-256-CBC)
// AlgorithmIdentifier whose encryption-scheme IV is ivLen bytes long.
func pbes2AlgorithmWithIV(t *testing.T, ivLen int) pkix.AlgorithmIdentifier {
	t.Helper()

	var kdfparams pbkdf2Params
	saltBytes, err := asn1.Marshal([]byte("salt-salt"))
	if err != nil {
		t.Fatal(err)
	}
	kdfparams.Salt.FullBytes = saltBytes
	kdfparams.Iterations = 2048
	kdfparams.Prf.Algorithm = oidHmacWithSHA256

	var params pbes2Params
	params.Kdf.Algorithm = oidPBKDF2
	if params.Kdf.Parameters.FullBytes, err = asn1.Marshal(kdfparams); err != nil {
		t.Fatal(err)
	}
	params.EncryptionScheme.Algorithm = oidAES256CBC
	if params.EncryptionScheme.Parameters.FullBytes, err = asn1.Marshal(make([]byte, ivLen)); err != nil {
		t.Fatal(err)
	}

	alg := pkix.AlgorithmIdentifier{Algorithm: oidPBES2}
	if alg.Parameters.FullBytes, err = asn1.Marshal(params); err != nil {
		t.Fatal(err)
	}
	return alg
}

// A PBES2 encryption scheme can carry an IV whose length doesn't match the AES block size; make sure it's rejected.
func TestPbDecryptBadPBES2IVLength(t *testing.T) {
	password, _ := bmpStringZeroTerminated("")

	for _, ivLen := range []int{0, 4, 8, 15, 17, 32} {
		td := testDecryptable{
			data:      make([]byte, 16),
			algorithm: pbes2AlgorithmWithIV(t, ivLen),
		}
		_, err := pbDecrypt(&td, password)
		if err == nil {
			t.Errorf("ivLen=%d: expected an error, got nil", ivLen)
		}
	}

	// Sanity check: the correct (16-byte) IV length is still accepted by
	// pbDecrypterFor (it fails later, on padding, not on the IV).
	if _, _, err := pbDecrypterFor(pbes2AlgorithmWithIV(t, 16), password); err != nil {
		t.Errorf("ivLen=16: unexpected error %v", err)
	}
}

// Ensure PKCS#12 file with invalid IV length is rejected with an error.
func TestDecodeTrustStoreBadPBES2IVLength(t *testing.T) {
	var ed encryptedData
	ed.Version = 0
	ed.EncryptedContentInfo.ContentType = oidDataContentType
	ed.EncryptedContentInfo.ContentEncryptionAlgorithm = pbes2AlgorithmWithIV(t, 4)
	ed.EncryptedContentInfo.EncryptedContent = make([]byte, 16)

	var ci contentInfo
	ci.ContentType = oidEncryptedDataContentType
	ci.Content.Class = 2
	ci.Content.Tag = 0
	ci.Content.IsCompound = true
	ci.Content.Bytes, _ = asn1.Marshal(ed)

	authenticatedSafeBytes, _ := asn1.Marshal([]contentInfo{ci})

	var pfx pfxPdu
	pfx.Version = 3
	pfx.AuthSafe.ContentType = oidDataContentType
	pfx.AuthSafe.Content.Class = 2
	pfx.AuthSafe.Content.Tag = 0
	pfx.AuthSafe.Content.IsCompound = true
	pfx.AuthSafe.Content.Bytes, _ = asn1.Marshal(authenticatedSafeBytes)

	pfxData, err := asn1.Marshal(pfx)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := DecodeTrustStore(pfxData, ""); err == nil {
		t.Error("expected an error decoding a file with a malformed PBES2 IV, got nil")
	}
}

type testDecryptable struct {
	data      []byte
	algorithm pkix.AlgorithmIdentifier
}

func (d testDecryptable) Algorithm() pkix.AlgorithmIdentifier { return d.algorithm }
func (d testDecryptable) Data() []byte                        { return d.data }
func (d *testDecryptable) SetData(data []byte)                { d.data = data }

func (params pbeParams) RawASN1() (raw asn1.RawValue) {
	asn1Bytes, err := asn1.Marshal(params)
	if err != nil {
		panic(err)
	}
	_, err = asn1.Unmarshal(asn1Bytes, &raw)
	if err != nil {
		panic(err)
	}
	return
}

// The PBKDF2 PRF AlgorithmIdentifier must carry an explicit NULL parameter
// (RFC 8018 A.2 definition of PBKDF2-PRFs). BoringSSL rejects PKCS#12 files that omit it.
func TestMakePBES2ParametersPrfHasNullParameter(t *testing.T) {
	raw, err := makePBES2Parameters(bytes.NewReader(make([]byte, 64)), []byte{1, 2, 3, 4, 5, 6, 7, 8}, 2048)
	if err != nil {
		t.Fatal(err)
	}

	var params pbes2Params
	if err := unmarshal(raw, &params); err != nil {
		t.Fatal(err)
	}
	var kdfParams pbkdf2Params
	if err := unmarshal(params.Kdf.Parameters.FullBytes, &kdfParams); err != nil {
		t.Fatal(err)
	}

	if !kdfParams.Prf.Algorithm.Equal(oidHmacWithSHA256) {
		t.Fatalf("prf algorithm = %v, want %v", kdfParams.Prf.Algorithm, oidHmacWithSHA256)
	}
	if !bytes.Equal(kdfParams.Prf.Parameters.FullBytes, asn1.NullBytes) {
		t.Errorf("prf parameters = %x, want ASN.1 NULL (%x)", kdfParams.Prf.Parameters.FullBytes, asn1.NullBytes)
	}
}
