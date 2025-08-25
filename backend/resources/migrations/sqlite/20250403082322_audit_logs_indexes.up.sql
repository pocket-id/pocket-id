PRAGMA foreign_keys=OFF;
BEGIN;
CREATE INDEX idx_audit_logs_event ON audit_logs(event);
CREATE INDEX idx_audit_logs_user_id ON audit_logs(user_id);
CREATE INDEX idx_audit_logs_client_name ON audit_logs((json_extract(data, '$.clientName')));

COMMIT;
PRAGMA foreign_keys=ON;
