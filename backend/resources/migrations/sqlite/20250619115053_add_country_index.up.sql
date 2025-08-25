PRAGMA foreign_keys=OFF;
BEGIN;
CREATE INDEX idx_audit_logs_country ON audit_logs(country);
COMMIT;
PRAGMA foreign_keys=ON;
