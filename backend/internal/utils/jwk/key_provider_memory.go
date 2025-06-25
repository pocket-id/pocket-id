package jwk

import (
	"github.com/lestrrat-go/jwx/v3/jwk"
)

type KeyProviderMemory struct{}

func (f *KeyProviderMemory) Init(opts KeyProviderOpts) error {
	return nil
}

func (f *KeyProviderMemory) LoadKey() (jwk.Key, error) {
	// No-op in this implementation
	// Returning nil key will cause the JWT service to generate a new key
	return nil, nil
}

func (f *KeyProviderMemory) SaveKey(key jwk.Key) error {
	// No-op in this implementation
	return nil
}

// Compile-time interface check
var _ KeyProvider = (*KeyProviderMemory)(nil)
