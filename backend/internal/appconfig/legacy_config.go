package appconfig

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/model"
)

// loadLegacyConfig loads the legacy config from the database
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
