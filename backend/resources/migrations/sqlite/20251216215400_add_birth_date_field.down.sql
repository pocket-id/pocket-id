PRAGMA foreign_keys=OFF;
BEGIN;
ALTER TABLE users DROP COLUMN birth_date;
COMMIT;
PRAGMA foreign_keys=ON;