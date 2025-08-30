PRAGMA foreign_keys=OFF;
BEGIN;
ALTER TABLE api_keys
ADD COLUMN expiration_email_sent BOOLEAN NOT NULL DEFAULT 0;
COMMIT;
PRAGMA foreign_keys=ON;
