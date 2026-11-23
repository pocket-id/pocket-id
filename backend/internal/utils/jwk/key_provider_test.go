package jwk

import (
	"context"
	"errors"
	"testing"

	"github.com/lestrrat-go/jwx/v4/jwa"
	"github.com/lestrrat-go/jwx/v4/jwk"
	"github.com/stretchr/testify/require"
)

type fakeKeyProvider struct {
	stored    jwk.Key
	saveErr   error
	saveCalls int
}

func (p *fakeKeyProvider) Init(KeyProviderOpts) error { return nil }

func (p *fakeKeyProvider) LoadKey(context.Context) (jwk.Key, error) { return p.stored, nil }

func (p *fakeKeyProvider) SaveKey(_ context.Context, key jwk.Key) error {
	p.saveCalls++
	if p.saveErr != nil {
		return p.saveErr
	}
	p.stored = key
	return nil
}

func (p *fakeKeyProvider) ReplaceKey(context.Context, jwk.Key) error { return nil }

func generateTestKey() (jwk.Key, error) { return GenerateKey(jwa.ES256().String(), "") }

func TestLoadOrGenerateKey(t *testing.T) {
	t.Run("keeps a stored key", func(t *testing.T) {
		stored, err := generateTestKey()
		require.NoError(t, err)
		provider := &fakeKeyProvider{stored: stored}

		key, err := LoadOrGenerateKey(t.Context(), provider, generateTestKey)
		require.NoError(t, err)
		require.Equal(t, stored, key)
		require.Zero(t, provider.saveCalls)
	})

	t.Run("stops after three retries of a conflicting save", func(t *testing.T) {
		provider := &fakeKeyProvider{saveErr: ErrRetryKeyStorage}

		_, err := LoadOrGenerateKey(t.Context(), provider, generateTestKey)
		require.ErrorIs(t, err, ErrRetryKeyStorage)
		require.Equal(t, 4, provider.saveCalls)
	})

	t.Run("does not retry a failed database write", func(t *testing.T) {
		saveErr := errors.New("test database error")
		provider := &fakeKeyProvider{saveErr: saveErr}

		_, err := LoadOrGenerateKey(t.Context(), provider, generateTestKey)
		require.ErrorIs(t, err, saveErr)
		require.Equal(t, 1, provider.saveCalls)
	})
}
