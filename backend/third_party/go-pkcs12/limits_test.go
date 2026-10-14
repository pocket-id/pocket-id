package pkcs12

import (
	"crypto/rand"
	"crypto/x509/pkix"
	"encoding/asn1"
	"strings"
	"testing"
)

func TestKDFIterationLimits(t *testing.T) {
	for _, iterations := range []int{-1, 0, maxKDFIterations + 1, int(^uint(0) >> 1)} {
		legacy, err := asn1.Marshal(pbeParams{Salt: []byte("salt"), Iterations: iterations})
		if err != nil {
			t.Fatal(err)
		}
		modern, err := makePBES2Parameters(rand.Reader, []byte("salt"), iterations)
		if err != nil {
			t.Fatal(err)
		}
		pbmac, err := makePBMAC1Parameters([]byte("salt"), iterations)
		if err != nil {
			t.Fatal(err)
		}
		algorithms := []pkix.AlgorithmIdentifier{
			{Algorithm: oidPBEWithSHAAnd3KeyTripleDESCBC, Parameters: asn1.RawValue{FullBytes: legacy}},
			{Algorithm: oidPBES2, Parameters: asn1.RawValue{FullBytes: modern}},
		}
		password, err := bmpStringZeroTerminated("password")
		if err != nil {
			t.Fatal(err)
		}
		for _, algorithm := range algorithms {
			if _, _, err := pbeCipherFor(algorithm, password); err == nil || !strings.Contains(err.Error(), "iteration count") {
				t.Errorf("encryption KDF accepted invalid iterations %d: %v", iterations, err)
			}
		}
		for _, algorithm := range []pkix.AlgorithmIdentifier{
			{Algorithm: oidSHA1},
			{Algorithm: oidPBMAC1, Parameters: asn1.RawValue{FullBytes: pbmac}},
		} {
			data := macData{Mac: digestInfo{Algorithm: algorithm}, Iterations: iterations}
			if _, err := doMac(&data, []byte("message"), password); err == nil || !strings.Contains(err.Error(), "iteration count") {
				t.Errorf("MAC KDF accepted invalid iterations %d: %v", iterations, err)
			}
		}
	}
	for _, iterations := range []int{1, 51200, 102400, maxKDFIterations} {
		if err := checkKDFIterations(iterations); err != nil {
			t.Errorf("rejected supported iterations %d: %v", iterations, err)
		}
	}
}
