PRAGMA foreign_keys=OFF;
BEGIN;
ALTER TABLE api_keys
DROP COLUMN expiration_email_sent;
COMMIT;
PRAGMA foreign_keys=ON;
