// Package haconfig enforces that the HA mode setting is fixed for the lifetime of a Pocket ID cluster.
package haconfig

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"gorm.io/gorm"
)

// KVKey is the key in the "kv" table under which the cluster's HA mode setting is recorded.
const KVKey = "ha_enabled"

// Check verifies that the HA mode setting has not changed since the Pocket ID cluster was created.
//
// On the first startup it atomically records the current value in the "kv" table; on every subsequent startup it reads back the stored value and compares it with the configured one, returning an error if they differ.
//
// The setting is immutable because it changes how replicas coordinate against the shared database, so switching it under an existing cluster would be unsafe. To change it, the data must be exported and re-imported into a new database.
func Check(parentCtx context.Context, db *gorm.DB, haEnabled bool) error {
	desired := strconv.FormatBool(haEnabled)

	ctx, cancel := context.WithTimeout(parentCtx, 10*time.Second)
	defer cancel()

	// Atomically insert the current value if the row does not exist yet, then read back whatever is stored.
	// On a conflict the existing value is kept: the no-op self-update (value = kv.value) lets RETURNING return the current value, so the value fixed when the cluster was created is never overwritten here.
	// This is the same portable "INSERT ... ON CONFLICT ... RETURNING" pattern used for the instance ID, and is valid for both SQLite and Postgres.
	var stored string
	err := db.
		WithContext(ctx).
		Raw(
			`INSERT INTO kv (key, value)
			VALUES ('ha_enabled', ?)
			ON CONFLICT (key) DO UPDATE SET
				value = kv.value
			RETURNING value`,
			desired,
		).
		Scan(&stored).
		Error
	if err != nil {
		return fmt.Errorf("failed to load the HA mode setting from the database: %w", err)
	}

	if stored != desired {
		return fmt.Errorf("the HA mode setting cannot be changed after a Pocket ID cluster has been created: the database was created with HAEnabled=%s, but the current configuration has HAEnabled=%s; to change this setting, export your data from Pocket ID and re-import it into a new database with the new value", stored, desired)
	}

	return nil
}
