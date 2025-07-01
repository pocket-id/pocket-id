ALTER TABLE audit_logs ALTER COLUMN ip_address SET NOT NULL;

DROP INDEX IF EXISTS idx_audit_logs_created_at;
DROP INDEX IF EXISTS idx_audit_logs_user_agent;
