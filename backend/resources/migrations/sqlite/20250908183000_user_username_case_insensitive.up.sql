PRAGMA foreign_keys=OFF;
BEGIN;

/* Normalize all existing usernames to lowercase to enforce case-insensitive semantics */
UPDATE users
SET username = lower(username);

/* Enforce case-insensitive uniqueness at the DB level.
   SQLite honors collation in indexes; this unique index on lower(username) prevents
   duplicates that differ only by casing, regardless of the column's original UNIQUE. */
CREATE UNIQUE INDEX IF NOT EXISTS users_username_lower_unique
    ON users (lower(username));

COMMIT;
PRAGMA foreign_keys=ON;