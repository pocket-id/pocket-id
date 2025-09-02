PRAGMA foreign_keys=OFF;
BEGIN;
ALTER TABLE users ADD COLUMN locale TEXT;
COMMIT;
PRAGMA foreign_keys=ON;
