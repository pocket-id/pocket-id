package pkcs12

import (
	"bytes"
	"encoding/asn1"
	"encoding/base64"
	"errors"
	"testing"
)

func TestBERFraming(t *testing.T) {
	tests := []struct {
		name string
		ber  []byte
		der  []byte
	}{
		{"sequence", []byte{0x30, 0x80, 2, 1, 1, 0, 0}, []byte{0x30, 3, 2, 1, 1}},
		{"octet chunks", []byte{0x24, 0x80, 4, 1, 'a', 0x24, 0x80, 4, 1, 'b', 0, 0, 0, 0}, []byte{4, 2, 'a', 'b'}},
		{"opaque authenticated bytes", []byte{4, 4, 0x30, 0x80, 0, 0}, []byte{4, 4, 0x30, 0x80, 0, 0}},
		{"implicit encrypted octets", []byte{0x30, 0x80, 6, 1, 42, 0x30, 3, 6, 1, 42, 0xa0, 0x80, 4, 1, 'a', 4, 1, 'b', 0, 0, 0, 0}, []byte{0x30, 12, 6, 1, 42, 0x30, 3, 6, 1, 42, 0x80, 2, 'a', 'b'}},
		{"explicit octet wrapper", []byte{0xa0, 0x80, 0x24, 0x80, 4, 1, 'a', 0, 0, 0, 0}, []byte{0xa0, 3, 4, 1, 'a'}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := normalizeBER(test.ber)
			if err != nil || !bytes.Equal(got, test.der) {
				t.Fatalf("normalization got %x, %v; want %x", got, err, test.der)
			}
		})
	}
}

func TestBERRejectMalformed(t *testing.T) {
	inputs := [][]byte{
		nil, {0x30}, {0x30, 0x80}, {0x30, 0x80, 0}, {4, 0x80, 0, 0},
		{0x30, 2, 0, 0}, {4, 2, 1}, {4, 0x84, 0xff, 0xff, 0xff, 0xff},
		{0x24, 0x80, 2, 1, 1, 0, 0}, {4, 0, 4, 0}, {0x1f, 0x80, 0},
	}
	for _, input := range inputs {
		if _, err := normalizeBER(input); err == nil {
			t.Errorf("accepted malformed input %x", input)
		}
	}
	deep := []byte{4, 0}
	for i := 0; i < 34; i++ {
		deep = append(append([]byte{0x30, 0x80}, deep...), 0, 0)
	}
	if _, err := normalizeBER(deep); err == nil {
		t.Fatal("accepted excessive nesting")
	}
	if _, err := normalizeBER(make([]byte, (1<<20)+1)); err == nil {
		t.Fatal("accepted excessive input size")
	}
	many := append([]byte{0x30, 0x80}, bytes.Repeat([]byte{4, 0}, 65536)...)
	many = append(many, 0, 0)
	if _, err := normalizeBER(many); err == nil {
		t.Fatal("accepted excessive element count")
	}
}

func TestBERAuthenticatedSafePreservesMAC(t *testing.T) {
	for _, encoded := range testdata {
		der, err := base64.StdEncoding.DecodeString(encoded)
		if err != nil {
			t.Fatal(err)
		}
		var pfx pfxPdu
		if _, err := asn1.Unmarshal(der, &pfx); err != nil {
			t.Fatal(err)
		}
		var safe []byte
		if _, err := asn1.Unmarshal(pfx.AuthSafe.Content.Bytes, &safe); err != nil {
			t.Fatal(err)
		}
		var raw asn1.RawValue
		if _, err := asn1.Unmarshal(safe, &raw); err != nil {
			t.Fatal(err)
		}
		// Authenticate the original BER bytes, which differ from the normalized DER parsed later
		berSafe := append(append([]byte{0x30, 0x80}, raw.Bytes...), 0, 0)
		password, err := bmpStringZeroTerminated("")
		if err != nil {
			t.Fatal(err)
		}
		// Preserve upstream fixtures that use the alternate zero-byte encoding for an empty password
		if errors.Is(verifyMac(&pfx.MacData, safe, password), ErrIncorrectPassword) {
			password = nil
		}
		if err := computeMac(&pfx.MacData, berSafe, password); err != nil {
			t.Fatal(err)
		}
		first, _ := asn1.Marshal(berSafe[:len(berSafe)/2])
		second, _ := asn1.Marshal(berSafe[len(berSafe)/2:])
		chunks := append([]byte{0x24, 0x80}, first...)
		chunks = append(chunks, second...)
		chunks = append(chunks, 0, 0)
		pfx.AuthSafe.Content = asn1.RawValue{Class: 2, Tag: 0, IsCompound: true, Bytes: chunks}
		ber, err := asn1.Marshal(pfx)
		if err != nil {
			t.Fatal(err)
		}
		if _, _, _, err := DecodeChain(ber, ""); err != nil {
			t.Fatalf("BER authenticated safe import failed: %v", err)
		}
		if _, _, _, err := DecodeChain(ber, "wrong"); !errors.Is(err, ErrIncorrectPassword) {
			t.Fatalf("wrong password was not rejected: %v", err)
		}
		pfx.MacData.Mac.Digest[0] ^= 1
		corrupt, err := asn1.Marshal(pfx)
		if err != nil {
			t.Fatal(err)
		}
		if _, _, _, err := DecodeChain(corrupt, ""); !errors.Is(err, ErrIncorrectPassword) {
			t.Fatalf("corrupt MAC was not rejected: %v", err)
		}
	}
}

func FuzzBERNormalization(f *testing.F) {
	f.Add([]byte{0x30, 0x80, 4, 1, 'a', 0, 0})
	f.Add([]byte{0x24, 0x80, 4, 0, 0, 0})
	f.Fuzz(func(t *testing.T, input []byte) {
		der, err := normalizeBER(input)
		if err != nil {
			return
		}
		again, err := normalizeBER(der)
		if err != nil || !bytes.Equal(der, again) {
			t.Fatalf("normalization is not idempotent: %v", err)
		}
	})
}
