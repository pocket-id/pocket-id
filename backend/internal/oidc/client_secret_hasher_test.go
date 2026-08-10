package oidc

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"github.com/pocket-id/pocket-id/backend/internal/model"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
)

func TestClientSecretHasherCompare(t *testing.T) {
	const value = "client-secret-value"

	bcryptHash, err := bcrypt.GenerateFromPassword([]byte(value), bcrypt.MinCost)
	require.NoError(t, err)

	for _, test := range []struct {
		name    string
		hash    string
		data    string
		wantErr bool
	}{
		{
			name: "matching SHA-256 hash",
			hash: "sha256:" + utils.CreateSha256Hash(value),
			data: value,
		},
		{
			name:    "SHA-256 hash of another secret",
			hash:    "sha256:" + utils.CreateSha256Hash(value),
			data:    "some-other-value",
			wantErr: true,
		},
		{
			name: "bcrypt hash migrated from the single-secret column",
			hash: "bcrypt:" + string(bcryptHash),
			data: value,
		},
		{
			name:    "bcrypt hash of another secret",
			hash:    "bcrypt:" + string(bcryptHash),
			data:    "some-other-value",
			wantErr: true,
		},
		{
			name:    "hash without an algorithm",
			hash:    utils.CreateSha256Hash(value),
			data:    value,
			wantErr: true,
		},
		{
			name:    "unknown algorithm",
			hash:    "md5:" + utils.CreateSha256Hash(value),
			data:    value,
			wantErr: true,
		},
		{
			name:    "empty hash, as returned for a client without secrets",
			hash:    "",
			data:    value,
			wantErr: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := clientSecretHasher{}.Compare(t.Context(), []byte(test.hash), []byte(test.data))
			if test.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestClientSecretHasherHash(t *testing.T) {
	const value = "client-secret-value"

	hash, err := clientSecretHasher{}.Hash(t.Context(), []byte(value))
	require.NoError(t, err)
	assert.Equal(t, "sha256:"+utils.CreateSha256Hash(value), string(hash))

	// A hash produced by the hasher is accepted by its own comparison
	err = clientSecretHasher{}.Compare(t.Context(), hash, []byte(value))
	require.NoError(t, err)
}

func TestClientSecretHasherAcceptsModelEncoding(t *testing.T) {
	const value = "client-secret-value"

	secret := model.OidcClientSecret{
		Algorithm: model.OidcClientSecretHashSHA256,
		Hash:      utils.CreateSha256Hash(value),
	}
	err := clientSecretHasher{}.Compare(t.Context(), secret.EncodedHash(), []byte(value))
	require.NoError(t, err)
}
