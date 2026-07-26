package usersignup

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/model"
)

// This file holds the one-time migration of the pre-actor signup tokens.
// The "actor tokens" migration freezes the signup_tokens table (and its user-group associations) into a JSON document stored in the "kv" table under the "signup_tokens_migrated" key.
// It's loaded here to create the per-token actors on first startup.

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

// migrateSignupTokens creates an actor for every signup token frozen into the kv table by the migration.
// It requires the actor state store to be available, so it must run after the actor host is ready.
// It is idempotent: tokens that have already been migrated are left untouched, so a token that has been used since it was migrated is never reset.
func (s *Service) migrateSignupTokens(ctx context.Context) error {
	migrated, err := loadMigratedSignupTokens(ctx, s.db)
	if err != nil {
		return err
	}
	if len(migrated) == 0 {
		return nil
	}

	var count int
	for _, m := range migrated {
		// Skip tokens that have already expired, since there would be nothing left to store
		expiresAt := time.Unix(m.ExpiresAt, 0)
		if !expiresAt.After(time.Now()) {
			continue
		}

		state := SignupTokenState{
			ID:           m.ID,
			ExpiresAt:    expiresAt,
			UsageLimit:   m.UsageLimit,
			UsageCount:   m.UsageCount,
			UserGroupIDs: m.UserGroupIDs,
			CreatedAt:    time.Unix(m.CreatedAt, 0),
		}

		// The token's value is the actor's ID
		// The "migrate" method only writes the state if the actor doesn't have one already
		_, err = s.actorService.Invoke(ctx, SignupTokenActorType, m.Token, signupTokenMethodMigrate, state)
		if err != nil {
			return fmt.Errorf("error migrating signup token '%s': %w", m.ID, err)
		}
		count++
	}

	slog.InfoContext(ctx, "Migrated signup tokens to actors", slog.Int("count", count))

	return nil
}

// loadMigratedSignupTokens reads the signup tokens frozen into the kv table by the migration
// It returns nil if there's nothing to migrate
func loadMigratedSignupTokens(ctx context.Context, db *gorm.DB) ([]migratedSignupToken, error) {
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

	return migrated, nil
}
