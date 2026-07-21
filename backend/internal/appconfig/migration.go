package appconfig

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/italypaleale/go-kit/utils"
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/model"
)

// This file holds the one-time migration of the legacy (pre-actor) app config
// The legacy config was stored in the "config_migrated" key of the kv table, and it's loaded here to bootstrap the AppConfig actor on first startup

// LoadLegacyConfig loads the legacy config from the database
// This was migrated to the "config_migrated" key in the kv table
func LoadLegacyConfig(ctx context.Context, db *gorm.DB) (map[string]string, error) {
	// Retrieve the migrated config from the kv table
	row := model.KV{
		Key: "config_migrated",
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	err := db.WithContext(ctx).First(&row).Error
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		// There's no migrated config in the database, nothing to do
		return nil, nil
	case err != nil:
		return nil, fmt.Errorf("failed to load migrated config from the database: %w", err)
	case row.Value == nil || len(*row.Value) == 0:
		// Also no migrated config, nothing to do
		return nil, nil
	}

	// The value is a JSON-encoded dictionary
	res := map[string]string{}
	err = json.Unmarshal([]byte(*row.Value), &res)
	if err != nil {
		return nil, fmt.Errorf("error parsing migrated config: %w", err)
	}

	if len(res) == 0 {
		return nil, nil
	}
	return res, nil
}

// fromLegacyConfig builds an appConfigModel from a legacy config map
// The map's keys correspond to the "json" tags on appConfigModel, and all values are strings that are cast to each field's type
// Keys that are missing (or have an empty value) retain the default value
func fromLegacyConfig(legacyCfg map[string]string) (*AppConfigModel, error) {
	// Start from the default configuration, then override with the values from the legacy config
	dest := getDefaultConfig()

	rt := reflect.ValueOf(dest).Elem().Type()
	rv := reflect.ValueOf(dest).Elem()
	for i := range rt.NumField() {
		field := rt.Field(i)

		// Get the value of the json tag, taking only what's before the comma
		key, _, _ := strings.Cut(field.Tag.Get("json"), ",")

		// Look up the value in the legacy config
		// If the key is missing or the value is empty, we keep the default value
		value, ok := legacyCfg[key]
		if !ok || value == "" {
			continue
		}

		// Cast the string value to the field's type
		fv := rv.Field(i)
		switch fv.Kind() { //nolint:exhaustive
		case reflect.String:
			fv.SetString(value)
		case reflect.Bool:
			fv.SetBool(utils.IsTruthy(value))
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			n, err := strconv.ParseInt(value, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("failed to parse integer value for key '%s': %w", key, err)
			}
			fv.SetInt(n)
		default:
			return nil, fmt.Errorf("unsupported field type '%s' for key '%s'", fv.Kind(), key)
		}
	}

	return dest, nil
}
