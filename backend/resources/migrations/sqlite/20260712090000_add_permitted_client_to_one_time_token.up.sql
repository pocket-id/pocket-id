PRAGMA foreign_keys=OFF;
BEGIN;

ALTER TABLE one_time_access_tokens ADD COLUMN permitted_client_id TEXT NOT NULL DEFAULT '';

COMMIT;
PRAGMA foreign_keys=ON;