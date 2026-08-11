package oidc

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"github.com/pocket-id/pocket-id/backend/internal/model"
)

// errUnsupportedHashAlgorithm is returned when a stored hash carries an algorithm this build cannot verify
var errUnsupportedHashAlgorithm = errors.New("unsupported client secret hash algorithm")

// clientSecretHasher verifies client secrets against the hashes stored in the client's credentials document
// Fosite hands it the encoded hashes produced by model.OidcClientSecret.EncodedHash, which carry the algorithm they were computed with
type clientSecretHasher struct{}

// Compare checks a presented client secret against a stored, algorithm-prefixed hash
func (clientSecretHasher) Compare(_ context.Context, hash []byte, data []byte) error {
	algorithm, encoded, found := strings.Cut(string(hash), ":")
	if !found {
		return errUnsupportedHashAlgorithm
	}

	switch model.OidcClientSecretHashAlgorithm(algorithm) {
	case model.OidcClientSecretHashSHA256:
		sum := sha256.Sum256(data)
		if subtle.ConstantTimeCompare([]byte(hex.EncodeToString(sum[:])), []byte(encoded)) != 1 {
			return errors.New("client secret does not match")
		}
		return nil
	case model.OidcClientSecretHashBcrypt:
		// Legacy hashes migrated from the single-secret column, which cannot be re-hashed because the secret's value is not recoverable
		return bcrypt.CompareHashAndPassword([]byte(encoded), data)
	default:
		return errUnsupportedHashAlgorithm
	}
}

// Hash is required by the fosite.Hasher interface but is never called, because Pocket ID hashes secrets when it stores them
func (clientSecretHasher) Hash(_ context.Context, data []byte) ([]byte, error) {
	sum := sha256.Sum256(data)
	return []byte(string(model.OidcClientSecretHashSHA256) + ":" + hex.EncodeToString(sum[:])), nil
}
