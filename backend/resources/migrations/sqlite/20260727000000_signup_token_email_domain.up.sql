PRAGMA foreign_keys=OFF;
BEGIN;
ALTER TABLE signup_tokens ADD COLUMN email_domain TEXT;
COMMIT;
PRAGMA foreign_keys=ON;
