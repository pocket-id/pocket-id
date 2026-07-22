package usersignup

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/model"
)

// This file holds the one-time migration of the pre-actor signup tokens.
// The "move tokens to actor state" migration freezes the signup_tokens table (and its user-group associations) into a JSON document stored in the "kv" table under the "signup_tokens_migrated" key.
// It's loaded here to seed the singleton signup token actor on first startup.

// signupTokensMigratedKey is the kv key under which the pre-actor signup tokens were frozen.
const signupTokensMigratedKey = "signup_tokens_migrated" //nolint:gosec // G101 false positive: this is the name of a kv key, not a credential

// migratedSignupToken is the JSON shape of a signup token frozen into the kv table by the migration.
// All timestamps are expressed as Unix seconds.
type migratedSignupToken struct {
	ID           string   `json:"id"`
	Token        string   `json:"token"`
	ExpiresAt    int64    `json:"expiresAt"`
	UsageLimit   int      `json:"usageLimit"`
	UsageCount   int      `json:"usageCount"`
	UserGroupIDs []string `json:"userGroupIds"`
	CreatedAt    int64    `json:"createdAt"`
}

// loadMigratedSignupTokens reads the signup tokens frozen into the kv table by the migration, so the singleton actor can seed its state from them on first startup
// It returns nil if there's nothing to migrate
func loadMigratedSignupTokens(ctx context.Context, db *gorm.DB) ([]storedSignupToken, error) {
	row := model.KV{
		Key: signupTokensMigratedKey,
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	err := db.WithContext(ctx).First(&row).Error
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		// There are no migrated signup tokens in the database, nothing to do
		return nil, nil
	case err != nil:
		return nil, fmt.Errorf("failed to load migrated signup tokens from the database: %w", err)
	case row.Value == nil || len(*row.Value) == 0:
		// Also no migrated signup tokens, nothing to do
		return nil, nil
	}

	var migrated []migratedSignupToken
	err = json.Unmarshal([]byte(*row.Value), &migrated)
	if err != nil {
		return nil, fmt.Errorf("error parsing migrated signup tokens: %w", err)
	}

	tokens := make([]storedSignupToken, len(migrated))
	for i, m := range migrated {
		tokens[i] = storedSignupToken{
			ID:           m.ID,
			Token:        m.Token,
			ExpiresAt:    time.Unix(m.ExpiresAt, 0),
			UsageLimit:   m.UsageLimit,
			UsageCount:   m.UsageCount,
			UserGroupIDs: m.UserGroupIDs,
			CreatedAt:    time.Unix(m.CreatedAt, 0),
		}
	}

	return tokens, nil
}
