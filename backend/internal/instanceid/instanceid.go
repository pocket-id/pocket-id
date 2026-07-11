package instanceid

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Load the instance ID for the Pocket ID cluster from the "kv" table
// If no instance ID exists yet, a new one is generated and persisted atomically
func Load(parentCtx context.Context, db *gorm.DB) (string, error) {
	// Candidate value used only if there's no instance ID stored yet
	newInstanceID := uuid.NewString()

	// We use a raw query because gorm can't build it for us in this atomic way
	// The syntax is valid for both SQLite and Postgres
	var value string
	ctx, cancel := context.WithTimeout(parentCtx, 10*time.Second)
	defer cancel()
	err := db.
		WithContext(ctx).
		Raw(
			`INSERT INTO kv (key, value)
			VALUES ('instance_id', ?)
			ON CONFLICT (key) DO UPDATE SET
				value = CASE
					WHEN kv.value = '' THEN excluded.value
					ELSE kv.value
				END
			RETURNING value`,
			newInstanceID,
		).
		Scan(&value).
		Error
	if err != nil {
		return "", fmt.Errorf("failed to load instance ID from the database: %w", err)
	}

	// We should have an instance ID now
	// This case should never happen
	if value == "" {
		return "", errors.New("retrieved instance ID is unexpectedly empty")
	}

	return value, nil
}
