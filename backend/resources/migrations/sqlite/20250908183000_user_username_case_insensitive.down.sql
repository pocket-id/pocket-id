PRAGMA foreign_keys=OFF;
BEGIN;

-- Drop the functional unique index enforcing case-insensitive uniqueness
DROP INDEX IF EXISTS users_username_lower_unique;

COMMIT;
PRAGMA foreign_keys=ON;