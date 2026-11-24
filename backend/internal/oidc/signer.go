package oidc

import (
	"errors"

	jose "github.com/go-jose/go-jose/v4"
)

// SigningKeyFromSigner wraps the raw signing key in a *jose.JSONWebKey that carries the
// key's algorithm.
//
// fosite's DefaultSigner picks the JOSE algorithm from the Go key TYPE alone: every
// *rsa.PrivateKey is signed with RS256 and every *ecdsa.PrivateKey with ES256, while an
// ed25519.PrivateKey is not supported at all.
func SigningKeyFromSigner(signer TokenSigner) (*jose.JSONWebKey, error) {
	rawKey := signer.GetPrivateKey()
	if rawKey == nil {
		return nil, errors.New("signing key is not available")
	}

	alg, err := signer.GetKeyAlg()
	if err != nil {
		return nil, err
	}

	signingKey := &jose.JSONWebKey{
		Key:       rawKey,
		Algorithm: alg.String(),
	}

	if keyID, ok := signer.GetKeyID(); ok {
		signingKey.KeyID = keyID
	}

	return signingKey, nil
}
