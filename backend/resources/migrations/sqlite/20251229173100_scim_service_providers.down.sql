PRAGMA foreign_keys=OFF;
BEGIN;

DROP TABLE scim_service_providers;
ALTER TABLE users DROP COLUMN updated_at;
ALTER TABLE user_groups DROP COLUMN updated_at;

COMMIT;
PRAGMA foreign_keys=ON;
