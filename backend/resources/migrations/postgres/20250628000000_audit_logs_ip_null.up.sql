ALTER TABLE audit_logs ALTER COLUMN ip_address DROP NOT NULL;

-- Add missing indexes
CREATE INDEX idx_audit_logs_created_at ON audit_logs(created_at);
CREATE INDEX idx_audit_logs_user_agent ON audit_logs(user_agent);
