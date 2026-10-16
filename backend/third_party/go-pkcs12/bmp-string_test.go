// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package pkcs12

import (
	"bytes"
	"encoding/hex"
	"testing"
)

var bmpStringTests = []struct {
	in             string
	expectedHex    string
	zeroTerminated bool
	shouldFail     bool
}{
	{"", "0000", true, false},
	{"", "", false, false},
	// Example from https://tools.ietf.org/html/rfc7292#appendix-B.
	{"Beavis", "0042006500610076006900730000", true, false},
	{"Beavis", "004200650061007600690073", false, false},
	// Some characters from the "Letterlike Symbols Unicode block".
	{"\u2115 - Double-struck N", "21150020002d00200044006f00750062006c0065002d00730074007200750063006b0020004e0000", true, false},
	{"\u2115 - Double-struck N", "21150020002d00200044006f00750062006c0065002d00730074007200750063006b0020004e", false, false},
	// any character outside the BMP should trigger an error.
	{"\U0001f000 East wind (Mahjong)", "", true, true},
	{"\U0001f000 East wind (Mahjong)", "", false, true},
}

func TestBMPString(t *testing.T) {
	for i, test := range bmpStringTests {
		expected, err := hex.DecodeString(test.expectedHex)
		if err != nil {
			t.Fatalf("#%d: failed to decode expectation", i)
		}

		var out []byte

		if test.zeroTerminated {
			out, err = bmpStringZeroTerminated(test.in)
		} else {
			out, err = bmpString(test.in)
		}

		if err == nil && test.shouldFail {
			t.Errorf("#%d: expected to fail, but produced %x", i, out)
			continue
		}

		if err != nil && !test.shouldFail {
			t.Errorf("#%d: failed unexpectedly: %s", i, err)
			continue
		}

		if !test.shouldFail {
			if !bytes.Equal(out, expected) {
				t.Errorf("#%d: expected %s, got %x", i, test.expectedHex, out)
				continue
			}

			roundTrip, err := decodeBMPString(out)
			if err != nil {
				t.Errorf("#%d: decoding output gave an error: %s", i, err)
				continue
			}

			if roundTrip != test.in {
				t.Errorf("#%d: decoding output resulted in %q, but it should have been %q", i, roundTrip, test.in)
				continue
			}
		}
	}
}
