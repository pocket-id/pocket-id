PRAGMA foreign_keys=OFF;
BEGIN;
ALTER TABLE audit_logs DROP COLUMN country;
ALTER TABLE audit_logs DROP COLUMN city;
COMMIT;
PRAGMA foreign_keys=ON;
