PRAGMA foreign_keys=OFF;
BEGIN;

ALTER TABLE one_time_access_tokens DROP COLUMN permitted_client_id;

COMMIT;
PRAGMA foreign_keys=ON;