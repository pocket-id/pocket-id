//go:build unit

// This file is only imported by unit tests

package testing

import (
	"crypto/ecdsa"
	"encoding/json"
	"testing"

	"github.com/lestrrat-go/jwx/v4/jwa"
	"github.com/lestrrat-go/jwx/v4/jwk"
	"github.com/lestrrat-go/jwx/v4/jws"
	"github.com/stretchr/testify/require"
)

// DBSCProof signs a challenge with a device key like a browser does for Device Bound Session Credentials
// headerKey is the key the proof carries in its header, which a registration proof needs and a refresh proof leaves out, and it differs from key to forge a proof
func DBSCProof(t *testing.T, key *ecdsa.PrivateKey, challenge string, headerKey *ecdsa.PrivateKey) string {
	t.Helper()

	headers := jws.NewHeaders()
	require.NoError(t, headers.Set(jws.TypeKey, "dbsc+jwt"))
	if headerKey != nil {
		publicKey, err := jwk.PublicKeyOf(headerKey)
		require.NoError(t, err)
		require.NoError(t, headers.Set(jws.JWKKey, publicKey))
	}

	payload, err := json.Marshal(map[string]string{"jti": challenge})
	require.NoError(t, err)
	signed, err := jws.Sign(payload, jws.WithKey(jwa.ES256(), key, jws.WithProtectedHeaders(headers)))
	require.NoError(t, err)
	return string(signed)
}
