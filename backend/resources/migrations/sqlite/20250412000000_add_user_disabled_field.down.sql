PRAGMA foreign_keys=OFF;
BEGIN;
DROP INDEX idx_users_disabled;

ALTER TABLE users
DROP COLUMN disabled;
COMMIT;
PRAGMA foreign_keys=ON;
