PRAGMA foreign_keys=OFF;
BEGIN;
ALTER TABLE users DROP COLUMN locale;
COMMIT;
PRAGMA foreign_keys=ON;
