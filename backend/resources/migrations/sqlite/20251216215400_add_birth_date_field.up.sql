PRAGMA foreign_keys=OFF;
BEGIN;
ALTER TABLE users ADD COLUMN birth_date DATE;
COMMIT;
PRAGMA foreign_keys=ON;