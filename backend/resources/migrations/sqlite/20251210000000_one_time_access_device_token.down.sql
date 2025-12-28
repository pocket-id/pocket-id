PRAGMA foreign_keys=OFF;
BEGIN;

ALTER TABLE one_time_access_tokens DROP COLUMN device_token;

COMMIT;
PRAGMA foreign_keys=ON;
