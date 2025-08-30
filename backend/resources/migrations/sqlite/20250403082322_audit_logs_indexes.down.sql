PRAGMA foreign_keys=OFF;
BEGIN;
DROP INDEX IF EXISTS idx_audit_logs_event;
DROP INDEX IF EXISTS idx_audit_logs_user_id;
DROP INDEX IF EXISTS idx_audit_logs_client_name;

COMMIT;
PRAGMA foreign_keys=ON;
