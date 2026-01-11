PRAGMA foreign_keys= OFF;
BEGIN;

DROP TABLE email_verification_tokens;
ALTER TABLE users DROP COLUMN email_verified;

COMMIT;
PRAGMA foreign_keys= ON;
