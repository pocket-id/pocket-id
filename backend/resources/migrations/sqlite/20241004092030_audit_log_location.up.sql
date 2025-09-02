PRAGMA foreign_keys=OFF;
BEGIN;
ALTER TABLE audit_logs ADD COLUMN country TEXT;
ALTER TABLE audit_logs ADD COLUMN city TEXT;
COMMIT;
PRAGMA foreign_keys=ON;
