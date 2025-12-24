PRAGMA foreign_keys=OFF;
BEGIN;

ALTER TABLE one_time_access_tokens ADD COLUMN device_token TEXT;

COMMIT;
PRAGMA foreign_keys=ON;
