package appconfig

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/italypaleale/francis/actor"

	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
)

// encryptedSecretPrefix marks the sensitive values that are encrypted in the actor state
const encryptedSecretPrefix = "encrypted:"

// encryptSecrets returns a copy of the config with its sensitive values encrypted with the ENCRYPTION_KEY
func (m *AppConfigModel) encryptSecrets() (*AppConfigModel, error) {
	return m.mapSecrets(func(value string) (string, error) {
		encrypted, err := datatype.EncryptedString(value).Value()
		if err != nil {
			return "", err
		}
		return encryptedSecretPrefix + fmt.Sprint(encrypted), nil
	})
}

// decryptSecrets returns a copy of the config with its sensitive values decrypted
func (m *AppConfigModel) decryptSecrets() (*AppConfigModel, error) {
	return m.mapSecrets(func(value string) (string, error) {
		var decrypted datatype.EncryptedString
		err := decrypted.Scan(strings.TrimPrefix(value, encryptedSecretPrefix))
		return string(decrypted), err
	})
}

// mapSecrets returns a copy of the config with fn applied to each sensitive value that is set
func (m *AppConfigModel) mapSecrets(fn func(value string) (string, error)) (*AppConfigModel, error) {
	clone := m.Clone()
	rv := reflect.ValueOf(clone).Elem()
	for i := range rv.NumField() {
		field := rv.Type().Field(i)
		if field.Tag.Get("sensitive") != "true" || rv.Field(i).String() == "" {
			continue
		}

		value, err := fn(rv.Field(i).String())
		if err != nil {
			return nil, fmt.Errorf("failed to process the value of %s: %w", field.Name, err)
		}
		rv.Field(i).SetString(value)
	}

	return clone, nil
}

// ReencryptSecrets returns the saved config with its sensitive values re-encrypted from oldKey to newKey, or nil if no config is saved yet
// The keys are the ones datatype.DeriveEncryptedStringKey derives from the old and the new ENCRYPTION_KEY
func ReencryptSecrets(ctx context.Context, store actor.Host, oldKey []byte, newKey []byte) (*AppConfigModel, error) {
	var state AppConfigModel
	err := store.GetState(ctx, AppConfigActorType, actor.SingletonActorID, &state)
	if errors.Is(err, actor.ErrStateNotFound) {
		return nil, nil
	} else if err != nil {
		return nil, fmt.Errorf("failed to load the app config: %w", err)
	}

	return state.mapSecrets(func(value string) (string, error) {
		decrypted, err := datatype.DecryptEncryptedStringWithKey(oldKey, strings.TrimPrefix(value, encryptedSecretPrefix))
		if err != nil {
			return "", err
		}
		encrypted, err := datatype.EncryptEncryptedStringWithKey(newKey, decrypted)
		if err != nil {
			return "", err
		}
		return encryptedSecretPrefix + encrypted, nil
	})
}
