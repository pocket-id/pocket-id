//go:build unit

package email

import (
	"encoding/asn1"
	"testing"

	testutils "github.com/pocket-id/pocket-id/backend/internal/utils/testing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	pkcs12 "software.sslmate.com/src/go-pkcs12"
)

func indefiniteSigningEnvelope(t *testing.T, der []byte) []byte {
	t.Helper()
	var outer asn1.RawValue
	rest, err := asn1.Unmarshal(der, &outer)
	require.NoError(t, err)
	require.Empty(t, rest)
	var version, authSafe, mac asn1.RawValue
	remaining, err := asn1.Unmarshal(outer.Bytes, &version)
	require.NoError(t, err)
	remaining, err = asn1.Unmarshal(remaining, &authSafe)
	require.NoError(t, err)
	rest, err = asn1.Unmarshal(remaining, &mac)
	require.NoError(t, err)
	require.Empty(t, rest)
	// Change envelope lengths without changing the authenticated safe bytes covered by the MAC
	ber := append([]byte{0x30, 0x80}, version.FullBytes...)
	ber = append(ber, 0x30, 0x80)
	ber = append(ber, authSafe.Bytes...)
	ber = append(ber, 0, 0)
	ber = append(ber, mac.FullBytes...)
	return append(ber, 0, 0)
}

func TestSigningUploadBER(t *testing.T) {
	for _, algorithm := range []string{"RSA", "ECDSA"} {
		t.Run(algorithm, func(t *testing.T) {
			db := testutils.NewDatabaseForTest(t)
			module, err := New(db)
			require.NoError(t, err)
			key, cert, _ := signingFixture(t, algorithm)
			der, err := pkcs12.LegacyDES.WithIterations(102400).Encode(key, cert, nil, " password ")
			require.NoError(t, err)
			ber := indefiniteSigningEnvelope(t, der)
			var raw asn1.RawValue
			_, err = asn1.Unmarshal(ber, &raw)
			require.ErrorContains(t, err, "indefinite length")
			require.NoError(t, module.UploadSigningCertificate(t.Context(), ber, " password ", "sender@example.test"))
			require.NoError(t, module.SetSigningEnabled(t.Context(), true, "sender@example.test"))
			before, err := module.SigningStatus(t.Context(), "sender@example.test")
			require.NoError(t, err)
			require.Error(t, module.UploadSigningCertificate(t.Context(), ber, "wrong", "sender@example.test"))
			// Corrupt the MAC salt while keeping the BER envelope structurally valid
			corrupt := append([]byte(nil), ber...)
			corrupt[len(corrupt)-20] ^= 1
			require.Error(t, module.UploadSigningCertificate(t.Context(), corrupt, " password ", "sender@example.test"))
			// Removing the MAC must not turn a failed integrity check into an accepted import
			var outer, version, authSafe asn1.RawValue
			_, err = asn1.Unmarshal(der, &outer)
			require.NoError(t, err)
			remaining, err := asn1.Unmarshal(outer.Bytes, &version)
			require.NoError(t, err)
			_, err = asn1.Unmarshal(remaining, &authSafe)
			require.NoError(t, err)
			noMAC := append([]byte{0x30, 0x80}, version.FullBytes...)
			noMAC = append(noMAC, authSafe.FullBytes...)
			noMAC = append(noMAC, 0, 0)
			require.ErrorContains(t, module.UploadSigningCertificate(t.Context(), noMAC, " password ", "sender@example.test"), "no MAC")
			after, err := module.SigningStatus(t.Context(), "sender@example.test")
			require.NoError(t, err)
			assert.Equal(t, before, after)
		})
	}
}
