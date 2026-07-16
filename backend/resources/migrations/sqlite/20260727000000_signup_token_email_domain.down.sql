PRAGMA foreign_keys=OFF;
BEGIN;
ALTER TABLE signup_tokens DROP COLUMN email_domain;
COMMIT;
PRAGMA foreign_keys=ON;
