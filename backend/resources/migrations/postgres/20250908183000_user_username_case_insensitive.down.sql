BEGIN;

-- Drop the functional unique index enforcing case-insensitive uniqueness
DROP INDEX IF EXISTS users_username_lower_unique;

COMMIT;